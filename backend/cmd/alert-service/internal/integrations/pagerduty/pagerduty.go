package pagerduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/edgelink/backend/cmd/alert-service/internal/integrations"
	"github.com/edgelink/backend/internal/domain"
	"go.uber.org/zap"
)

const (
	// PagerDuty Events API v2 endpoint
	eventsAPIURL = "https://events.pagerduty.com/v2/enqueue"
)

// Config PagerDuty配置
type Config struct {
	IntegrationKey string                              // PagerDuty Integration Key
	Enabled        bool                                // 是否启用
	Priority       int                                 // 优先级
	RetryConfig    integrations.RetryConfig            // 重试配置
	SeverityMap    integrations.AlertSeverityMapping   // 严重程度映射
	DefaultService string                              // 默认服务名称
}

// Enabled 实现IntegrationConfig接口
func (c *Config) Enabled() bool {
	return c.Enabled
}

// Priority 实现IntegrationConfig接口
func (c *Config) Priority() int {
	return c.Priority
}

// RetryConfig 实现IntegrationConfig接口
func (c *Config) RetryConfig() integrations.RetryConfig {
	return c.RetryConfig
}

// Integration PagerDuty集成
type Integration struct {
	config     *Config
	httpClient *http.Client
	logger     *zap.Logger
}

// NewIntegration 创建PagerDuty集成
func NewIntegration(config *Config, logger *zap.Logger) *Integration {
	if config.SeverityMap.Critical == "" {
		// 默认严重程度映射
		config.SeverityMap = integrations.AlertSeverityMapping{
			Critical: "critical",
			High:     "error",
			Medium:   "warning",
			Low:      "info",
		}
	}

	if config.RetryConfig.MaxRetries == 0 {
		config.RetryConfig = integrations.DefaultRetryConfig()
	}

	return &Integration{
		config: config,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger,
	}
}

// Name 实现Integration接口
func (i *Integration) Name() string {
	return "pagerduty"
}

// SendAlert 发送告警到PagerDuty
func (i *Integration) SendAlert(ctx context.Context, alert *domain.Alert) error {
	// 构建PagerDuty事件
	event := i.buildEvent(alert)

	// 序列化
	jsonData, err := json.Marshal(event)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "marshal", alert.ID.String(), err, false)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", eventsAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "create_request", alert.ID.String(), err, false)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "send_request", alert.ID.String(), err, true)
	}
	defer resp.Body.Close()

	// 检查响应
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		i.logger.Info("PagerDuty event sent successfully",
			zap.String("alert_id", alert.ID.String()),
			zap.Int("status_code", resp.StatusCode),
		)
		return nil
	}

	// 解析错误响应
	var errorResp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Errors  []string `json:"errors"`
	}
	json.NewDecoder(resp.Body).Decode(&errorResp)

	err = fmt.Errorf("PagerDuty API error: %s - %s", errorResp.Status, errorResp.Message)
	retryable := resp.StatusCode >= 500 || resp.StatusCode == 429 // 5xx或速率限制可重试

	return integrations.NewIntegrationError(i.Name(), "api_error", alert.ID.String(), err, retryable)
}

// ResolveAlert 解决PagerDuty告警
func (i *Integration) ResolveAlert(ctx context.Context, alertID string) error {
	event := Event{
		RoutingKey:  i.config.IntegrationKey,
		EventAction: "resolve",
		DedupKey:    alertID, // 使用alertID作为去重键
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "marshal", alertID, err, false)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", eventsAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "create_request", alertID, err, false)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "send_request", alertID, err, true)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		i.logger.Info("PagerDuty alert resolved",
			zap.String("alert_id", alertID),
		)
		return nil
	}

	return integrations.NewIntegrationError(i.Name(), "resolve_failed", alertID,
		fmt.Errorf("status code: %d", resp.StatusCode), resp.StatusCode >= 500)
}

