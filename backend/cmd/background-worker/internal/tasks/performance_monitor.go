package tasks

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PerformanceMonitorTask 性能监控任务
type PerformanceMonitorTask struct {
	sessionRepo repository.SessionRepository
	alertRepo   repository.AlertRepository
	logger      *zap.Logger
}

// NewPerformanceMonitorTask 创建性能监控任务
func NewPerformanceMonitorTask(
	sessionRepo repository.SessionRepository,
	alertRepo repository.AlertRepository,
	logger *zap.Logger,
) *PerformanceMonitorTask {
	return &PerformanceMonitorTask{
		sessionRepo: sessionRepo,
		alertRepo:   alertRepo,
		logger:      logger,
	}
}

// Run 执行性能监控 (检查延迟p95)
func (t *PerformanceMonitorTask) Run(ctx context.Context) error {
	t.logger.Info("Running performance monitor task")

	// 查询活跃会话
	sessions, err := t.sessionRepo.FindActiveSessions(ctx, 1000)
	if err != nil {
		return err
	}

	latencyThreshold := 500 // ms
	alertsCreated := 0
	now := time.Now()

	for _, session := range sessions {
		// 检查平均延迟
		if session.AvgLatencyMs == nil {
			continue
		}

		avgLatency := *session.AvgLatencyMs

		// 如果延迟超过阈值
		if avgLatency > latencyThreshold {
			// 检查是否已存在未解决的高延迟告警
			existingAlert, err := t.checkExistingAlert(ctx, session.DeviceAID, domain.AlertTypeHighLatency)
			if err != nil {
				t.logger.Error("Failed to check existing alert",
					zap.Error(err),
					zap.String("session_id", session.ID.String()),
				)
				continue
			}

			// 如果已存在活跃告警,跳过
			if existingAlert != nil {
				continue
			}

			// 确定严重程度
			severity := domain.SeverityMedium
			if avgLatency > 1000 {
				severity = domain.SeverityHigh
			}
			if avgLatency > 2000 {
				severity = domain.SeverityCritical
			}

			metadata := domain.JSONB{
				"session_id":      session.ID.String(),
				"device_a_id":     session.DeviceAID.String(),
				"device_b_id":     session.DeviceBID.String(),
				"avg_latency_ms":  avgLatency,
				"connection_type": session.ConnectionType,
				"bytes_sent":      session.BytesSent,
				"bytes_received":  session.BytesReceived,
			}

			alert := &domain.Alert{
				ID:             uuid.New(),
				DeviceID:       &session.DeviceAID,
				Severity:       severity,
				Type:             domain.AlertTypeHighLatency,
				Title:          "检测到高延迟",
				Message:        "会话出现高延迟,可能影响用户体验。",
				Status:         domain.AlertStatusActive,
				Metadata:       metadata,
				CreatedAt:      now,
			}

			if err := t.alertRepo.Create(ctx, alert); err != nil {
				t.logger.Error("Failed to create high latency alert",
					zap.Error(err),
					zap.String("session_id", session.ID.String()),
				)
				continue
			}

			alertsCreated++
			t.logger.Info("High latency alert created",
				zap.String("alert_id", alert.ID.String()),
				zap.String("session_id", session.ID.String()),
				zap.Int("avg_latency_ms", avgLatency),
			)
		}
	}

	t.logger.Info("Performance monitor task completed",
		zap.Int("sessions_checked", len(sessions)),
		zap.Int("alerts_created", alertsCreated),
	)

	return nil
}

// checkExistingAlert 检查是否存在未解决的同类型告警
func (t *PerformanceMonitorTask) checkExistingAlert(
	ctx context.Context,
	deviceID uuid.UUID,
	alertType domain.AlertType,
) (*domain.Alert, error) {
	status := domain.AlertStatusActive
	alerts, _, err := t.alertRepo.FindByFilters(ctx, &repository.AlertFilters{
		DeviceID:  &deviceID,
		AlertType: &alertType,
		Status:    &status,
		Limit:     1,
	})

	if err != nil {
		return nil, err
	}

	if len(alerts) > 0 {
		return alerts[0], nil
	}

	return nil, nil
}
