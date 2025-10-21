package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgelink/backend/cmd/alert-service/internal/checker"
	"github.com/edgelink/backend/cmd/alert-service/internal/deduplication"
	"github.com/edgelink/backend/cmd/alert-service/internal/generator"
	"github.com/edgelink/backend/cmd/alert-service/internal/notifier"
	"github.com/edgelink/backend/cmd/alert-service/internal/scheduler"
	"github.com/edgelink/backend/internal/config"
	"github.com/edgelink/backend/internal/database"
	"github.com/edgelink/backend/internal/logger"
	"github.com/edgelink/backend/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		// 配置模块
		fx.Provide(
			config.LoadConfig,
			logger.NewLogger,
		),

		// 数据库模块
		fx.Provide(
			database.NewPostgresDB,
			NewRedisClient,
		),

		// 仓储层
		fx.Provide(
			repository.NewDeviceRepository,
			repository.NewAlertRepository,
			repository.NewSessionRepository,
		),

		// 告警服务组件
		fx.Provide(
			NewDeduplicationManager,
			NewSchedulerConfig,
			checker.NewThresholdChecker,
			generator.NewAlertGenerator,
			func(cfg *config.Config, logger *zap.Logger) (*notifier.EmailNotifier, error) {
				return notifier.NewEmailNotifier(cfg, logger)
			},
			notifier.NewWebhookNotifier,
			scheduler.NewNotificationScheduler,
		),

		// 启动告警服务
		fx.Invoke(runAlertService),
	)

	app.Run()
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

// NewDeduplicationManager 创建去重管理器
func NewDeduplicationManager(redisClient *redis.Client, cfg *config.Config, logger *zap.Logger) *deduplication.Manager {
	dedupeConfig := &deduplication.Config{
		DedupeWindow:        cfg.Alert.DedupeWindow,
		SilentPeriod:        cfg.Alert.SilentPeriod,
		EscalationThreshold: cfg.Alert.EscalationThreshold,
		LockTimeout:         cfg.Alert.LockTimeout,
	}
	return deduplication.NewManager(redisClient, dedupeConfig, logger)
}

// NewSchedulerConfig 创建调度器配置
func NewSchedulerConfig(cfg *config.Config) scheduler.SchedulerConfig {
	return scheduler.SchedulerConfig{
		RulesFile:       "alert-rules.yaml",
		EnableRuleEngine: true,
		EnableHotReload: true,
		ReloadInterval:  5 * time.Minute,
	}
}


// runAlertService 运行告警服务
func runAlertService(
	lifecycle fx.Lifecycle,
	log *zap.Logger,
	cfg *config.Config,
	redisClient *redis.Client,
	thresholdChecker *checker.ThresholdChecker,
	alertGenerator *generator.AlertGenerator,
	notificationScheduler *scheduler.NotificationScheduler,
) {
	ctx, cancel := context.WithCancel(context.Background())

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info("Starting Alert Service")

			// 启动告警检查循环
			go runCheckLoop(ctx, log, cfg, thresholdChecker, alertGenerator, notificationScheduler)

			// 启动通知调度器
			go notificationScheduler.Start(ctx)

			return nil
		},
		OnStop: func(context.Context) error {
			log.Info("Shutting down Alert Service")
			cancel()
			return nil
		},
	})

	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info("Received shutdown signal")
		cancel()
	}()
}

// runCheckLoop 运行检查循环
func runCheckLoop(
	ctx context.Context,
	log *zap.Logger,
	cfg *config.Config,
	thresholdChecker *checker.ThresholdChecker,
	alertGenerator *generator.AlertGenerator,
	notificationScheduler *scheduler.NotificationScheduler,
) {
	checkInterval := cfg.Alert.CheckInterval
	if checkInterval == 0 {
		checkInterval = 1 * time.Minute
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Info("Alert check loop started", zap.Duration("interval", checkInterval))

	for {
		select {
		case <-ctx.Done():
			log.Info("Check loop shutting down")
			return

		case <-ticker.C:
			// 执行健康检查
			issues := thresholdChecker.CheckAll(ctx)

			// 为每个问题生成告警
			for _, issue := range issues {
				alert, err := alertGenerator.GenerateAlert(ctx, issue)
				if err != nil {
					log.Error("Failed to generate alert",
						zap.Error(err),
						zap.String("issue_type", issue.Type),
					)
					continue
				}

				// 如果告警为nil（在静默期内），跳过通知
				if alert == nil {
					continue
				}

				// 调度通知
				if err := notificationScheduler.Schedule(ctx, alert); err != nil {
					log.Error("Failed to schedule notification",
						zap.Error(err),
						zap.String("alert_id", alert.ID.String()),
					)
				}
			}

			if len(issues) > 0 {
				log.Info("Health check completed",
					zap.Int("issues_found", len(issues)),
				)
			}
		}
	}
}
