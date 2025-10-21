package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/edgelink/backend/cmd/background-worker/internal/tasks"
	"github.com/edgelink/backend/internal/config"
	"github.com/edgelink/backend/internal/database"
	"github.com/edgelink/backend/internal/logger"
	"github.com/edgelink/backend/internal/repository"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
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
			repository.NewDeviceKeyRepository,
		),

		// 后台任务
		fx.Provide(
			tasks.NewDeviceHealthTask,
			tasks.NewPerformanceMonitorTask,
			tasks.NewSecurityMonitorTask,
			tasks.NewKeyExpiryTask,
		),

		// 启动后台工作器
		fx.Invoke(runBackgroundWorker),
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

// runBackgroundWorker 运行后台工作器
func runBackgroundWorker(
	lifecycle fx.Lifecycle,
	log *zap.Logger,
	cfg *config.Config,
	deviceHealthTask *tasks.DeviceHealthTask,
	performanceMonitorTask *tasks.PerformanceMonitorTask,
	securityMonitorTask *tasks.SecurityMonitorTask,
	keyExpiryTask *tasks.KeyExpiryTask,
) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建cron调度器
	c := cron.New()

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info("Starting Background Worker")

			// 调度任务
			// 设备健康检查 - 每分钟
			c.AddFunc("@every 1m", func() {
				if err := deviceHealthTask.Run(ctx); err != nil {
					log.Error("Device health task failed", zap.Error(err))
				}
			})

			// 性能监控 - 每5分钟
			c.AddFunc("@every 5m", func() {
				if err := performanceMonitorTask.Run(ctx); err != nil {
					log.Error("Performance monitor task failed", zap.Error(err))
				}
			})

			// 安全监控 - 每分钟
			c.AddFunc("@every 1m", func() {
				if err := securityMonitorTask.Run(ctx); err != nil {
					log.Error("Security monitor task failed", zap.Error(err))
				}
			})

			// 密钥过期检查 - 每天凌晨2点
			c.AddFunc("0 2 * * *", func() {
				if err := keyExpiryTask.Run(ctx); err != nil {
					log.Error("Key expiry task failed", zap.Error(err))
				}
			})

			// 启动调度器
			c.Start()

			log.Info("Background worker started with scheduled tasks")

			return nil
		},
		OnStop: func(context.Context) error {
			log.Info("Shutting down Background Worker")
			cancel()
			c.Stop()
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
