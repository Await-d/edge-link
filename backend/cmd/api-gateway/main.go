package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgelink/backend/cmd/api-gateway/internal/handler"
	"github.com/edgelink/backend/cmd/api-gateway/internal/router"
	"github.com/edgelink/backend/internal/audit"
	"github.com/edgelink/backend/internal/service"
	"github.com/edgelink/backend/internal/auth"
	"github.com/edgelink/backend/internal/config"
	"github.com/edgelink/backend/internal/database"
	"github.com/edgelink/backend/internal/logger"
	"github.com/edgelink/backend/internal/repository"
	"github.com/gin-gonic/gin"
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
		),

		// 仓储层
		fx.Provide(
			repository.NewOrganizationRepository,
			repository.NewVirtualNetworkRepository,
			repository.NewDeviceRepository,
			repository.NewDeviceKeyRepository,
			repository.NewPreSharedKeyRepository,
			repository.NewSessionRepository,
			repository.NewAlertRepository,
			repository.NewAuditLogRepository,
			repository.NewAdminUserRepository,
		),

		// 认证模块
		fx.Provide(
			auth.NewPSKAuthenticator,
		),

		// 服务层
		fx.Provide(
			service.NewDeviceService,
			service.NewTopologyService,
		),

		// 处理器层
		fx.Provide(
			handler.NewDeviceHandler,
			handler.NewAdminHandler,
		)

		// WebSocket处理器
		fx.Provide(
			websocket.NewWebSocketHandler,
		),

		// 审计中间件
		fx.Provide(
			audit.NewAuditMiddleware,
		),

		// HTTP路由器
		fx.Provide(
			router.SetupRouter,
		),

		// HTTP服务器
		fx.Invoke(runHTTPServer),

		// WebSocket广播器
		fx.Invoke(startWebSocketBroadcaster),
	)

	app.Run()
}

// runHTTPServer 启动HTTP服务器
func runHTTPServer(
	lifecycle fx.Lifecycle,
	log *zap.Logger,
	cfg *config.Config,
	router *gin.Engine,
) {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("Starting API Gateway",
				zap.Int("port", cfg.Server.Port),
			)

			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatal("Failed to start HTTP server", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Shutting down API Gateway")

			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Error("Failed to gracefully shutdown server", zap.Error(err))
				return err
			}

			log.Info("API Gateway stopped")
			return nil
		},
	})

	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info("Received shutdown signal")
	}()
}

// startWebSocketBroadcaster 启动WebSocket广播器
func startWebSocketBroadcaster(
	lifecycle fx.Lifecycle,
	log *zap.Logger,
	wsHandler *websocket.WebSocketHandler,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("Starting WebSocket broadcaster")
			// WebSocket广播器已经在NewWebSocketHandler中启动
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Stopping WebSocket broadcaster")
			wsHandler.Shutdown()
			return nil
		},
	})
}
