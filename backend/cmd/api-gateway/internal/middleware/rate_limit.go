package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/edgelink/backend/internal/cache"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	// 是否启用速率限制
	Enabled bool
	// 全局速率限制（每分钟请求数）
	GlobalLimit int64
	// 按组织速率限制（每组织每分钟请求数）
	PerOrganizationLimit int64
	// 按IP速率限制（每IP每分钟请求数）
	PerIPLimit int64
	// 按用户速率限制（每用户每分钟请求数）
	PerUserLimit int64
	// 白名单IP列表（不受速率限制）
	WhitelistIPs []string
}

// RateLimiter 速率限制中间件
type RateLimiter struct {
	config      *RateLimitConfig
	redisClient *cache.RedisClient
}

// NewRateLimiter 创建速率限制中间件
func NewRateLimiter(config *RateLimitConfig, redisClient *cache.RedisClient) *RateLimiter {
	// 设置默认值
	if config.GlobalLimit == 0 {
		config.GlobalLimit = 10000 // 全局默认每分钟10000请求
	}
	if config.PerOrganizationLimit == 0 {
		config.PerOrganizationLimit = 1000 // 每组织默认每分钟1000请求
	}
	if config.PerIPLimit == 0 {
		config.PerIPLimit = 100 // 每IP默认每分钟100请求
	}
	if config.PerUserLimit == 0 {
		config.PerUserLimit = 500 // 每用户默认每分钟500请求
	}

	return &RateLimiter{
		config:      config,
		redisClient: redisClient,
	}
}

// Middleware 速率限制中间件处理函数
func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果未启用速率限制，直接通过
		if !r.config.Enabled {
			c.Next()
			return
		}

		// 检查是否在白名单中
		clientIP := c.ClientIP()
		if r.isWhitelisted(clientIP) {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// 1. 全局速率限制
		if r.config.GlobalLimit > 0 {
			key := "global"
			if _, err := r.redisClient.IncrementRateLimit(ctx, key, r.config.GlobalLimit); err != nil {
				r.handleRateLimitExceeded(c, "global", r.config.GlobalLimit)
				return
			}
		}

		// 2. 按IP速率限制
		if r.config.PerIPLimit > 0 {
			key := fmt.Sprintf("ip:%s", clientIP)
			if _, err := r.redisClient.IncrementRateLimit(ctx, key, r.config.PerIPLimit); err != nil {
				r.handleRateLimitExceeded(c, "ip", r.config.PerIPLimit)
				return
			}
		}

		// 3. 按组织速率限制（从请求头或上下文中获取组织ID）
		if r.config.PerOrganizationLimit > 0 {
			if orgID, exists := c.Get("organization_id"); exists {
				key := fmt.Sprintf("org:%s", orgID)
				if _, err := r.redisClient.IncrementRateLimit(ctx, key, r.config.PerOrganizationLimit); err != nil {
					r.handleRateLimitExceeded(c, "organization", r.config.PerOrganizationLimit)
					return
				}
			}
		}

		// 4. 按用户速率限制（从请求头或上下文中获取用户ID）
		if r.config.PerUserLimit > 0 {
			if userID, exists := c.Get("user_id"); exists {
				key := fmt.Sprintf("user:%s", userID)
				if _, err := r.redisClient.IncrementRateLimit(ctx, key, r.config.PerUserLimit); err != nil {
					r.handleRateLimitExceeded(c, "user", r.config.PerUserLimit)
					return
				}
			} else if deviceID, exists := c.Get("device_id"); exists {
				// 对于设备API，使用设备ID作为限制标识
				key := fmt.Sprintf("device:%s", deviceID)
				if _, err := r.redisClient.IncrementRateLimit(ctx, key, r.config.PerUserLimit); err != nil {
					r.handleRateLimitExceeded(c, "device", r.config.PerUserLimit)
					return
				}
			}
		}

		c.Next()
	}
}

// isWhitelisted 检查IP是否在白名单中
func (r *RateLimiter) isWhitelisted(ip string) bool {
	for _, whitelistedIP := range r.config.WhitelistIPs {
		// 支持CIDR格式（简单实现，生产环境应使用net.ParseCIDR）
		if strings.HasPrefix(ip, whitelistedIP) {
			return true
		}
	}
	return false
}

// handleRateLimitExceeded 处理速率限制超出
func (r *RateLimiter) handleRateLimitExceeded(c *gin.Context, limitType string, limit int64) {
	// 设置响应头
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	c.Header("X-RateLimit-Remaining", "0")
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(cache.TTLRateLimit).Unix()))
	c.Header("Retry-After", fmt.Sprintf("%d", int(cache.TTLRateLimit.Seconds())))

	// 记录日志
	c.Set("rate_limit_exceeded", true)
	c.Set("rate_limit_type", limitType)

	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":   "rate_limit_exceeded",
		"message": fmt.Sprintf("Rate limit exceeded for %s. Limit: %d requests per minute.", limitType, limit),
		"retry_after": int(cache.TTLRateLimit.Seconds()),
	})

	c.Abort()
}

// PerEndpointRateLimiter 针对特定端点的速率限制
func PerEndpointRateLimiter(redisClient *cache.RedisClient, endpoint string, limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// 根据IP+端点组合作为限制键
		clientIP := c.ClientIP()
		key := fmt.Sprintf("endpoint:%s:ip:%s", endpoint, clientIP)

		if _, err := redisClient.IncrementRateLimit(ctx, key, limit); err != nil {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", fmt.Sprintf("%d", int(cache.TTLRateLimit.Seconds())))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": fmt.Sprintf("Rate limit exceeded for endpoint %s. Limit: %d requests per minute.", endpoint, limit),
				"retry_after": int(cache.TTLRateLimit.Seconds()),
			})

			c.Abort()
			return
		}

		c.Next()
	}
}

// SlidingWindowRateLimiter 滑动窗口速率限制（更精确）
// 相比固定窗口，滑动窗口可以防止突发流量
type SlidingWindowRateLimiter struct {
	redisClient *cache.RedisClient
	limit       int64
	window      time.Duration
}

// NewSlidingWindowRateLimiter 创建滑动窗口速率限制器
func NewSlidingWindowRateLimiter(redisClient *cache.RedisClient, limit int64, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		redisClient: redisClient,
		limit:       limit,
		window:      window,
	}
}

// Middleware 滑动窗口速率限制中间件
func (sw *SlidingWindowRateLimiter) Middleware(identifierFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := identifierFunc(c)
		if identifier == "" {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		key := fmt.Sprintf("sliding:%s", identifier)
		now := time.Now().UnixNano()

		// 使用Redis ZSET实现滑动窗口
		client := sw.redisClient.Client()

		// 移除窗口外的记录
		windowStart := now - sw.window.Nanoseconds()
		client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

		// 获取当前窗口内的请求数
		count, err := client.ZCard(ctx, key).Result()
		if err == nil && count >= sw.limit {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", sw.limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", fmt.Sprintf("%d", int(sw.window.Seconds())))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": fmt.Sprintf("Rate limit exceeded. Limit: %d requests per %v.", sw.limit, sw.window),
				"retry_after": int(sw.window.Seconds()),
			})

			c.Abort()
			return
		}

		// 记录当前请求
		client.ZAdd(ctx, key, redis.Z{
			Score:  float64(now),
			Member: fmt.Sprintf("%d", now),
		})

		// 设置过期时间
		client.Expire(ctx, key, sw.window+time.Minute)

		c.Next()
	}
}
