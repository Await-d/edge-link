package checker

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// HealthIssue 健康问题
type HealthIssue struct {
	Type        string                 // 问题类型: "device_offline", "high_latency", "connection_failed"
	DeviceID    string                 // 设备ID
	Severity    string                 // 严重程度: "critical", "high", "medium", "low"
	Message     string                 // 问题描述
	Metadata    map[string]interface{} // 附加元数据
	DetectedAt  time.Time              // 检测时间
}

// ThresholdChecker 阈值检查器
type ThresholdChecker struct {
	deviceRepo  repository.DeviceRepository
	sessionRepo repository.SessionRepository
	logger      *zap.Logger
}

// NewThresholdChecker 创建阈值检查器
func NewThresholdChecker(
	deviceRepo repository.DeviceRepository,
	sessionRepo repository.SessionRepository,
	logger *zap.Logger,
) *ThresholdChecker {
	return &ThresholdChecker{
		deviceRepo:  deviceRepo,
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

// CheckAll 执行所有健康检查
func (tc *ThresholdChecker) CheckAll(ctx context.Context) []HealthIssue {
	var issues []HealthIssue

	// 检查离线设备
	offlineIssues := tc.CheckOfflineDevices(ctx)
	issues = append(issues, offlineIssues...)

	// 检查高延迟
	latencyIssues := tc.CheckHighLatency(ctx)
	issues = append(issues, latencyIssues...)

	return issues
}

// CheckOfflineDevices 检查离线设备 (超过5分钟未上线)
func (tc *ThresholdChecker) CheckOfflineDevices(ctx context.Context) []HealthIssue {
	var issues []HealthIssue

	// 查询所有在线设备
	onlineFlag := true
	devices, err := tc.deviceRepo.FindByVirtualNetwork(ctx, uuidNil(), &onlineFlag)
	if err != nil {
		tc.logger.Error("Failed to query devices", zap.Error(err))
		return issues
	}

	now := time.Now()
	offlineThreshold := 5 * time.Minute

	for _, device := range devices {
		// 检查最后上线时间
		if device.LastSeenAt != nil {
			timeSinceLastSeen := now.Sub(*device.LastSeenAt)

			if timeSinceLastSeen > offlineThreshold {
				severity := "high"
				if timeSinceLastSeen > 30*time.Minute {
					severity = "critical"
				}

				issues = append(issues, HealthIssue{
					Type:     "device_offline",
					DeviceID: device.ID.String(),
					Severity: severity,
					Message:  "Device has been offline for an extended period",
					Metadata: map[string]interface{}{
						"device_name":       device.Name,
						"last_seen_at":      *device.LastSeenAt,
						"offline_duration":  timeSinceLastSeen.String(),
						"virtual_network_id": device.VirtualNetworkID.String(),
					},
					DetectedAt: now,
				})
			}
		}
	}

	return issues
}

// CheckHighLatency 检查高延迟会话 (平均延迟 > 500ms)
func (tc *ThresholdChecker) CheckHighLatency(ctx context.Context) []HealthIssue {
	var issues []HealthIssue

	// 查询活跃会话
	activeSessions, err := tc.sessionRepo.FindActiveSessions(ctx, 1000)
	if err != nil {
		tc.logger.Error("Failed to query active sessions", zap.Error(err))
		return issues
	}

	latencyThreshold := 500 // ms

	for _, session := range activeSessions {
		if session.AvgLatencyMs != nil && *session.AvgLatencyMs > latencyThreshold {
			severity := "medium"
			if *session.AvgLatencyMs > 1000 {
				severity = "high"
			}

			issues = append(issues, HealthIssue{
				Type:     "high_latency",
				DeviceID: session.DeviceAID.String(),
				Severity: severity,
				Message:  "High latency detected in active session",
				Metadata: map[string]interface{}{
					"session_id":       session.ID.String(),
					"device_a_id":      session.DeviceAID.String(),
					"device_b_id":      session.DeviceBID.String(),
					"avg_latency_ms":   *session.AvgLatencyMs,
					"connection_type":  session.ConnectionType,
				},
				DetectedAt: time.Now(),
			})
		}
	}

	return issues
}

// uuidNil 返回nil UUID (用于查询所有网络的设备)
func uuidNil() uuid.UUID {
	return uuid.UUID{}
}
