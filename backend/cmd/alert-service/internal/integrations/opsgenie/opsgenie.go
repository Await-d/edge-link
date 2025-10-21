package opsgenie

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
	// Opsgenie API endpoint
	alertsAPIURL = "https://api.opsgenie.com/v2/alerts"
)

// Config Opsgenie配置
type Config struct {
	APIKey         string                              // Opsgenie API Key
	Enabled        bool                                // 是否启用
	Priority       int                                 // 优先级
	RetryConfig    integrations.RetryConfig            // 重试配置
	PriorityMap    PriorityMapping                     // 优先级映射
	DefaultTeams   []string                            // 默认团队
	DefaultTags    []string                            // 默认标签
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

// PriorityMapping Opsgenie优先级映射
type PriorityMapping struct {
	Critical string // P1
	High     string // P2
	Medium   string // P3
	Low      string // P4
}

// MapPriority 映射严重程度到Opsgenie优先级
func (m *PriorityMapping) MapPriority(severity domain.Severity) string {
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
		return "P5"
	}
}

// Integration Opsgenie集成
type Integration struct {
	config     *Config
	httpClient *http.Client
	logger     *zap.Logger
}

// NewIntegration 创建Opsgenie集成
func NewIntegration(config *Config, logger *zap.Logger) *Integration {
	if config.PriorityMap.Critical == "" {
		// 默认优先级映射
		config.PriorityMap = PriorityMapping{
			Critical: "P1",
			High:     "P2",
			Medium:   "P3",
			Low:      "P4",
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
	return "opsgenie"
}

// SendAlert 发送告警到Opsgenie
func (i *Integration) SendAlert(ctx context.Context, alert *domain.Alert) error {
	// 构建Opsgenie告警
	opsgenieAlert := i.buildAlert(alert)

	// 序列化
	jsonData, err := json.Marshal(opsgenieAlert)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "marshal", alert.ID.String(), err, false)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", alertsAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "create_request", alert.ID.String(), err, false)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "GenieKey "+i.config.APIKey)

	// 发送请求
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "send_request", alert.ID.String(), err, true)
	}
	defer resp.Body.Close()

	// 检查响应
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		i.logger.Info("Opsgenie alert created successfully",
			zap.String("alert_id", alert.ID.String()),
			zap.Int("status_code", resp.StatusCode),
		)
		return nil
	}

	// 解析错误响应
	var errorResp struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}
	json.NewDecoder(resp.Body).Decode(&errorResp)

	err = fmt.Errorf("Opsgenie API error: %s (code: %d)", errorResp.Message, errorResp.Code)
	retryable := resp.StatusCode >= 500 || resp.StatusCode == 429

	return integrations.NewIntegrationError(i.Name(), "api_error", alert.ID.String(), err, retryable)
}

// ResolveAlert 解决Opsgenie告警
func (i *Integration) ResolveAlert(ctx context.Context, alertID string) error {
	url := fmt.Sprintf("%s/%s/close", alertsAPIURL, alertID)

	closeRequest := struct {
		User string `json:"user"`
		Note string `json:"note"`
	}{
		User: "EdgeLink System",
		Note: "Alert resolved automatically by EdgeLink",
	}

	jsonData, _ := json.Marshal(closeRequest)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "create_request", alertID, err, false)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "GenieKey "+i.config.APIKey)

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "send_request", alertID, err, true)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		i.logger.Info("Opsgenie alert closed",
			zap.String("alert_id", alertID),
		)
		return nil
	}

	return integrations.NewIntegrationError(i.Name(), "close_failed", alertID,
		fmt.Errorf("status code: %d", resp.StatusCode), resp.StatusCode >= 500)
}

// UpdateAlert 更新告警状态
func (i *Integration) UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error {
	var url string
	var action string

	switch status {
	case domain.AlertStatusAcknowledged:
		url = fmt.Sprintf("%s/%s/acknowledge", alertsAPIURL, alertID)
		action = "acknowledge"
	case domain.AlertStatusResolved:
		return i.ResolveAlert(ctx, alertID)
	default:
		return nil
	}

	updateRequest := struct {
		User string `json:"user"`
		Note string `json:"note"`
	}{
		User: "EdgeLink System",
		Note: fmt.Sprintf("Alert %s by EdgeLink", action),
	}

	jsonData, _ := json.Marshal(updateRequest)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "GenieKey "+i.config.APIKey)

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("update failed: status code %d", resp.StatusCode)
}

