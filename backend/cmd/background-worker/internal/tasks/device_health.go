package tasks

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DeviceHealthTask 设备健康检查任务
type DeviceHealthTask struct {
	deviceRepo repository.DeviceRepository
	alertRepo  repository.AlertRepository
	logger     *zap.Logger
}

// NewDeviceHealthTask 创建设备健康检查任务
func NewDeviceHealthTask(
	deviceRepo repository.DeviceRepository,
	alertRepo repository.AlertRepository,
	logger *zap.Logger,
) *DeviceHealthTask {
	return &DeviceHealthTask{
		deviceRepo: deviceRepo,
		alertRepo:  alertRepo,
		logger:     logger,
	}
}

// Run 执行设备健康检查
func (t *DeviceHealthTask) Run(ctx context.Context) error {
	t.logger.Info("Running device health check task")

	// 查询所有在线设备
	onlineFlag := true
	devices, err := t.deviceRepo.FindByVirtualNetwork(ctx, uuid.UUID{}, &onlineFlag)
	if err != nil {
		return err
	}

	now := time.Now()
	offlineThreshold := 5 * time.Minute
	alertsCreated := 0

	for _, device := range devices {
		// 检查最后上线时间
		if device.LastSeenAt == nil {
			continue
		}

		timeSinceLastSeen := now.Sub(*device.LastSeenAt)

		// 如果设备离线超过阈值
		if timeSinceLastSeen > offlineThreshold {
			// 检查是否已存在未解决的离线告警
			existingAlert, err := t.checkExistingAlert(ctx, device.ID, domain.AlertTypeDeviceOffline)
			if err != nil {
				t.logger.Error("Failed to check existing alert",
					zap.Error(err),
					zap.String("device_id", device.ID.String()),
				)
				continue
			}

			// 如果已存在活跃告警,跳过
			if existingAlert != nil {
				continue
			}

			// 创建新的离线告警
			severity := domain.SeverityHigh
			if timeSinceLastSeen > 30*time.Minute {
				severity = domain.SeverityCritical
			}

			metadata := domain.JSONB{
				"device_name":        device.Name,
				"last_seen_at":       device.LastSeenAt.Format(time.RFC3339),
				"offline_duration":   timeSinceLastSeen.String(),
				"virtual_network_id": device.VirtualNetworkID.String(),
				"platform":           device.Platform,
			}

			alert := &domain.Alert{
				ID:             uuid.New(),
				DeviceID:       &device.ID,				Severity:       severity,
				Type:             domain.AlertTypeDeviceOffline,
				Title:          "设备离线",
				Message:        "设备已离线超过阈值时间,请检查设备连接状态。离线时长: " + timeSinceLastSeen.String(),
				Status:         domain.AlertStatusActive,
				Metadata:       metadata,
				CreatedAt:      now,
			}

			if err := t.alertRepo.Create(ctx, alert); err != nil {
				t.logger.Error("Failed to create offline alert",
					zap.Error(err),
					zap.String("device_id", device.ID.String()),
				)
				continue
			}

			alertsCreated++
			t.logger.Info("Offline alert created",
				zap.String("alert_id", alert.ID.String()),
				zap.String("device_id", device.ID.String()),
				zap.String("device_name", device.Name),
				zap.Duration("offline_duration", timeSinceLastSeen),
			)
		}
	}

	t.logger.Info("Device health check completed",
		zap.Int("devices_checked", len(devices)),
		zap.Int("alerts_created", alertsCreated),
	)

	return nil
}

// checkExistingAlert 检查是否存在未解决的同类型告警
func (t *DeviceHealthTask) checkExistingAlert(
	ctx context.Context,
	deviceID uuid.UUID,
	alertType domain.AlertType,
) (*domain.Alert, error) {
	// 查询该设备的活跃告警
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
