package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/edgelink/backend/cmd/alert-service/internal/notifier"
	"github.com/edgelink/backend/internal/domain"
	"go.uber.org/zap"
)

// Executor 动作执行器
type Executor struct {
	emailNotifier   *notifier.EmailNotifier
	webhookNotifier *notifier.WebhookNotifier
	httpClient      *http.Client
	logger          *zap.Logger
}

// NewExecutor 创建动作执行器
func NewExecutor(
	emailNotifier *notifier.EmailNotifier,
	webhookNotifier *notifier.WebhookNotifier,
	logger *zap.Logger,
) *Executor {
	return &Executor{
		emailNotifier:   emailNotifier,
		webhookNotifier: webhookNotifier,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Execute 执行动作
func (e *Executor) Execute(ctx context.Context, action *Action, execCtx *ExecutionContext) *ExecutionResult {
	startTime := time.Now()
	result := &ExecutionResult{
		RuleID:     execCtx.Rule.ID,
		ActionType: action.Type,
		ExecutedAt: startTime,
		Metadata:   make(map[string]interface{}),
	}

	// 检查动作是否启用
	if !action.Enabled {
		result.Success = false
		result.Error = fmt.Errorf("action is disabled")
		result.Duration = time.Since(startTime)
		return result
	}

	// 执行对应类型的动作
	var err error
	switch action.Type {
	case ActionTypeEmail:
		err = e.executeEmail(ctx, action, execCtx)
	case ActionTypeWebhook:
		err = e.executeWebhook(ctx, action, execCtx)
	case ActionTypeSlack:
		err = e.executeSlack(ctx, action, execCtx)
	case ActionTypePagerDuty:
		err = e.executePagerDuty(ctx, action, execCtx)
	case ActionTypeDingTalk:
		err = e.executeDingTalk(ctx, action, execCtx)
	case ActionTypeWeChat:
		err = e.executeWeChat(ctx, action, execCtx)
	case ActionTypeTelegram:
		err = e.executeTelegram(ctx, action, execCtx)
	case ActionTypeCustom:
		err = e.executeCustom(ctx, action, execCtx)
	default:
		err = fmt.Errorf("unsupported action type: %s", action.Type)
	}

	result.Success = (err == nil)
	result.Error = err
	result.Duration = time.Since(startTime)

	if err != nil {
		e.logger.Error("Action execution failed",
			zap.String("rule_id", execCtx.Rule.ID),
			zap.String("action_type", string(action.Type)),
			zap.Error(err),
		)
	} else {
		e.logger.Info("Action executed successfully",
			zap.String("rule_id", execCtx.Rule.ID),
			zap.String("action_type", string(action.Type)),
			zap.Duration("duration", result.Duration),
		)
	}

	return result
}

// executeEmail 执行邮件通知
func (e *Executor) executeEmail(ctx context.Context, action *Action, execCtx *ExecutionContext) error {
	recipients, ok := action.Config["recipients"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid recipients config")
	}

	// 转换为字符串数组
	recipientList := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if email, ok := r.(string); ok {
			recipientList = append(recipientList, email)
		}
	}

	if len(recipientList) == 0 {
		return fmt.Errorf("no valid recipients")
	}

	return e.emailNotifier.SendAlert(ctx, execCtx.Alert, recipientList)
}

// executeWebhook 执行Webhook通知
func (e *Executor) executeWebhook(ctx context.Context, action *Action, execCtx *ExecutionContext) error {
	url, ok := action.Config["url"].(string)
	if !ok {
		return fmt.Errorf("invalid webhook url config")
	}

	return e.webhookNotifier.SendAlert(ctx, execCtx.Alert, url)
}

// executeSlack 执行Slack通知
func (e *Executor) executeSlack(ctx context.Context, action *Action, execCtx *ExecutionContext) error {
	webhookURL, ok := action.Config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("invalid slack webhook_url config")
	}

	channel, _ := action.Config["channel"].(string)
	username, _ := action.Config["username"].(string)
	if username == "" {
		username = "EdgeLink Alerts"
	}

	// 构建Slack消息
	message := map[string]interface{}{
		"username": username,
		"text":     fmt.Sprintf("*%s*", execCtx.Alert.Title),
		"attachments": []map[string]interface{}{
			{
				"color": e.getSeverityColor(execCtx.Alert.Severity),
				"fields": []map[string]interface{}{
					{
						"title": "Severity",
						"value": string(execCtx.Alert.Severity),
						"short": true,
					},
					{
						"title": "Type",
						"value": string(execCtx.Alert.Type),
						"short": true,
					},
					{
						"title": "Message",
						"value": execCtx.Alert.Message,
						"short": false,
					},
					{
						"title": "Time",
						"value": execCtx.Alert.CreatedAt.Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}

	if channel != "" {
		message["channel"] = channel
	}

	return e.sendHTTPJSON(ctx, webhookURL, message)
}

// executePagerDuty 执行PagerDuty通知
func (e *Executor) executePagerDuty(ctx context.Context, action *Action, execCtx *ExecutionContext) error {
	serviceKey, ok := action.Config["service_key"].(string)
	if !ok {
		return fmt.Errorf("invalid pagerduty service_key config")
	}

	// 构建PagerDuty事件
	event := map[string]interface{}{
		"routing_key":  serviceKey,
		"event_action": "trigger",
		"payload": map[string]interface{}{
			"summary":   execCtx.Alert.Title,
			"severity":  e.mapPagerDutySeverity(execCtx.Alert.Severity),
			"source":    "EdgeLink",
			"timestamp": execCtx.Alert.CreatedAt.Format(time.RFC3339),
			"custom_details": map[string]interface{}{
				"type":    string(execCtx.Alert.Type),
				"message": execCtx.Alert.Message,
			},
		},
		"dedup_key": execCtx.Alert.ID.String(),
	}

	return e.sendHTTPJSON(ctx, "https://events.pagerduty.com/v2/enqueue", event)
}

// executeDingTalk 执行钉钉通知
func (e *Executor) executeDingTalk(ctx context.Context, action *Action, execCtx *ExecutionContext) error {
	webhookURL, ok := action.Config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("invalid dingtalk webhook_url config")
	}

	// 构建钉钉消息
	message := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": execCtx.Alert.Title,
			"text": fmt.Sprintf(
				"## %s\n\n"+
					"**严重程度**: %s\n\n"+
					"**类型**: %s\n\n"+
					"**消息**: %s\n\n"+
					"**时间**: %s",
				execCtx.Alert.Title,
				execCtx.Alert.Severity,
				execCtx.Alert.Type,
				execCtx.Alert.Message,
				execCtx.Alert.CreatedAt.Format("2006-01-02 15:04:05"),
			),
		},
	}

	// 支持@人
	if atMobiles, ok := action.Config["at_mobiles"].([]interface{}); ok && len(atMobiles) > 0 {
		mobiles := make([]string, 0, len(atMobiles))
		for _, m := range atMobiles {
			if mobile, ok := m.(string); ok {
				mobiles = append(mobiles, mobile)
			}
		}
		message["at"] = map[string]interface{}{
			"atMobiles": mobiles,
			"isAtAll":   false,
		}
	}

	return e.sendHTTPJSON(ctx, webhookURL, message)
}

// executeWeChat 执行企业微信通知
func (e *Executor) executeWeChat(ctx context.Context, action *Action, execCtx *ExecutionContext) error {
	webhookURL, ok := action.Config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("invalid wechat webhook_url config")
	}

	// 构建企业微信消息
	message := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": fmt.Sprintf(
				"## %s\n"+
					">严重程度: <font color=\"%s\">%s</font>\n"+
					">类型: %s\n"+
					">消息: %s\n"+
					">时间: %s",
				execCtx.Alert.Title,
				e.getSeverityColor(execCtx.Alert.Severity),
				execCtx.Alert.Severity,
				execCtx.Alert.Type,
				execCtx.Alert.Message,
				execCtx.Alert.CreatedAt.Format("2006-01-02 15:04:05"),
			),
		},
	}

	return e.sendHTTPJSON(ctx, webhookURL, message)
}

