package main

import (
	"context"
	"fmt"
	"time"

	"github.com/edgelink/backend/cmd/alert-service/internal/notifier"
	"github.com/edgelink/backend/internal/config"
	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 邮件通知系统使用示例

func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// 2. 创建日志器
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 3. 创建邮件通知器
	emailNotifier, err := notifier.NewEmailNotifier(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create email notifier", zap.Error(err))
	}
	defer emailNotifier.Stop()

	// 4. 创建示例告警
	alert := &domain.Alert{
		ID:        uuid.New(),
		Title:     "高CPU使用率告警",
		Message:   "设备CPU使用率已超过90%,持续时间超过5分钟,请立即检查",
		Severity:  domain.SeverityHigh,
		Type:      domain.AlertTypePerformance,
		Status:    domain.AlertStatusActive,
		CreatedAt: time.Now(),
	}

	deviceID := uuid.New()
	alert.DeviceID = &deviceID

	// 5. 发送告警邮件
	recipients := []string{
		"admin@example.com",
		"ops@example.com",
	}

	ctx := context.Background()
	if err := emailNotifier.SendAlert(ctx, alert, recipients); err != nil {
		logger.Error("Failed to send alert", zap.Error(err))
	} else {
		logger.Info("Alert email queued successfully")
	}

	// 6. 查询邮件发送统计
	time.Sleep(2 * time.Second) // 等待邮件发送
	stats := emailNotifier.GetStats()
	fmt.Printf("Email Stats:\n")
	fmt.Printf("  Total Sent: %d\n", stats.TotalSent)
	fmt.Printf("  Total Failed: %d\n", stats.TotalFailed)
	fmt.Printf("  Total Retried: %d\n", stats.TotalRetried)
	fmt.Printf("  Queue Length: %d\n", stats.QueueLength)
	if !stats.LastSentTime.IsZero() {
		fmt.Printf("  Last Sent: %s\n", stats.LastSentTime.Format(time.RFC3339))
	}
	if stats.LastError != "" {
		fmt.Printf("  Last Error: %s\n", stats.LastError)
	}

	// 7. 等待所有邮件发送完成
	time.Sleep(5 * time.Second)
}

// 示例2: 批量发送告警邮件
func exampleBatchSend() {
	cfg, _ := config.Load()
	logger, _ := zap.NewProduction()
	emailNotifier, _ := notifier.NewEmailNotifier(cfg, logger)
	defer emailNotifier.Stop()

	ctx := context.Background()
	alerts := generateAlerts(10) // 生成10条告警

	for _, alert := range alerts {
		recipients := []string{"admin@example.com"}
		if err := emailNotifier.SendAlert(ctx, alert, recipients); err != nil {
			logger.Error("Failed to queue alert", zap.Error(err))
		}
	}

	// 等待所有邮件发送
	time.Sleep(10 * time.Second)

	stats := emailNotifier.GetStats()
	logger.Info("Batch send completed",
		zap.Int64("sent", stats.TotalSent),
		zap.Int64("failed", stats.TotalFailed),
	)
}

// 示例3: 自定义邮件内容(不使用告警)
func exampleCustomEmail() {
	// 注意: 这需要直接使用Provider接口
	cfg, _ := config.Load()
	logger, _ := zap.NewProduction()

	// 创建SMTP提供商
	provider, err := notifier.NewSMTPProvider(&cfg.Email, logger)
	if err != nil {
		logger.Fatal("Failed to create provider", zap.Error(err))
	}

	// 发送自定义邮件
	msg := &notifier.EmailMessage{
		To:       []string{"user@example.com"},
		Subject:  "EdgeLink系统通知",
		HTMLBody: "<h1>您好</h1><p>这是一封测试邮件</p>",
		TextBody: "您好\n\n这是一封测试邮件",
	}

	ctx := context.Background()
	if err := provider.Send(ctx, msg); err != nil {
		logger.Error("Failed to send custom email", zap.Error(err))
	}
}

// 示例4: 配置不同的邮件提供商
func exampleProviderSwitch() {
	// SMTP配置
	smtpConfig := &config.EmailConfig{
		Provider: "smtp",
		SMTP: config.SMTPConfig{
			Host:        "smtp.gmail.com",
			Port:        587,
			Username:    "your-email@gmail.com",
			Password:    "your-app-password",
			UseStartTLS: true,
		},
		FromAddress: "noreply@edgelink.com",
		FromName:    "EdgeLink",
		QueueSize:   1000,
		MaxRetries:  3,
		RetryDelay:  5 * time.Second,
		RateLimit:   100,
		RatePeriod:  time.Minute,
	}

	// SendGrid配置(示例)
	sendgridConfig := &config.EmailConfig{
		Provider: "sendgrid",
		SendGrid: config.SendGridConfig{
			APIKey:      "SG.xxxxx",
			SandboxMode: false,
		},
		FromAddress: "noreply@edgelink.com",
		QueueSize:   1000,
	}

	logger, _ := zap.NewProduction()

	// 根据环境选择提供商
	environment := "production"
	var emailCfg *config.EmailConfig
	if environment == "production" {
		emailCfg = sendgridConfig
	} else {
		emailCfg = smtpConfig
	}

	cfg := &config.Config{Email: *emailCfg}
	emailNotifier, _ := notifier.NewEmailNotifier(cfg, logger)
	defer emailNotifier.Stop()

	logger.Info("Email notifier initialized with provider",
		zap.String("provider", emailCfg.Provider),
	)
}

// 辅助函数: 生成测试告警
func generateAlerts(count int) []*domain.Alert {
	alerts := make([]*domain.Alert, count)
	severities := []domain.Severity{
		domain.SeverityCritical,
		domain.SeverityHigh,
		domain.SeverityMedium,
		domain.SeverityLow,
	}

	for i := 0; i < count; i++ {
		alerts[i] = &domain.Alert{
			ID:        uuid.New(),
			Title:     fmt.Sprintf("告警 #%d", i+1),
			Message:   fmt.Sprintf("这是第%d条测试告警", i+1),
			Severity:  severities[i%len(severities)],
			Type:      domain.AlertTypePerformance,
			Status:    domain.AlertStatusActive,
			CreatedAt: time.Now(),
		}
	}

	return alerts
}
