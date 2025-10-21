package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/edgelink/backend/cmd/alert-service/internal/checker"
	"github.com/edgelink/backend/cmd/alert-service/internal/deduplication"
	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AlertGenerator 告警生成器
type AlertGenerator struct {
	alertRepo  repository.AlertRepository
	deviceRepo repository.DeviceRepository
	dedupeManager *deduplication.Manager
	logger     *zap.Logger
}

// NewAlertGenerator 创建告警生成器
func NewAlertGenerator(
	alertRepo repository.AlertRepository,
	deviceRepo repository.DeviceRepository,
	dedupeManager *deduplication.Manager,
	logger *zap.Logger,
) *AlertGenerator {
	return &AlertGenerator{
		alertRepo:  alertRepo,
		deviceRepo: deviceRepo,
		dedupeManager: dedupeManager,
		logger:     logger,
	}
}

// GenerateAlert 根据健康问题生成告警
func (ag *AlertGenerator) GenerateAlert(ctx context.Context, issue checker.HealthIssue) (*domain.Alert, error) {
	// 解析设备ID
	deviceID, err := uuid.Parse(issue.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device ID: %w", err)
	}

	// 确定告警类型
	alertType := ag.mapIssueTypeToAlertType(issue.Type)

	// 确定严重程度
	severity := ag.mapSeverity(issue.Severity)

	// 构造去重键
	dedupeKey := deduplication.AlertKey{
		DeviceID:  issue.DeviceID,
		AlertType: string(alertType),
	}

	// 检查是否在静默期内
	inSilent, err := ag.dedupeManager.IsInSilentPeriod(ctx, dedupeKey)
	if err != nil {
		ag.logger.Warn("Failed to check silent period", zap.Error(err))
	} else if inSilent {
		ag.logger.Debug("Alert is in silent period, skipping",
			zap.String("device_id", issue.DeviceID),
			zap.String("alert_type", string(alertType)),
		)
		return nil, nil
	}

	// 检查是否应该创建新告警
	shouldCreate, existingAlertID, shouldEscalate, err := ag.dedupeManager.ShouldCreateAlert(
		ctx,
		dedupeKey,
		string(severity),
	)
	if err != nil {
		ag.logger.Error("Failed to check deduplication", zap.Error(err))
		// 失败时仍然创建告警，确保不丢失
		shouldCreate = true
	}

	now := time.Now()

	if shouldCreate {
		// 创建新告警
		return ag.createNewAlert(ctx, deviceID, alertType, severity, issue, dedupeKey, now)
	}

	// 更新现有告警
	return ag.updateExistingAlert(ctx, *existingAlertID, dedupeKey, shouldEscalate, severity, issue, now)
}

// createNewAlert 创建新告警
func (ag *AlertGenerator) createNewAlert(
	ctx context.Context,
	deviceID uuid.UUID,
	alertType domain.AlertType,
	severity domain.Severity,
	issue checker.HealthIssue,
	dedupeKey deduplication.AlertKey,
	now time.Time,
) (*domain.Alert, error) {
	// 生成告警标题和消息
	title, message := ag.generateAlertContent(issue)

	// 序列化元数据
	metadataJSON, err := json.Marshal(issue.Metadata)
	if err != nil {
		ag.logger.Warn("Failed to marshal metadata", zap.Error(err))
		metadataJSON = []byte("{}")
	}

	var metadata domain.JSONB
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		metadata = domain.JSONB{}
	}

	// 创建告警实体
	alert := &domain.Alert{
		ID:              uuid.New(),
		DeviceID:        &deviceID,
		Severity:        severity,
		Type:            alertType,
		Title:           title,
		Message:         message,
		Status:          domain.AlertStatusActive,
		Metadata:        metadata,
		OccurrenceCount: 1,
		FirstSeenAt:     now,
		LastSeenAt:      now,
		CreatedAt:       now,
	}

	// 保存告警到数据库
	if err := ag.alertRepo.Create(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	// 记录去重信息
	if err := ag.dedupeManager.RecordAlert(ctx, alert.ID, dedupeKey, string(severity), true); err != nil {
		ag.logger.Error("Failed to record dedupe info", zap.Error(err))
	}

	ag.logger.Info("New alert created",
		zap.String("alert_id", alert.ID.String()),
		zap.String("type", string(alertType)),
		zap.String("severity", string(severity)),
		zap.String("device_id", deviceID.String()),
	)

	return alert, nil
}

