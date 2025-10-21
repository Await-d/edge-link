package middleware

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig CORS配置
type CORSConfig struct {
	// 是否启用CORS
	Enabled bool
	// 允许的源列表（* 表示所有源）
	AllowedOrigins []string
	// 允许的HTTP方法
	AllowedMethods []string
	// 允许的请求头
	AllowedHeaders []string
	// 暴露给客户端的响应头
	ExposedHeaders []string
	// 是否允许携带凭证（cookies）
	AllowCredentials bool
	// 预检请求缓存时间（秒）
	MaxAge int
}

// NewCORSConfig 创建默认CORS配置
func NewCORSConfig() *CORSConfig {
	return &CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{
			"https://app.edgelink.com",
			"https://admin.edgelink.com",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept",
			"Accept-Encoding",
			"Authorization",
			"X-Request-ID",
			"X-CSRF-Token",
			"X-Device-ID",
			"X-Organization-ID",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:           12 * 3600, // 12小时
	}
}

// CORSMiddleware CORS中间件
func CORSMiddleware(config *CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")

		// 如果没有Origin头，跳过CORS处理（同源请求）
		if origin == "" {
			c.Next()
			return
		}

		// 检查origin是否在允许列表中
		if !isAllowedOrigin(origin, config.AllowedOrigins) {
			// Origin不在允许列表中，拒绝请求
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden_origin",
				"message": "Origin not allowed",
			})
			return
		}

		// 设置CORS响应头
		c.Header("Access-Control-Allow-Origin", origin)

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if len(config.ExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
		}

		// 处理预检请求（OPTIONS）
		if c.Request.Method == http.MethodOptions {
			if len(config.AllowedMethods) > 0 {
				c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			}

			if len(config.AllowedHeaders) > 0 {
				c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}

			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", string(rune(config.MaxAge)))
			}

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isAllowedOrigin 检查origin是否在允许列表中
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		// 支持通配符
		if allowed == "*" {
			return true
		}

		// 精确匹配
		if allowed == origin {
			return true
		}

		// 支持子域名通配符（例如：*.edgelink.com）
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}

// ProductionCORSConfig 生产环境CORS配置
// 严格限制允许的源
func ProductionCORSConfig(allowedDomains []string) *CORSConfig {
	return &CORSConfig{
		Enabled:        true,
		AllowedOrigins: allowedDomains,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Request-ID",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"X-Request-ID",
		},
		AllowCredentials: true,
		MaxAge:           3600, // 1小时
	}
}

// DevelopmentCORSConfig 开发环境CORS配置
// 允许所有源（仅用于开发）
func DevelopmentCORSConfig() *CORSConfig {
	return &CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{
			"*", // 允许所有源（仅开发环境！）
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"*", // 允许所有请求头（仅开发环境！）
		},
		ExposedHeaders: []string{
			"*", // 暴露所有响应头（仅开发环境！）
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24小时
	}
}

// SecurityHeaders 安全响应头中间件
// 添加常见的安全响应头
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止MIME类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// XSS保护
		c.Header("X-XSS-Protection", "1; mode=block")

		// 强制HTTPS（仅在生产环境启用）
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 推荐内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'")

		// 推荐权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// 推荐引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// RequestID 请求ID中间件
// 为每个请求生成唯一ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取ID
		requestID := c.GetHeader("X-Request-ID")

		// 如果没有，生成新ID
		if requestID == "" {
			requestID = generateRequestID()
		}

		// 设置到上下文和响应头
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	// 简单实现：时间戳 + 随机数
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
}

// TrustedProxies 配置可信代理
func TrustedProxies(trustedProxies []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置可信代理
		if err := c.Engine().SetTrustedProxies(trustedProxies); err != nil {
			// 记录错误但继续处理请求
			c.Set("trusted_proxy_error", err.Error())
		}
		c.Next()
	}
}
