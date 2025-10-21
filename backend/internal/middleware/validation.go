package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ValidationConfig 验证配置
type ValidationConfig struct {
	// 是否启用验证
	Enabled bool
	// 最大请求体大小（字节）
	MaxBodySize int64
	// 是否验证Content-Type
	ValidateContentType bool
	// 允许的Content-Type列表
	AllowedContentTypes []string
}

// Validator 验证中间件
type Validator struct {
	config *ValidationConfig
}

// NewValidator 创建验证中间件
func NewValidator(config *ValidationConfig) *Validator {
	// 设置默认值
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 10 * 1024 * 1024 // 默认10MB
	}
	if len(config.AllowedContentTypes) == 0 {
		config.AllowedContentTypes = []string{
			"application/json",
			"application/x-www-form-urlencoded",
			"multipart/form-data",
		}
	}

	return &Validator{
		config: config,
	}
}

// Middleware 验证中间件处理函数
func (v *Validator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !v.config.Enabled {
			c.Next()
			return
		}

		// 1. 验证Content-Type
		if v.config.ValidateContentType && c.Request.Method != http.MethodGet {
			contentType := c.ContentType()
			if contentType != "" && !v.isAllowedContentType(contentType) {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error":   "invalid_content_type",
					"message": fmt.Sprintf("Content-Type '%s' is not supported", contentType),
				})
				c.Abort()
				return
			}
		}

		// 2. 验证请求体大小
		if c.Request.ContentLength > v.config.MaxBodySize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "request_too_large",
				"message": fmt.Sprintf("Request body exceeds maximum size of %d bytes", v.config.MaxBodySize),
			})
			c.Abort()
			return
		}

		// 3. 验证常见请求头
		if err := v.validateHeaders(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_headers",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isAllowedContentType 检查Content-Type是否允许
func (v *Validator) isAllowedContentType(contentType string) bool {
	// 移除charset等参数
	contentType = strings.Split(contentType, ";")[0]
	contentType = strings.TrimSpace(contentType)

	for _, allowed := range v.config.AllowedContentTypes {
		if strings.EqualFold(contentType, allowed) {
			return true
		}
	}
	return false
}

// validateHeaders 验证请求头
func (v *Validator) validateHeaders(c *gin.Context) error {
	// 验证User-Agent（防止空User-Agent攻击）
	userAgent := c.GetHeader("User-Agent")
	if userAgent == "" {
		return fmt.Errorf("User-Agent header is required")
	}

	// 验证X-Request-ID（如果存在）
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		if !isValidUUID(requestID) {
			return fmt.Errorf("X-Request-ID must be a valid UUID")
		}
	}

	return nil
}

// ======================== 输入清理函数 ========================

// SanitizeString 清理字符串输入（防止XSS、SQL注入）
func SanitizeString(input string) string {
	// 移除控制字符
	input = removeControlCharacters(input)

	// 移除前后空白
	input = strings.TrimSpace(input)

	// HTML转义（防止XSS）
	input = htmlEscape(input)

	return input
}

// removeControlCharacters 移除控制字符
func removeControlCharacters(input string) string {
	var result strings.Builder
	for _, r := range input {
		// 保留换行符、回车符、制表符
		if r == '\n' || r == '\r' || r == '\t' {
			result.WriteRune(r)
			continue
		}
		// 移除其他控制字符
		if r >= 32 || r == '\t' || r == '\n' || r == '\r' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// htmlEscape HTML转义
func htmlEscape(input string) string {
	replacer := strings.NewReplacer(
		"<", "&lt;",
		">", "&gt;",
		"&", "&amp;",
		"\"", "&quot;",
		"'", "&#x27;",
		"/", "&#x2F;",
	)
	return replacer.Replace(input)
}

// ValidateEmail 验证电子邮件格式
func ValidateEmail(email string) bool {
	// 简单的电子邮件验证正则
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateIPAddress 验证IP地址格式
func ValidateIPAddress(ip string) bool {
	// IPv4正则
	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if ipv4Regex.MatchString(ip) {
		// 验证每个数字段范围
		parts := strings.Split(ip, ".")
		for _, part := range parts {
			var num int
			fmt.Sscanf(part, "%d", &num)
			if num < 0 || num > 255 {
				return false
			}
		}
		return true
	}

	// IPv6正则（简化版本）
	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{0,4}:){7}[0-9a-fA-F]{0,4}$`)
	return ipv6Regex.MatchString(ip)
}

// ValidateDeviceName 验证设备名称
func ValidateDeviceName(name string) error {
	if name == "" {
		return fmt.Errorf("device name cannot be empty")
	}

	if utf8.RuneCountInString(name) > 64 {
		return fmt.Errorf("device name too long (max 64 characters)")
	}

	// 只允许字母、数字、连字符、下划线
	validNameRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("device name can only contain letters, numbers, hyphens, and underscores")
	}

	return nil
}

// ValidateUUID 验证UUID格式
func ValidateUUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}
	return nil
}

// isValidUUID 检查是否为有效UUID
func isValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// ValidateCIDR 验证CIDR格式
func ValidateCIDR(cidr string) error {
	// CIDR格式：IP/前缀长度
	cidrRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`)
	if !cidrRegex.MatchString(cidr) {
		return fmt.Errorf("invalid CIDR format")
	}

	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid CIDR format")
	}

	// 验证IP部分
	if !ValidateIPAddress(parts[0]) {
		return fmt.Errorf("invalid IP address in CIDR")
	}

	// 验证前缀长度
	var prefix int
	fmt.Sscanf(parts[1], "%d", &prefix)
	if prefix < 0 || prefix > 32 {
		return fmt.Errorf("invalid CIDR prefix length (must be 0-32)")
	}

	return nil
}

// ======================== SQL注入防护 ========================

// IsSQLInjectionAttempt 检测是否可能为SQL注入攻击
func IsSQLInjectionAttempt(input string) bool {
	input = strings.ToLower(input)

	// SQL注入常见模式
	sqlPatterns := []string{
		"union select",
		"drop table",
		"drop database",
		"delete from",
		"insert into",
		"update ",
		"' or '1'='1",
		"\" or \"1\"=\"1",
		"'; --",
		"\"; --",
		"' or 1=1",
		"\" or 1=1",
		"xp_cmdshell",
		"exec(",
		"execute(",
	}

	for _, pattern := range sqlPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}

	return false
}

// SQLInjectionGuard SQL注入防护中间件
func SQLInjectionGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查查询参数
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if IsSQLInjectionAttempt(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "invalid_input",
						"message": fmt.Sprintf("Invalid input detected in parameter '%s'", key),
					})
					c.Abort()
					return
				}
			}
		}

		// 检查路径参数
		for _, param := range c.Params {
			if IsSQLInjectionAttempt(param.Value) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_input",
					"message": fmt.Sprintf("Invalid input detected in path parameter '%s'", param.Key),
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ======================== XSS防护 ========================

// XSSGuard XSS攻击防护中间件
func XSSGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全响应头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}

// ======================== CSRF防护 ========================

// CSRFGuard CSRF攻击防护中间件
func CSRFGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 对于非幂等方法（POST, PUT, DELETE, PATCH），验证CSRF token
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.Request.Method != http.MethodOptions {
			token := c.GetHeader("X-CSRF-Token")
			if token == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "csrf_token_missing",
					"message": "CSRF token is required",
				})
				c.Abort()
				return
			}

			// TODO: 验证CSRF token有效性（需要与session集成）
			// 这里只是占位符，实际实现需要根据session验证token
		}

		c.Next()
	}
}
