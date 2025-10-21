package slack

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

// Config Slack配置
type Config struct {
	WebhookURL  string                   // Slack Webhook URL
	Enabled     bool                     // 是否启用
	Priority    int                      // 优先级
	RetryConfig integrations.RetryConfig // 重试配置
	Channel     string                   // 默认频道（可选，覆盖Webhook默认频道）
	Username    string                   // 机器人用户名
	IconEmoji   string                   // 图标emoji
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

// ColorMapping Slack颜色映射
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
		return "#808080"
	}
}

// Integration Slack集成
type Integration struct {
	config     *Config
	httpClient *http.Client
	logger     *zap.Logger
}

// NewIntegration 创建Slack集成
func NewIntegration(config *Config, logger *zap.Logger) *Integration {
	if config.ColorMap.Critical == "" {
		// 默认颜色映射
		config.ColorMap = ColorMapping{
			Critical: "#FF0000", // 红色
			High:     "#FF6600", // 橙色
			Medium:   "#FFCC00", // 黄色
			Low:      "#36A64F", // 绿色
		}
	}

	if config.Username == "" {
		config.Username = "EdgeLink Alert Bot"
	}

	if config.IconEmoji == "" {
		config.IconEmoji = ":warning:"
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
	return "slack"
}

// SendAlert 发送告警到Slack
func (i *Integration) SendAlert(ctx context.Context, alert *domain.Alert) error {
	// 构建Slack消息
	message := i.buildMessage(alert)

	// 序列化
	jsonData, err := json.Marshal(message)
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

	// Slack Webhook返回200表示成功
	if resp.StatusCode == 200 {
		i.logger.Info("Slack message sent successfully",
			zap.String("alert_id", alert.ID.String()),
		)
		return nil
	}

	// 读取错误响应
	var errorMsg string
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	errorMsg = buf.String()

	err = fmt.Errorf("Slack webhook error: %s", errorMsg)
	retryable := resp.StatusCode >= 500 || resp.StatusCode == 429

	return integrations.NewIntegrationError(i.Name(), "webhook_error", alert.ID.String(), err, retryable)
}

// ResolveAlert Slack不支持原生解决告警，发送解决通知
func (i *Integration) ResolveAlert(ctx context.Context, alertID string) error {
	message := Message{
		Username:  i.config.Username,
		IconEmoji: ":white_check_mark:",
		Attachments: []Attachment{
			{
				Color:     "#36A64F",
				Title:     "Alert Resolved",
				Text:      fmt.Sprintf("Alert `%s` has been resolved", alertID),
				Timestamp: time.Now().Unix(),
			},
		},
	}

	if i.config.Channel != "" {
		message.Channel = i.config.Channel
	}

	jsonData, _ := json.Marshal(message)
	req, _ := http.NewRequestWithContext(ctx, "POST", i.config.WebhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		i.logger.Info("Slack resolve notification sent",
			zap.String("alert_id", alertID),
		)
		return nil
	}

	return fmt.Errorf("failed to send resolve notification: status %d", resp.StatusCode)
}

// UpdateAlert Slack不支持原生更新，发送状态变更通知
func (i *Integration) UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error {
	if status == domain.AlertStatusResolved {
		return i.ResolveAlert(ctx, alertID)
	}

	// 对于其他状态变更，发送简单通知
	message := Message{
		Username:  i.config.Username,
		IconEmoji: i.config.IconEmoji,
		Text:      fmt.Sprintf("Alert `%s` status changed to *%s*", alertID, status),
	}

	if i.config.Channel != "" {
		message.Channel = i.config.Channel
	}

	jsonData, _ := json.Marshal(message)
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
	// 发送测试消息
	message := Message{
		Username:  i.config.Username,
		IconEmoji: ":white_check_mark:",
		Text:      "EdgeLink Alert Service - Health Check :heart:",
	}

	if i.config.Channel != "" {
		message.Channel = i.config.Channel
	}

	jsonData, _ := json.Marshal(message)
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

// buildMessage 构建Slack消息（使用Block Kit）
func (i *Integration) buildMessage(alert *domain.Alert) Message {
	color := i.config.ColorMap.MapColor(alert.Severity)

	// 构建字段
	fields := []Field{
		{
			Title: "Severity",
			Value: string(alert.Severity),
			Short: true,
		},
		{
			Title: "Type",
			Value: string(alert.Type),
			Short: true,
		},
		{
			Title: "Time",
			Value: alert.CreatedAt.Format("2006-01-02 15:04:05"),
			Short: true,
		},
	}

	if alert.DeviceID != nil {
		fields = append(fields, Field{
			Title: "Device ID",
			Value: alert.DeviceID.String(),
			Short: true,
		})
	}

	// 构建附件
	attachment := Attachment{
		Color:      color,
		Title:      alert.Title,
		Text:       alert.Message,
		Fields:     fields,
		Footer:     "EdgeLink Alert Service",
		FooterIcon: "https://platform.slack-edge.com/img/default_application_icon.png",
		Timestamp:  alert.CreatedAt.Unix(),
	}

	// 添加操作按钮（可选）
	attachment.Actions = []Action{
		{
			Type:  "button",
			Text:  "Acknowledge",
			Style: "primary",
			Value: fmt.Sprintf("ack:%s", alert.ID.String()),
		},
		{
			Type:  "button",
			Text:  "Resolve",
			Style: "danger",
			Value: fmt.Sprintf("resolve:%s", alert.ID.String()),
		},
	}

	message := Message{
		Username:    i.config.Username,
		IconEmoji:   i.getEmojiForSeverity(alert.Severity),
		Attachments: []Attachment{attachment},
	}

	if i.config.Channel != "" {
		message.Channel = i.config.Channel
	}

	return message
}

// getEmojiForSeverity 根据严重程度获取emoji
func (i *Integration) getEmojiForSeverity(severity domain.Severity) string {
	switch severity {
	case domain.SeverityCritical:
		return ":rotating_light:"
	case domain.SeverityHigh:
		return ":warning:"
	case domain.SeverityMedium:
		return ":large_orange_diamond:"
	case domain.SeverityLow:
		return ":information_source:"
	default:
		return ":grey_question:"
	}
}

// Message Slack消息结构
type Message struct {
	Channel     string       `json:"channel,omitempty"`      // 频道（覆盖webhook默认）
	Username    string       `json:"username,omitempty"`     // 用户名
	IconEmoji   string       `json:"icon_emoji,omitempty"`   // 图标emoji
	IconURL     string       `json:"icon_url,omitempty"`     // 图标URL
	Text        string       `json:"text,omitempty"`         // 文本内容
	Attachments []Attachment `json:"attachments,omitempty"`  // 附件（丰富格式）
	Blocks      []Block      `json:"blocks,omitempty"`       // Blocks（新格式）
}

// Attachment Slack附件（传统格式）
type Attachment struct {
	Color      string   `json:"color,omitempty"`       // 侧边栏颜色
	Fallback   string   `json:"fallback,omitempty"`    // 后备文本
	Title      string   `json:"title,omitempty"`       // 标题
	TitleLink  string   `json:"title_link,omitempty"`  // 标题链接
	Text       string   `json:"text,omitempty"`        // 文本内容
	Pretext    string   `json:"pretext,omitempty"`     // 前置文本
	Fields     []Field  `json:"fields,omitempty"`      // 字段列表
	Footer     string   `json:"footer,omitempty"`      // 页脚
	FooterIcon string   `json:"footer_icon,omitempty"` // 页脚图标
	Timestamp  int64    `json:"ts,omitempty"`          // Unix时间戳
	Actions    []Action `json:"actions,omitempty"`     // 操作按钮
}

// Field Slack字段
type Field struct {
	Title string `json:"title"`           // 字段标题
	Value string `json:"value"`           // 字段值
	Short bool   `json:"short,omitempty"` // 是否短字段（并排显示）
}

// Action Slack操作按钮
type Action struct {
	Type    string `json:"type"`              // 类型：button
	Text    string `json:"text"`              // 按钮文本
	URL     string `json:"url,omitempty"`     // 链接URL
	Value   string `json:"value,omitempty"`   // 值
	Style   string `json:"style,omitempty"`   // 样式：default, primary, danger
	Confirm *struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"confirm,omitempty"` // 确认对话框
}

// Block Slack Block Kit（新格式，更灵活）
type Block struct {
	Type     string                 `json:"type"`                // section, divider, image, actions, etc.
	Text     *TextObject            `json:"text,omitempty"`      // 文本对象
	Fields   []*TextObject          `json:"fields,omitempty"`    // 字段列表
	Accessory interface{}           `json:"accessory,omitempty"` // 附件元素
	Elements []interface{}          `json:"elements,omitempty"`  // 元素列表
}

// TextObject Slack文本对象
type TextObject struct {
	Type string `json:"type"`           // plain_text, mrkdwn
	Text string `json:"text"`           // 文本内容
	Emoji bool  `json:"emoji,omitempty"` // 是否解析emoji
}
