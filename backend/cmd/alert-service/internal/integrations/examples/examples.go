package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/edgelink/backend/cmd/alert-service/internal/config"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations"
	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Example 展示如何使用集成管理器
func Example() {
	// 创建logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 加载配置
	cfg, err := loadIntegrationsConfig("config/integrations.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// 创建集成工厂
	factory := integrations.NewFactory(logger)

	// 创建集成管理器
	manager, err := factory.CreateManager(cfg)
	if err != nil {
		logger.Fatal("Failed to create integration manager", zap.Error(err))
	}

	// 执行健康检查
	ctx := context.Background()
	healthResults := factory.HealthCheckAll(ctx, manager)
	fmt.Printf("Health check results: %+v\n", healthResults)

	// 创建测试告警
	alert := createTestAlert()

	// 发送告警
	if err := manager.SendAlert(ctx, alert); err != nil {
		logger.Error("Failed to send alert", zap.Error(err))
	} else {
		logger.Info("Alert sent successfully")
	}

	// 等待一段时间后解决告警
	time.Sleep(5 * time.Second)
	if err := manager.ResolveAlert(ctx, alert.ID.String()); err != nil {
		logger.Error("Failed to resolve alert", zap.Error(err))
	} else {
		logger.Info("Alert resolved successfully")
	}

	// 获取指标
	metrics := manager.GetMetrics()
	for name, metric := range metrics {
		fmt.Printf("Integration: %s, Total: %d, Success: %d, Failure: %d, AvgTime: %v\n",
			name, metric.TotalSent, metric.SuccessCount, metric.FailureCount, metric.AvgResponseTime)
	}
}

// ExampleWithManualRegistration 手动注册集成示例
func ExampleWithManualRegistration() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	manager := integrations.NewManager(logger)

	// 手动创建和注册Slack集成
	slackConfig := &config.SlackConfig{
		Enabled:    true,
		Priority:   1,
		WebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		Username:   "Test Bot",
		IconEmoji:  ":test_tube:",
	}

	slackIntegration := integrations.NewFactory(logger).CreateIntegration("slack", slackConfig)
	manager.Register(slackIntegration, slackConfig.ToSlackConfig())

	// 发送测试告警
	ctx := context.Background()
	alert := createTestAlert()
	manager.SendAlert(ctx, alert)
}

// ExampleErrorHandling 错误处理示例
func ExampleErrorHandling() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	manager := integrations.NewManager(logger)

	// 配置错误的集成（用于测试错误处理）
	badConfig := &config.SlackConfig{
		Enabled:    true,
		Priority:   1,
		WebhookURL: "https://invalid-webhook-url.example.com/webhook",
	}

	factory := integrations.NewFactory(logger)
	slackIntegration, _ := factory.CreateIntegration("slack", badConfig)
	manager.Register(slackIntegration, badConfig.ToSlackConfig())

	// 尝试发送告警（会失败并重试）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	alert := createTestAlert()
	err := manager.SendAlert(ctx, alert)

	// 检查错误类型
	if integrationErr, ok := err.(*integrations.IntegrationError); ok {
		logger.Error("Integration error",
			zap.String("integration", integrationErr.Integration),
			zap.String("operation", integrationErr.Operation),
			zap.Bool("retryable", integrationErr.Retryable),
			zap.Error(integrationErr.Err),
		)
	}
}

// ExamplePriorityRouting 优先级路由示例
func ExamplePriorityRouting() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建配置（按优先级排序）
	cfg := &config.IntegrationsConfig{
		PagerDuty: &config.PagerDutyConfig{
			Enabled:        true,
			Priority:       1, // 最高优先级
			IntegrationKey: os.Getenv("PAGERDUTY_INTEGRATION_KEY"),
		},
		Slack: &config.SlackConfig{
			Enabled:    true,
			Priority:   2, // 次要优先级（备用）
			WebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		},
		Discord: &config.DiscordConfig{
			Enabled:    true,
			Priority:   3, // 最低优先级（最后备用）
			WebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
		},
	}

	factory := integrations.NewFactory(logger)
	manager, _ := factory.CreateManager(cfg)

	// 发送告警（会同时发送到所有平台，但日志会按优先级排序）
	ctx := context.Background()
	alert := createTestAlert()
	manager.SendAlert(ctx, alert)
}

