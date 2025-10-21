package teams

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

// Config Microsoft Teams配置
type Config struct {
	WebhookURL  string                   // Teams Webhook URL
	Enabled     bool                     // 是否启用
	Priority    int                      // 优先级
	RetryConfig integrations.RetryConfig // 重试配置
	ColorMap    ColorMapping             // 颜色映射
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

// ColorMapping Teams颜色映射（十六进制）
type ColorMapping struct {
	Critical string // 危急：红色
	High     string // 高：橙色
	Medium   string // 中：黄色
	Low      string // 低：绿色
}

// MapColor 映射严重程度到颜色
func (m *ColorMapping) MapColor(severity domain.Severity) string {
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
		return "808080" // 灰色
	}
}

// Integration Microsoft Teams集成
type Integration struct {
	config     *Config
	httpClient *http.Client
	logger     *zap.Logger
}

// NewIntegration 创建Teams集成
func NewIntegration(config *Config, logger *zap.Logger) *Integration {
	if config.ColorMap.Critical == "" {
		// 默认颜色映射
		config.ColorMap = ColorMapping{
			Critical: "FF0000", // 红色
			High:     "FF6600", // 橙色
			Medium:   "FFCC00", // 黄色
			Low:      "36A64F", // 绿色
		}
	}

	if config.RetryConfig.MaxRetries == 0 {
		config.RetryConfig = integrations.DefaultRetryConfig()
	}

	return &Integration{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// Name 实现Integration接口
func (i *Integration) Name() string {
	return "teams"
}

// SendAlert 发送告警到Teams
func (i *Integration) SendAlert(ctx context.Context, alert *domain.Alert) error {
	// 构建Teams卡片消息
	card := i.buildAdaptiveCard(alert)

	// 序列化
	jsonData, err := json.Marshal(card)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "marshal", alert.ID.String(), err, false)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", i.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "create_request", alert.ID.String(), err, false)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return integrations.NewIntegrationError(i.Name(), "send_request", alert.ID.String(), err, true)
	}
	defer resp.Body.Close()

	// Teams Webhook成功返回200
	if resp.StatusCode == 200 {
		i.logger.Info("Teams message sent successfully",
			zap.String("alert_id", alert.ID.String()),
		)
		return nil
	}

	// 读取错误响应
	var errorMsg string
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	errorMsg = buf.String()

	err = fmt.Errorf("Teams webhook error: %s", errorMsg)
	retryable := resp.StatusCode >= 500 || resp.StatusCode == 429

	return integrations.NewIntegrationError(i.Name(), "webhook_error", alert.ID.String(), err, retryable)
}

