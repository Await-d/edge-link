package router

import (
	"github.com/edgelink/backend/cmd/api-gateway/internal/handler"
	"github.com/edgelink/backend/internal/audit"
	"github.com/gin-gonic/gin"
)

// SetupRouter 配置API网关路由
func SetupRouter(
	deviceHandler *handler.DeviceHandler,
	adminHandler *handler.AdminHandler,
	auditMiddleware *audit.AuditMiddleware,
) *gin.Engine {
	// 创建Gin引擎
	r := gin.Default()

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// API v1路由组
	v1 := r.Group("/api/v1")
	{
		// 设备相关端点
		device := v1.Group("/device")
		{
			// POST /api/v1/device/register - 注册新设备
			device.POST("/register", deviceHandler.RegisterDevice)

			// GET /api/v1/device/{device_id}/config - 获取设备配置
			device.GET("/:device_id/config", deviceHandler.GetDeviceConfig)

			// POST /api/v1/device/{device_id}/metrics - 提交设备指标
			device.POST("/:device_id/metrics", deviceHandler.SubmitDeviceMetrics)
		}

		// 管理员端点
		admin := v1.Group("/admin")
		admin.Use(auditMiddleware.Middleware()) // 应用审计日志中间件
		{
			// 设备管理
			admin.GET("/devices", adminHandler.GetDevices)
			admin.DELETE("/devices/:device_id", adminHandler.DeleteDevice)

			// 虚拟网络管理
			admin.GET("/virtual-networks", adminHandler.GetVirtualNetworks)
			admin.POST("/virtual-networks", adminHandler.CreateVirtualNetwork)

			// 告警管理
			admin.GET("/alerts", adminHandler.GetAlerts)
			admin.POST("/alerts/:alert_id/acknowledge", adminHandler.AcknowledgeAlert)

			// 审计日志
			admin.GET("/audit-logs", adminHandler.GetAuditLogs)
		}
	}

	return r
}