// loadIntegrationsConfig 加载集成配置
func loadIntegrationsConfig(path string) (*config.IntegrationsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg struct {
		Integrations *config.IntegrationsConfig `yaml:"integrations"`
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg.Integrations, nil
}

// createTestAlert 创建测试告警
func createTestAlert() *domain.Alert {
	deviceID := uuid.New()
	now := time.Now()

	return &domain.Alert{
		ID:        uuid.New(),
		DeviceID:  &deviceID,
		Severity:  domain.SeverityCritical,
		Type:      domain.AlertTypeDeviceOffline,
		Title:     "Device Offline Alert",
		Message:   "Device test-device-01 has been offline for 5 minutes",
		Status:    domain.AlertStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata: domain.JSONB{
			"device_name":    "test-device-01",
			"last_seen":      now.Add(-5 * time.Minute).Format(time.RFC3339),
			"location":       "us-west-2",
			"impact_level":   "high",
			"affected_users": 150,
		},
	}
}

// TestIntegration 集成测试工具
func TestIntegration(integrationType string) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	ctx := context.Background()
	factory := integrations.NewFactory(logger)

	var integration integrations.Integration
	var err error

	// 根据类型创建集成
	switch integrationType {
	case "pagerduty":
		cfg := &config.PagerDutyConfig{
			Enabled:        true,
			Priority:       1,
			IntegrationKey: os.Getenv("PAGERDUTY_INTEGRATION_KEY"),
		}
		integration, err = factory.CreateIntegration("pagerduty", cfg.ToPagerDutyConfig())

	case "slack":
		cfg := &config.SlackConfig{
			Enabled:    true,
			Priority:   1,
			WebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		}
		integration, err = factory.CreateIntegration("slack", cfg.ToSlackConfig())

	case "discord":
		cfg := &config.DiscordConfig{
			Enabled:    true,
			Priority:   1,
			WebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
		}
		integration, err = factory.CreateIntegration("discord", cfg.ToDiscordConfig())

	default:
		logger.Fatal("Unsupported integration type", zap.String("type", integrationType))
	}

	if err != nil {
		logger.Fatal("Failed to create integration", zap.Error(err))
	}

	// 执行健康检查
	logger.Info("Running health check...")
	if err := integration.HealthCheck(ctx); err != nil {
		logger.Error("Health check failed", zap.Error(err))
	} else {
		logger.Info("Health check passed")
	}

	// 发送测试告警
	logger.Info("Sending test alert...")
	alert := createTestAlert()
	if err := integration.SendAlert(ctx, alert); err != nil {
		logger.Error("Failed to send alert", zap.Error(err))
	} else {
		logger.Info("Alert sent successfully")
	}

	// 等待后解决
	time.Sleep(3 * time.Second)
	logger.Info("Resolving alert...")
	if err := integration.ResolveAlert(ctx, alert.ID.String()); err != nil {
		logger.Error("Failed to resolve alert", zap.Error(err))
	} else {
		logger.Info("Alert resolved successfully")
	}
}

func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run examples.go example           - Run basic example")
		fmt.Println("  go run examples.go test <type>       - Test specific integration")
		fmt.Println("  go run examples.go priority          - Test priority routing")
		fmt.Println("  go run examples.go error-handling    - Test error handling")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "example":
		Example()
	case "test":
		if len(os.Args) < 3 {
			fmt.Println("Please specify integration type: pagerduty, slack, discord, etc.")
			os.Exit(1)
		}
		TestIntegration(os.Args[2])
	case "priority":
		ExamplePriorityRouting()
	case "error-handling":
		ExampleErrorHandling()
	case "manual":
		ExampleWithManualRegistration()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