// updateExistingAlert 更新现有告警
func (ag *AlertGenerator) updateExistingAlert(
	ctx context.Context,
	alertID uuid.UUID,
	dedupeKey deduplication.AlertKey,
	shouldEscalate bool,
	severity domain.Severity,
	issue checker.HealthIssue,
	now time.Time,
) (*domain.Alert, error) {
	// 获取当前出现次数
	occurrenceCount, err := ag.dedupeManager.GetOccurrenceCount(ctx, dedupeKey)
	if err != nil {
		ag.logger.Error("Failed to get occurrence count", zap.Error(err))
		occurrenceCount = 1
	}
	occurrenceCount++

	// 更新告警出现次数
	if err := ag.alertRepo.UpdateOccurrence(ctx, alertID, occurrenceCount); err != nil {
		return nil, fmt.Errorf("failed to update occurrence: %w", err)
	}

	// 记录去重信息
	if err := ag.dedupeManager.RecordAlert(ctx, alertID, dedupeKey, string(severity), false); err != nil {
		ag.logger.Error("Failed to update dedupe info", zap.Error(err))
	}

	ag.logger.Info("Alert occurrence updated",
		zap.String("alert_id", alertID.String()),
		zap.Int("occurrence_count", occurrenceCount),
	)

	// 检查是否需要升级严重程度
	if shouldEscalate {
		newSeverity := ag.escalateSeverity(severity)
		if err := ag.alertRepo.EscalateSeverity(ctx, alertID, newSeverity); err != nil {
			ag.logger.Error("Failed to escalate severity", zap.Error(err))
		} else {
			// 标记已升级
			if err := ag.dedupeManager.MarkEscalated(ctx, dedupeKey); err != nil {
				ag.logger.Error("Failed to mark escalated", zap.Error(err))
			}

			ag.logger.Warn("Alert severity escalated",
				zap.String("alert_id", alertID.String()),
				zap.String("old_severity", string(severity)),
				zap.String("new_severity", string(newSeverity)),
				zap.Int("occurrence_count", occurrenceCount),
			)
		}
	}

	// 获取更新后的告警
	alert, err := ag.alertRepo.FindByID(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated alert: %w", err)
	}

	return alert, nil
}

// ResolveDeviceAlerts 自动解决设备告警（当设备恢复在线时）
func (ag *AlertGenerator) ResolveDeviceAlerts(ctx context.Context, deviceID uuid.UUID, alertType domain.AlertType) error {
	// 解决数据库中的告警
	if err := ag.alertRepo.ResolveByDeviceAndType(ctx, deviceID, alertType); err != nil {
		return fmt.Errorf("failed to resolve alerts: %w", err)
	}

	// 移除去重信息
	dedupeKey := deduplication.AlertKey{
		DeviceID:  deviceID.String(),
		AlertType: string(alertType),
	}

	if err := ag.dedupeManager.RemoveDedupeInfo(ctx, dedupeKey); err != nil {
		ag.logger.Error("Failed to remove dedupe info", zap.Error(err))
	}

	// 设置静默期
	if err := ag.dedupeManager.SetSilentPeriod(ctx, dedupeKey); err != nil {
		ag.logger.Error("Failed to set silent period", zap.Error(err))
	}

	ag.logger.Info("Device alerts auto-resolved",
		zap.String("device_id", deviceID.String()),
		zap.String("alert_type", string(alertType)),
	)

	return nil
}

// escalateSeverity 提升严重程度
func (ag *AlertGenerator) escalateSeverity(current domain.Severity) domain.Severity {
	switch current {
	case domain.SeverityLow:
		return domain.SeverityMedium
	case domain.SeverityMedium:
		return domain.SeverityHigh
	case domain.SeverityHigh:
		return domain.SeverityCritical
	default:
		return current
	}
}

// mapIssueTypeToAlertType 映射问题类型到告警类型
func (ag *AlertGenerator) mapIssueTypeToAlertType(issueType string) domain.AlertType {
	switch issueType {
	case "device_offline":
		return domain.AlertTypeDeviceOffline
	case "high_latency":
		return domain.AlertTypeHighLatency
	case "connection_failed":
		return domain.AlertTypeTunnelFailure
	default:
		return domain.AlertTypeTunnelFailure
	}
}

// mapSeverity 映射严重程度
func (ag *AlertGenerator) mapSeverity(severity string) domain.Severity {
	switch severity {
	case "critical":
		return domain.SeverityCritical
	case "high":
		return domain.SeverityHigh
	case "medium":
		return domain.SeverityMedium
	case "low":
		return domain.SeverityLow
	default:
		return domain.SeverityMedium
	}
}

// generateAlertContent 生成告警标题和消息
func (ag *AlertGenerator) generateAlertContent(issue checker.HealthIssue) (string, string) {
	var title, message string

	switch issue.Type {
	case "device_offline":
		deviceName := "Unknown"
		if name, ok := issue.Metadata["device_name"].(string); ok {
			deviceName = name
		}

		title = fmt.Sprintf("设备离线: %s", deviceName)
		message = fmt.Sprintf("设备 %s 已离线超过阈值时间。请检查设备连接状态。", deviceName)

		if duration, ok := issue.Metadata["offline_duration"].(string); ok {
			message += fmt.Sprintf(" 离线时长: %s", duration)
		}

	case "high_latency":
		title = "检测到高延迟"
		message = "会话出现高延迟,可能影响用户体验。"

		if latency, ok := issue.Metadata["avg_latency_ms"].(int); ok {
			message += fmt.Sprintf(" 平均延迟: %dms", latency)
		}

	case "connection_failed":
		title = "连接失败"
		message = "设备连接建立失败。请检查网络配置和NAT穿透设置。"

	default:
		title = "系统告警"
		message = issue.Message
	}

	return title, message
}