// ResolveAlert Teams发送解决通知
func (i *Integration) ResolveAlert(ctx context.Context, alertID string) error {
	card := Message{
		Type:       "message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: AdaptiveCard{
					Type:    "AdaptiveCard",
					Version: "1.4",
					Body: []CardElement{
						{
							Type:   "TextBlock",
							Text:   "Alert Resolved",
							Weight: "bolder",
							Size:   "large",
							Color:  "good",
						},
						{
							Type: "TextBlock",
							Text: fmt.Sprintf("Alert `%s` has been resolved", alertID),
							Wrap: true,
						},
					},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(card)
	req, _ := http.NewRequestWithContext(ctx, "POST", i.config.WebhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		i.logger.Info("Teams resolve notification sent",
			zap.String("alert_id", alertID),
		)
		return nil
	}

	return fmt.Errorf("failed to send resolve notification: status %d", resp.StatusCode)
}

// UpdateAlert Teams发送状态变更通知
func (i *Integration) UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error {
	if status == domain.AlertStatusResolved {
		return i.ResolveAlert(ctx, alertID)
	}

	card := Message{
		Type:       "message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: AdaptiveCard{
					Type:    "AdaptiveCard",
					Version: "1.4",
					Body: []CardElement{
						{
							Type:   "TextBlock",
							Text:   "Alert Status Updated",
							Weight: "bolder",
							Size:   "medium",
						},
						{
							Type: "TextBlock",
							Text: fmt.Sprintf("Alert `%s` status changed to **%s**", alertID, status),
							Wrap: true,
						},
					},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(card)
	req, _ := http.NewRequestWithContext(ctx, "POST", i.config.WebhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ValidateConfig 验证配置
func (i *Integration) ValidateConfig() error {
	if i.config.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}
	return nil
}

// HealthCheck 健康检查
func (i *Integration) HealthCheck(ctx context.Context) error {
	card := Message{
		Type:       "message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: AdaptiveCard{
					Type:    "AdaptiveCard",
					Version: "1.4",
					Body: []CardElement{
						{
							Type:   "TextBlock",
							Text:   "Health Check",
							Weight: "bolder",
							Size:   "medium",
							Color:  "good",
						},
						{
							Type: "TextBlock",
							Text: "EdgeLink Alert Service is healthy",
							Wrap: true,
						},
					},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(card)
	req, _ := http.NewRequestWithContext(ctx, "POST", i.config.WebhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("health check failed: status %d", resp.StatusCode)
}

// buildAdaptiveCard 构建Adaptive Card消息
func (i *Integration) buildAdaptiveCard(alert *domain.Alert) Message {
	color := i.config.ColorMap.MapColor(alert.Severity)

	// 构建事实列表
	facts := []Fact{
		{
			Title: "Severity",
			Value: string(alert.Severity),
		},
		{
			Title: "Type",
			Value: string(alert.Type),
		},
		{
			Title: "Time",
			Value: alert.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}

	if alert.DeviceID != nil {
		facts = append(facts, Fact{
			Title: "Device ID",
			Value: alert.DeviceID.String(),
		})
	}

	// 构建卡片元素
	body := []CardElement{
		{
			Type:   "TextBlock",
			Text:   alert.Title,
			Weight: "bolder",
			Size:   "large",
		},
		{
			Type: "TextBlock",
			Text: alert.Message,
			Wrap: true,
		},
		{
			Type:  "FactSet",
			Facts: facts,
		},
	}

	// 构建操作按钮
	actions := []CardAction{
		{
			Type:  "Action.OpenUrl",
			Title: "View Details",
			URL:   fmt.Sprintf("https://edgelink.example.com/alerts/%s", alert.ID.String()),
		},
	}

	card := Message{
		Type:       "message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: AdaptiveCard{
					Type:    "AdaptiveCard",
					Version: "1.4",
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Body:    body,
					Actions: actions,
					MSTeams: &MSTeamsMetadata{
						Width: "full",
					},
				},
			},
		},
	}

	// 添加主题颜色
	card.ThemeColor = color

	return card
}

// Message Teams消息结构
type Message struct {
	Type        string       `json:"type"`                   // message
	Attachments []Attachment `json:"attachments"`            // 附件（Adaptive Card）
	Summary     string       `json:"summary,omitempty"`      // 摘要（通知文本）
	ThemeColor  string       `json:"themeColor,omitempty"`   // 主题颜色
}

// Attachment Teams附件
type Attachment struct {
	ContentType string       `json:"contentType"` // application/vnd.microsoft.card.adaptive
	Content     AdaptiveCard `json:"content"`     // Adaptive Card内容
}

// AdaptiveCard Adaptive Card结构
type AdaptiveCard struct {
	Type    string        `json:"type"`              // AdaptiveCard
	Version string        `json:"version"`           // 1.4
	Schema  string        `json:"$schema,omitempty"` // Schema URL
	Body    []CardElement `json:"body"`              // 卡片主体元素
	Actions []CardAction  `json:"actions,omitempty"` // 操作按钮
	MSTeams *MSTeamsMetadata `json:"msteams,omitempty"` // Teams特定元数据
}

// CardElement 卡片元素
type CardElement struct {
	Type   string `json:"type"`             // TextBlock, FactSet, Image, etc.
	Text   string `json:"text,omitempty"`   // 文本内容
	Weight string `json:"weight,omitempty"` // 字重：lighter, default, bolder
	Size   string `json:"size,omitempty"`   // 大小：small, default, medium, large, extraLarge
	Color  string `json:"color,omitempty"`  // 颜色：default, dark, light, accent, good, warning, attention
	Wrap   bool   `json:"wrap,omitempty"`   // 是否换行
	Facts  []Fact `json:"facts,omitempty"`  // 事实列表（用于FactSet）
}

// Fact 事实
type Fact struct {
	Title string `json:"title"` // 标题
	Value string `json:"value"` // 值
}

// CardAction 卡片操作
type CardAction struct {
	Type  string `json:"type"`            // Action.OpenUrl, Action.Submit, etc.
	Title string `json:"title"`           // 按钮标题
	URL   string `json:"url,omitempty"`   // URL（用于OpenUrl）
	Data  interface{} `json:"data,omitempty"` // 数据（用于Submit）
}

// MSTeamsMetadata Teams特定元数据
type MSTeamsMetadata struct {
	Width string `json:"width,omitempty"` // full
}