// UpdateAlert 更新告警状态（PagerDuty通过acknowledge实现）
func (i *Integration) UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error {
	// PagerDuty仅支持acknowledge和resolve
	if status == domain.AlertStatusAcknowledged {
		event := Event{
			RoutingKey:  i.config.IntegrationKey,
			EventAction: "acknowledge",
			DedupKey:    alertID,
		}

		jsonData, _ := json.Marshal(event)
		req, _ := http.NewRequestWithContext(ctx, "POST", eventsAPIURL, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		resp, err := i.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
	} else if status == domain.AlertStatusResolved {
		return i.ResolveAlert(ctx, alertID)
	}

	return nil
}

// ValidateConfig 验证配置
func (i *Integration) ValidateConfig() error {
	if i.config.IntegrationKey == "" {
		return fmt.Errorf("integration_key is required")
	}
	return nil
}

// HealthCheck 健康检查
func (i *Integration) HealthCheck(ctx context.Context) error {
	// 发送一个测试事件（使用特殊的dedup_key）
	event := Event{
		RoutingKey:  i.config.IntegrationKey,
		EventAction: "trigger",
		DedupKey:    "edgelink-health-check-" + time.Now().Format("20060102"),
		Payload: EventPayload{
			Summary:  "EdgeLink Health Check",
			Severity: "info",
			Source:   "edgelink-alert-service",
		},
	}

	jsonData, _ := json.Marshal(event)
	req, _ := http.NewRequestWithContext(ctx, "POST", eventsAPIURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("health check failed: status %d", resp.StatusCode)
}

// buildEvent 构建PagerDuty事件
func (i *Integration) buildEvent(alert *domain.Alert) Event {
	severity := i.config.SeverityMap.MapSeverity(alert.Severity)

	payload := EventPayload{
		Summary:   alert.Title,
		Severity:  severity,
		Source:    i.getSource(alert),
		Timestamp: alert.CreatedAt.Format(time.RFC3339),
		Component: string(alert.Type),
		Group:     i.config.DefaultService,
	}

	// 添加自定义详情
	if alert.DeviceID != nil {
		payload.CustomDetails = map[string]interface{}{
			"device_id": alert.DeviceID.String(),
			"message":   alert.Message,
		}
	} else {
		payload.CustomDetails = map[string]interface{}{
			"message": alert.Message,
		}
	}

	// 添加元数据
	if alert.Metadata != nil {
		for k, v := range alert.Metadata {
			payload.CustomDetails[k] = v
		}
	}

	return Event{
		RoutingKey:  i.config.IntegrationKey,
		EventAction: "trigger",
		DedupKey:    alert.ID.String(), // 使用alert ID作为去重键
		Payload:     payload,
	}
}

// getSource 获取告警来源
func (i *Integration) getSource(alert *domain.Alert) string {
	if alert.DeviceID != nil {
		return fmt.Sprintf("device-%s", alert.DeviceID.String())
	}
	return "edgelink-system"
}

// Event PagerDuty事件结构
type Event struct {
	RoutingKey  string       `json:"routing_key"`           // Integration Key
	EventAction string       `json:"event_action"`          // trigger, acknowledge, resolve
	DedupKey    string       `json:"dedup_key,omitempty"`   // 去重键（建议使用alert ID）
	Payload     EventPayload `json:"payload,omitempty"`     // 事件负载
	Client      string       `json:"client,omitempty"`      // 客户端名称
	ClientURL   string       `json:"client_url,omitempty"`  // 客户端URL
}

// EventPayload PagerDuty事件负载
type EventPayload struct {
	Summary       string                 `json:"summary"`                  // 必填：事件摘要
	Severity      string                 `json:"severity"`                 // 必填：critical, error, warning, info
	Source        string                 `json:"source"`                   // 必填：告警来源
	Timestamp     string                 `json:"timestamp,omitempty"`      // ISO 8601时间戳
	Component     string                 `json:"component,omitempty"`      // 组件名称
	Group         string                 `json:"group,omitempty"`          // 逻辑分组
	Class         string                 `json:"class,omitempty"`          // 事件类别
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"` // 自定义详情
}
