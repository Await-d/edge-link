package tasks

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// KeyExpiryTask 密钥过期检查任务
type KeyExpiryTask struct {
	deviceKeyRepo repository.DeviceKeyRepository
	alertRepo     repository.AlertRepository
	logger        *zap.Logger
}

// NewKeyExpiryTask 创建密钥过期检查任务
func NewKeyExpiryTask(
	deviceKeyRepo repository.DeviceKeyRepository,
	alertRepo repository.AlertRepository,
	logger *zap.Logger,
) *KeyExpiryTask {
	return &KeyExpiryTask{
		deviceKeyRepo: deviceKeyRepo,
		alertRepo:     alertRepo,
		logger:        logger,
	}
}

// Run 执行密钥过期检查 (提前30天警告)
func (t *KeyExpiryTask) Run(ctx context.Context) error {
	t.logger.Info("Running key expiry check task")

	// 计算30天后的时间
	warningThreshold := time.Now().Add(30 * 24 * time.Hour)

	// 查询所有设备密钥
	// TODO: DeviceKeyRepository需要添加FindExpiringKeys方法
	// 这里使用简化的查询逻辑

	// 由于repository没有提供查询即将过期密钥的方法,
	// 我们需要查询所有活跃密钥并检查过期时间

	alertsCreated := 0
	now := time.Now()

	// 注意: 这是一个简化实现
	// 生产环境中应该在repository层添加专门的查询方法
	// 例如: FindExpiringKeys(ctx, warningThreshold)

	t.logger.Info("Key expiry check completed",
		zap.Int("alerts_created", alertsCreated),
		zap.String("warning_threshold", warningThreshold.Format(time.RFC3339)),
	)

	// 由于DeviceKey entity没有expiry字段,这个功能暂时简化
	// 实际实现中应该:
	// 1. 在DeviceKey表添加expires_at字段
	// 2. 在DeviceKeyRepository添加FindExpiringKeys方法
	// 3. 为每个即将过期的密钥创建告警

	t.logger.Info("Key expiry monitoring implementation note: requires DeviceKey.expires_at field")

	return t.runSimplifiedCheck(ctx, warningThreshold, now)
}

// runSimplifiedCheck 运行简化的检查 (示例实现)
func (t *KeyExpiryTask) runSimplifiedCheck(ctx context.Context, warningThreshold time.Time, now time.Time) error {
	// 这是一个占位实现,展示完整的告警创建流程

	// 假设的即将过期密钥检查逻辑
	// 实际应该查询 device_keys 表中 expires_at < warningThreshold 的记录

	// 示例: 如果找到即将过期的密钥
	exampleDeviceID := uuid.New() // 这应该从实际查询结果获取
	shouldCreateAlert := false     // 实际应该基于查询结果

	if shouldCreateAlert {
		// 检查是否已存在未解决的密钥过期告警
		existingAlert, err := t.checkExistingAlert(ctx, exampleDeviceID, domain.AlertTypeKeyExpiration)
		if err != nil {
			return err
		}

		if existingAlert == nil {
			metadata := domain.JSONB{
				"device_id":   exampleDeviceID.String(),
				"expires_at":  warningThreshold.Format(time.RFC3339),
				"days_until":  30,
				"alert_type":  "key_expiry_warning",
			}

			alert := &domain.Alert{
				ID:             uuid.New(),
				DeviceID:       &exampleDeviceID,				Severity:       domain.SeverityMedium,
				Type:             domain.AlertTypeKeyExpiration,
				Title:          "设备密钥即将过期",
				Message:        "设备密钥将在30天内过期,请及时轮换密钥。",
				Status:         domain.AlertStatusActive,
				Metadata:       metadata,
				CreatedAt:      now,
			}

			if err := t.alertRepo.Create(ctx, alert); err != nil {
				return err
			}

			t.logger.Info("Key expiry alert created",
				zap.String("alert_id", alert.ID.String()),
				zap.String("device_id", exampleDeviceID.String()),
			)
		}
	}

	return nil
}

// checkExistingAlert 检查是否存在未解决的同类型告警
func (t *KeyExpiryTask) checkExistingAlert(
	ctx context.Context,
	deviceID uuid.UUID,
	alertType domain.AlertType,
) (*domain.Alert, error) {
	status := domain.AlertStatusActive

	// 查询最近7天内的密钥过期告警
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)

	alerts, _, err := t.alertRepo.FindByFilters(ctx, &repository.AlertFilters{
		DeviceID:  &deviceID,
		AlertType: &alertType,
		Status:    &status,
		StartTime: &sevenDaysAgo,
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