// executeTelegram 执行Telegram通知
func (e *Executor) executeTelegram(ctx context.Context, action *Action, execCtx *ExecutionContext) error {
	botToken, ok := action.Config["bot_token"].(string)
	if !ok {
		return fmt.Errorf("invalid telegram bot_token config")
	}

	chatID, ok := action.Config["chat_id"]
	if !ok {
		return fmt.Errorf("invalid telegram chat_id config")
	}

	// 构建Telegram消息
	text := fmt.Sprintf(
		"*%s*\n\n"+
			"Severity: %s\n"+
			"Type: %s\n"+
			"Message: %s\n"+
			"Time: %s",
		execCtx.Alert.Title,
		execCtx.Alert.Severity,
		execCtx.Alert.Type,
		execCtx.Alert.Message,
		execCtx.Alert.CreatedAt.Format(time.RFC3339),
	)

	message := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	return e.sendHTTPJSON(ctx, url, message)
}

// executeCustom 执行自定义动作
func (e *Executor) executeCustom(ctx context.Context, action *Action, execCtx *ExecutionContext) error {
	url, ok := action.Config["url"].(string)
	if !ok {
		return fmt.Errorf("invalid custom url config")
	}

	method, _ := action.Config["method"].(string)
	if method == "" {
		method = "POST"
	}

	// 构建自定义请求体
	body := map[string]interface{}{
		"alert": execCtx.Alert,
		"rule":  execCtx.Rule.ID,
	}

	// 允许自定义请求体模板
	if customBody, ok := action.Config["body"].(map[string]interface{}); ok {
		body = customBody
	}

	return e.sendHTTPJSON(ctx, url, body)
}