// ValidateConfig 验证配置
func (i *Integration) ValidateConfig() error {
	if i.config.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	return nil
}

// HealthCheck 健康检查
func (i *Integration) HealthCheck(ctx context.Context) error {
	// 使用列表API进行健康检查
	req, _ := http.NewRequestWithContext(ctx, "GET", alertsAPIURL+"?limit=1", nil)
	req.Header.Set("Authorization", "GenieKey "+i.config.APIKey)

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

// buildAlert 构建Opsgenie告警
func (i *Integration) buildAlert(alert *domain.Alert) Alert {
	priority := i.config.PriorityMap.MapPriority(alert.Severity)

	opsgenieAlert := Alert{
		Message:     alert.Title,
		Alias:       alert.ID.String(), // 使用alert ID作为alias
		Description: alert.Message,
		Priority:    priority,
		Source:      i.getSource(alert),
		Tags:        i.buildTags(alert),
		Details:     i.buildDetails(alert),
	}

	// 添加团队路由
	if len(i.config.DefaultTeams) > 0 {
		opsgenieAlert.Teams = make([]Team, len(i.config.DefaultTeams))
		for idx, teamName := range i.config.DefaultTeams {
			opsgenieAlert.Teams[idx] = Team{Name: teamName}
		}
	}

	return opsgenieAlert
}

// getSource 获取告警来源
func (i *Integration) getSource(alert *domain.Alert) string {
	if alert.DeviceID != nil {
		return fmt.Sprintf("device-%s", alert.DeviceID.String())
	}
	return "edgelink-system"
}

// buildTags 构建标签
func (i *Integration) buildTags(alert *domain.Alert) []string {
	tags := append([]string{}, i.config.DefaultTags...)
	tags = append(tags, string(alert.Type), string(alert.Severity))

	if alert.DeviceID != nil {
		tags = append(tags, "device")
	}

	return tags
}

// buildDetails 构建详情
func (i *Integration) buildDetails(alert *domain.Alert) map[string]string {
	details := make(map[string]string)

	details["alert_type"] = string(alert.Type)
	details["severity"] = string(alert.Severity)
	details["created_at"] = alert.CreatedAt.Format(time.RFC3339)

	if alert.DeviceID != nil {
		details["device_id"] = alert.DeviceID.String()
	}

	// 添加元数据
	if alert.Metadata != nil {
		for k, v := range alert.Metadata {
			details[k] = fmt.Sprintf("%v", v)
		}
	}

	return details
}

// Alert Opsgenie告警结构
type Alert struct {
	Message     string            `json:"message"`               // 必填：告警消息
	Alias       string            `json:"alias,omitempty"`       // 告警别名（用于去重）
	Description string            `json:"description,omitempty"` // 详细描述
	Priority    string            `json:"priority,omitempty"`    // P1-P5
	Source      string            `json:"source,omitempty"`      // 告警来源
	Tags        []string          `json:"tags,omitempty"`        // 标签
	Details     map[string]string `json:"details,omitempty"`     // 自定义详情
	Entity      string            `json:"entity,omitempty"`      // 实体
	Teams       []Team            `json:"teams,omitempty"`       // 团队路由
	VisibleTo   []Responder       `json:"visibleTo,omitempty"`   // 可见性
}

// Team 团队
type Team struct {
	Name string `json:"name,omitempty"` // 团队名称
	ID   string `json:"id,omitempty"`   // 团队ID
}

// Responder 响应者
type Responder struct {
	Type     string `json:"type"`               // team, user, escalation, schedule
	Name     string `json:"name,omitempty"`     // 名称
	ID       string `json:"id,omitempty"`       // ID
	Username string `json:"username,omitempty"` // 用户名
}
