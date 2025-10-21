package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"go.uber.org/zap"
)

// WebhookPayload Webhook负载
type WebhookPayload struct {
	AlertID      string                 `json:"alert_id"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Severity     string                 `json:"severity"`
	AlertType    string                 `json:"alert_type"`
	DeviceID     *string                `json:"device_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    string                 `json:"created_at"`
	Timestamp    string                 `json:"timestamp"`
}

// WebhookNotifier Webhook通知器
type WebhookNotifier struct {
	httpClient *http.Client
	logger     *zap.Logger
	maxRetries int
	retryDelay time.Duration
}

// NewWebhookNotifier 创建Webhook通知器
func NewWebhookNotifier(logger *zap.Logger) *WebhookNotifier {
	return &WebhookNotifier{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:     logger,
		maxRetries: 3,
		retryDelay: 2 * time.Second,
	}
}

// SendAlert 发送告警到Webhook
func (wn *WebhookNotifier) SendAlert(ctx context.Context, alert *domain.Alert, webhookURL string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	// 构建Webhook负载
	payload := wn.buildPayload(alert)

	// 序列化为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 带重试的发送
	var lastErr error
	for attempt := 0; attempt <= wn.maxRetries; attempt++ {
		if attempt > 0 {
			// 重试前等待
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wn.retryDelay * time.Duration(attempt)):
			}

			wn.logger.Info("Retrying webhook send",
				zap.Int("attempt", attempt),
				zap.String("alert_id", alert.ID.String()),
			)
		}

		// 发送HTTP POST请求
		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "EdgeLink-AlertService/1.0")

		resp, err := wn.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			continue
		}

		// 检查响应状态
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			wn.logger.Info("Webhook sent successfully",
				zap.String("alert_id", alert.ID.String()),
				zap.String("url", webhookURL),
				zap.Int("status_code", resp.StatusCode),
			)
			return nil
		}

		resp.Body.Close()
		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return fmt.Errorf("webhook send failed after %d attempts: %w", wn.maxRetries+1, lastErr)
}

// buildPayload 构建Webhook负载
func (wn *WebhookNotifier) buildPayload(alert *domain.Alert) WebhookPayload {
	payload := WebhookPayload{
		AlertID:   alert.ID.String(),
		Title:     alert.Title,
		Message:   alert.Message,
		Severity:  string(alert.Severity),
		AlertType: string(alert.Type),
		CreatedAt: alert.CreatedAt.Format(time.RFC3339),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if alert.DeviceID != nil {
		deviceIDStr := alert.DeviceID.String()
		payload.DeviceID = &deviceIDStr
	}

	if alert.Metadata != nil {
		// 转换JSONB到map
		metadata := make(map[string]interface{})
		for k, v := range alert.Metadata {
			metadata[k] = v
		}
		payload.Metadata = metadata
	}

	return payload
}

// SendBatch 批量发送告警到Webhook
func (wn *WebhookNotifier) SendBatch(ctx context.Context, alerts []*domain.Alert, webhookURL string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	// 构建批量负载
	payloads := make([]WebhookPayload, len(alerts))
	for i, alert := range alerts {
		payloads[i] = wn.buildPayload(alert)
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(map[string]interface{}{
		"alerts": payloads,
		"count":  len(alerts),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal batch payload: %w", err)
	}

	// 发送HTTP POST请求
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "EdgeLink-AlertService/1.0")

	resp, err := wn.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	wn.logger.Info("Batch webhook sent successfully",
		zap.Int("alert_count", len(alerts)),
		zap.String("url", webhookURL),
	)

	return nil
}