// sendHTTPJSON 发送HTTP JSON请求
func (e *Executor) sendHTTPJSON(ctx context.Context, url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getSeverityColor 获取严重程度对应的颜色
func (e *Executor) getSeverityColor(severity domain.Severity) string {
	switch severity {
	case domain.SeverityCritical:
		return "danger"
	case domain.SeverityHigh:
		return "warning"
	case domain.SeverityMedium:
		return "info"
	case domain.SeverityLow:
		return "comment"
	default:
		return "info"
	}
}

// mapPagerDutySeverity 映射到PagerDuty严重程度
func (e *Executor) mapPagerDutySeverity(severity domain.Severity) string {
	switch severity {
	case domain.SeverityCritical:
		return "critical"
	case domain.SeverityHigh:
		return "error"
	case domain.SeverityMedium:
		return "warning"
	case domain.SeverityLow:
		return "info"
	default:
		return "info"
	}
}

// ExecuteWithRetry 带重试的执行
func (e *Executor) ExecuteWithRetry(ctx context.Context, action *Action, execCtx *ExecutionContext) *ExecutionResult {
	var result *ExecutionResult

	retryPolicy := action.RetryPolicy
	if retryPolicy == nil {
		// 使用默认重试策略
		retryPolicy = &RetryPolicy{
			MaxRetries:  3,
			RetryDelay:  5 * time.Second,
			BackoffRate: 2.0,
		}
	}

	delay := retryPolicy.RetryDelay
	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		execCtx.PreviousTries = attempt
		result = e.Execute(ctx, action, execCtx)

		if result.Success {
			return result
		}

		// 最后一次尝试失败，不再重试
		if attempt == retryPolicy.MaxRetries {
			break
		}

		e.logger.Warn("Action execution failed, retrying",
			zap.String("rule_id", execCtx.Rule.ID),
			zap.String("action_type", string(action.Type)),
			zap.Int("attempt", attempt+1),
			zap.Duration("retry_after", delay),
		)

		// 等待后重试
		select {
		case <-ctx.Done():
			result.Error = ctx.Err()
			return result
		case <-time.After(delay):
			// 指数退避
			delay = time.Duration(float64(delay) * retryPolicy.BackoffRate)
		}
	}

	return result
}
