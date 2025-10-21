package tasks

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SecurityMonitorTask 安全监控任务
type SecurityMonitorTask struct {
	redisClient *redis.Client
	alertRepo   repository.AlertRepository
	logger      *zap.Logger
}

// NewSecurityMonitorTask 创建安全监控任务
func NewSecurityMonitorTask(
	redisClient *redis.Client,
	alertRepo repository.AlertRepository,
	logger *zap.Logger,
) *SecurityMonitorTask {
	return &SecurityMonitorTask{
		redisClient: redisClient,
		alertRepo:   alertRepo,
		logger:      logger,
	}
}

// Run 执行安全监控 (检测失败认证尝试)
func (t *SecurityMonitorTask) Run(ctx context.Context) error {
	t.logger.Info("Running security monitor task")

	// 检查认证失败计数
	// 使用Redis存储认证失败次数: "auth_failures:{device_id}" -> count

	// 扫描所有认证失败键
	var cursor uint64
	var keys []string
	pattern := "auth_failures:*"

	for {
		var err error
		var batch []string
		batch, cursor, err = t.redisClient.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	alertsCreated := 0
	failureThreshold := int64(5) // 5次失败尝试触发告警
	now := time.Now()

	for _, key := range keys {
		// 获取失败次数
		count, err := t.redisClient.Get(ctx, key).Int64()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			t.logger.Error("Failed to get auth failure count",
				zap.Error(err),
				zap.String("key", key),
			)
			continue
		}

		// 如果失败次数超过阈值
		if count >= failureThreshold {
			// 从键中提取设备ID
			// key format: "auth_failures:{device_id}"
			deviceIDStr := key[len("auth_failures:"):]
			deviceID, err := uuid.Parse(deviceIDStr)
			if err != nil {
				t.logger.Warn("Invalid device ID in Redis key",
					zap.String("key", key),
				)
				continue
			}

			// 检查是否已存在未解决的安全告警
			existingAlert, err := t.checkExistingAlert(ctx, deviceID, domain.AlertTypeFailedAuth)
			if err != nil {
				t.logger.Error("Failed to check existing alert",
					zap.Error(err),
					zap.String("device_id", deviceID.String()),
				)
				continue
			}

			// 如果已存在活跃告警,跳过
			if existingAlert != nil {
				continue
			}

			// 确定严重程度
			severity := domain.SeverityHigh
			if count >= 10 {
				severity = domain.SeverityCritical
			}

			metadata := domain.JSONB{
				"device_id":       deviceID.String(),
				"failure_count":   count,
				"detection_time":  now.Format(time.RFC3339),
				"alert_type":      "authentication_failures",
			}

			alert := &domain.Alert{
				ID:             uuid.New(),
				DeviceID:       &deviceID,
				Severity:       severity,
				Type:             domain.AlertTypeFailedAuth,
				Title:          "检测到多次认证失败",
				Message:        "设备出现多次认证失败尝试,可能存在安全风险。",
				Status:         domain.AlertStatusActive,
				Metadata:       metadata,
				CreatedAt:      now,
			}

			if err := t.alertRepo.Create(ctx, alert); err != nil {
				t.logger.Error("Failed to create security alert",
					zap.Error(err),
					zap.String("device_id", deviceID.String()),
				)
				continue
			}

			alertsCreated++
			t.logger.Warn("Security alert created for authentication failures",
				zap.String("alert_id", alert.ID.String()),
				zap.String("device_id", deviceID.String()),
				zap.Int64("failure_count", count),
			)

			// 清除计数器 (告警已创建)
			t.redisClient.Del(ctx, key)
		}
	}

	t.logger.Info("Security monitor task completed",
		zap.Int("keys_checked", len(keys)),
		zap.Int("alerts_created", alertsCreated),
	)

	return nil
}

// checkExistingAlert 检查是否存在未解决的同类型告警
func (t *SecurityMonitorTask) checkExistingAlert(
	ctx context.Context,
	deviceID uuid.UUID,
	alertType domain.AlertType,
) (*domain.Alert, error) {
	status := domain.AlertStatusActive

	// 查询最近1小时内的安全告警
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	alerts, _, err := t.alertRepo.FindByFilters(ctx, &repository.AlertFilters{
		DeviceID:  &deviceID,
		AlertType: &alertType,
		Status:    &status,
		StartTime: &oneHourAgo,
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
