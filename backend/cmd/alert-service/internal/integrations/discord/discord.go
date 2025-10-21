package discord

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

// Config Discord配置
type Config struct {
	WebhookURL  string                   // Discord Webhook URL
	Enabled     bool                     // 是否启用
	Priority    int                      // 优先级
	RetryConfig integrations.RetryConfig // 重试配置
	Username    string                   // 机器人用户名
	AvatarURL   string                   // 头像URL
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

// ColorMapping Discord颜色映射（十进制颜色值）
type ColorMapping struct {
	Critical int // 危急：红色
	High     int // 高：橙色
	Medium   int // 中：黄色
	Low      int // 低：绿色
}

// MapColor 映射严重程度到颜色
func (m *ColorMapping) MapColor(severity domain.Severity) int {
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
		return 8421504 // 灰色
	}
}

// Integration Discord集成
type Integration struct {
	config     *Config
	httpClient *http.Client
	logger     *zap.Logger
}

// NewIntegration 创建Discord集成
func NewIntegration(config *Config, logger *zap.Logger) *Integration {
	if config.ColorMap.Critical == 0 {
		// 默认颜色映射（十进制）
		config.ColorMap = ColorMapping{
			Critical: 16711680, // 红色 #FF0000
			High:     16744192, // 橙色 #FF6600
			Medium:   16776960, // 黄色 #FFFF00
			Low:      3581519,  // 绿色 #36A64F
		}
	}

	if config.Username == "" {
		config.Username = "EdgeLink Alert Bot"
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
	return "discord"
}

// SendAlert 发送告警到Discord
func (i *Integration) SendAlert(ctx context.Context, alert *domain.Alert) error {
	// 构建Discord消息
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

	// Discord Webhook成功返回204 No Content
	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		i.logger.Info("Discord message sent successfully",
			zap.String("alert_id", alert.ID.String()),
		)
		return nil
	}

	// 读取错误响应
	var errorResp struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}
	json.NewDecoder(resp.Body).Decode(&errorResp)

	err = fmt.Errorf("Discord webhook error: %s (code: %d)", errorResp.Message, errorResp.Code)
	retryable := resp.StatusCode >= 500 || resp.StatusCode == 429

	return integrations.NewIntegrationError(i.Name(), "webhook_error", alert.ID.String(), err, retryable)
}

