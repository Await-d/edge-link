package integrations

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
)

// Integration 统一告警集成接口
// 所有第三方平台适配器必须实现此接口
type Integration interface {
	// Name 返回集成平台名称
	Name() string

	// SendAlert 发送单个告警
	SendAlert(ctx context.Context, alert *domain.Alert) error

	// ResolveAlert 解决告警（如果平台支持）
	ResolveAlert(ctx context.Context, alertID string) error

	// UpdateAlert 更新告警状态（如果平台支持）
	UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error

	// ValidateConfig 验证配置是否有效
	ValidateConfig() error

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// IntegrationConfig 集成配置接口
type IntegrationConfig interface {
	// Enabled 是否启用该集成
	Enabled() bool

	// Priority 优先级（数字越小优先级越高，用于备用通道）
	Priority() int

	// RetryConfig 获取重试配置
	RetryConfig() RetryConfig
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int           // 最大重试次数
	InitialDelay  time.Duration // 初始延迟
	MaxDelay      time.Duration // 最大延迟
	BackoffFactor float64       // 退避因子（指数退避）
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  2 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// AlertSeverityMapping 告警严重程度映射
type AlertSeverityMapping struct {
	Critical string
	High     string
	Medium   string
	Low      string
}

// MapSeverity 映射EdgeLink严重程度到平台特定值
func (m *AlertSeverityMapping) MapSeverity(severity domain.Severity) string {
	switch severity {
	case domain.SeverityCritical:
		return m.Critical
	case domain.SeverityHigh:
		return m.High
	case domain.SeverityMedium:
		return m.Medium
	case domain.SeverityLow:
		return m.Low
	default:
		return m.Low
	}
}

// IntegrationMetrics 集成指标
type IntegrationMetrics struct {
	TotalSent      int64         // 总发送数
	SuccessCount   int64         // 成功数
	FailureCount   int64         // 失败数
	LastSentTime   time.Time     // 最后发送时间
	AvgResponseTime time.Duration // 平均响应时间
}

// IntegrationError 集成错误
type IntegrationError struct {
	Integration string // 集成平台名称
	Operation   string // 操作类型
	AlertID     string // 告警ID
	Err         error  // 原始错误
	Retryable   bool   // 是否可重试
}

func (e *IntegrationError) Error() string {
	return e.Err.Error()
}

func (e *IntegrationError) Unwrap() error {
	return e.Err
}

// NewIntegrationError 创建集成错误
func NewIntegrationError(integration, operation, alertID string, err error, retryable bool) *IntegrationError {
	return &IntegrationError{
		Integration: integration,
		Operation:   operation,
		AlertID:     alertID,
		Err:         err,
		Retryable:   retryable,
	}
}