// ResolveAlert Discord发送解决通知
func (i *Integration) ResolveAlert(ctx context.Context, alertID string) error {
	embed := Embed{
		Title:       "Alert Resolved",
		Description: fmt.Sprintf("Alert `%s` has been resolved", alertID),
		Color:       3581519, // 绿色
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	message := Message{
		Username:  i.config.Username,
		AvatarURL: i.config.AvatarURL,
		Embeds:    []Embed{embed},
	}

	jsonData, _ := json.Marshal(message)
	req, _ := http.NewRequestWithContext(ctx, "POST", i.config.WebhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		i.logger.Info("Discord resolve notification sent",
			zap.String("alert_id", alertID),
		)
		return nil
	}

	return fmt.Errorf("failed to send resolve notification: status %d", resp.StatusCode)
}

// UpdateAlert Discord发送状态变更通知
func (i *Integration) UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error {
	if status == domain.AlertStatusResolved {
		return i.ResolveAlert(ctx, alertID)
	}

	embed := Embed{
		Title:       "Alert Status Updated",
		Description: fmt.Sprintf("Alert `%s` status changed to **%s**", alertID, status),
		Color:       i.getColorForStatus(status),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	message := Message{
		Username:  i.config.Username,
		AvatarURL: i.config.AvatarURL,
		Embeds:    []Embed{embed},
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
	embed := Embed{
		Title:       "Health Check",
		Description: "EdgeLink Alert Service is healthy",
		Color:       3581519, // 绿色
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	message := Message{
		Username:  i.config.Username,
		AvatarURL: i.config.AvatarURL,
		Embeds:    []Embed{embed},
	}

	jsonData, _ := json.Marshal(message)
	req, _ := http.NewRequestWithContext(ctx, "POST", i.config.WebhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("health check failed: status %d", resp.StatusCode)
}

// buildMessage 构建Discord消息
func (i *Integration) buildMessage(alert *domain.Alert) Message {
	color := i.config.ColorMap.MapColor(alert.Severity)

	// 构建字段
	fields := []EmbedField{
		{
			Name:   "Severity",
			Value:  string(alert.Severity),
			Inline: true,
		},
		{
			Name:   "Type",
			Value:  string(alert.Type),
			Inline: true,
		},
		{
			Name:   "Time",
			Value:  alert.CreatedAt.Format("2006-01-02 15:04:05"),
			Inline: true,
		},
	}

	if alert.DeviceID != nil {
		fields = append(fields, EmbedField{
			Name:   "Device ID",
			Value:  alert.DeviceID.String(),
			Inline: false,
		})
	}

	// 构建嵌入消息
	embed := Embed{
		Title:       alert.Title,
		Description: alert.Message,
		Color:       color,
		Fields:      fields,
		Footer: &EmbedFooter{
			Text: "EdgeLink Alert Service",
		},
		Timestamp: alert.CreatedAt.Format(time.RFC3339),
	}

	message := Message{
		Username:  i.config.Username,
		AvatarURL: i.config.AvatarURL,
		Embeds:    []Embed{embed},
	}

	return message
}

// getColorForStatus 根据状态获取颜色
func (i *Integration) getColorForStatus(status domain.AlertStatus) int {
	switch status {
	case domain.AlertStatusAcknowledged:
		return 16776960 // 黄色
	case domain.AlertStatusResolved:
		return 3581519 // 绿色
	default:
		return 8421504 // 灰色
	}
}

// Message Discord消息结构
type Message struct {
	Content   string  `json:"content,omitempty"`    // 纯文本内容
	Username  string  `json:"username,omitempty"`   // 覆盖用户名
	AvatarURL string  `json:"avatar_url,omitempty"` // 覆盖头像
	TTS       bool    `json:"tts,omitempty"`        // 文本转语音
	Embeds    []Embed `json:"embeds,omitempty"`     // 嵌入消息（最多10个）
}

// Embed Discord嵌入消息
type Embed struct {
	Title       string        `json:"title,omitempty"`       // 标题
	Description string        `json:"description,omitempty"` // 描述
	URL         string        `json:"url,omitempty"`         // 标题链接
	Color       int           `json:"color,omitempty"`       // 颜色（十进制）
	Fields      []EmbedField  `json:"fields,omitempty"`      // 字段列表（最多25个）
	Footer      *EmbedFooter  `json:"footer,omitempty"`      // 页脚
	Timestamp   string        `json:"timestamp,omitempty"`   // ISO8601时间戳
	Thumbnail   *EmbedMedia   `json:"thumbnail,omitempty"`   // 缩略图
	Image       *EmbedMedia   `json:"image,omitempty"`       // 图片
	Author      *EmbedAuthor  `json:"author,omitempty"`      // 作者
}

// EmbedField Discord嵌入字段
type EmbedField struct {
	Name   string `json:"name"`             // 字段名称
	Value  string `json:"value"`            // 字段值
	Inline bool   `json:"inline,omitempty"` // 是否内联（并排显示）
}

// EmbedFooter Discord嵌入页脚
type EmbedFooter struct {
	Text    string `json:"text"`                // 页脚文本
	IconURL string `json:"icon_url,omitempty"`  // 页脚图标URL
}

// EmbedMedia Discord嵌入媒体
type EmbedMedia struct {
	URL      string `json:"url"`                // 媒体URL
	ProxyURL string `json:"proxy_url,omitempty"` // 代理URL
	Height   int    `json:"height,omitempty"`   // 高度
	Width    int    `json:"width,omitempty"`    // 宽度
}

// EmbedAuthor Discord嵌入作者
type EmbedAuthor struct {
	Name    string `json:"name"`                // 作者名称
	URL     string `json:"url,omitempty"`       // 作者链接
	IconURL string `json:"icon_url,omitempty"`  // 作者图标URL
}
