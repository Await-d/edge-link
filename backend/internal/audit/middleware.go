package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuditMiddleware 审计日志中间件
type AuditMiddleware struct {
	auditLogRepo repository.AuditLogRepository
	logger       *zap.Logger
}

// NewAuditMiddleware 创建审计日志中间件
func NewAuditMiddleware(auditLogRepo repository.AuditLogRepository, logger *zap.Logger) *AuditMiddleware {
	return &AuditMiddleware{
		auditLogRepo: auditLogRepo,
		logger:       logger,
	}
}

// Middleware Gin中间件函数
func (am *AuditMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只审计需要记录的操作
		if !am.shouldAudit(c) {
			c.Next()
			return
		}

		// 记录操作前状态
		beforeState := am.captureBeforeState(c)

		// 捕获响应内容
		responseWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = responseWriter

		// 记录开始时间
		startTime := time.Now()

		// 执行请求
		c.Next()

		// 记录操作后状态
		afterState := am.captureAfterState(c, responseWriter)

		// 创建审计日志
		auditLog := am.createAuditLog(c, beforeState, afterState, startTime)
		if auditLog != nil {
			if err := am.auditLogRepo.Create(c.Request.Context(), auditLog); err != nil {
				am.logger.Error("Failed to create audit log",
					zap.Error(err),
					zap.String("action", auditLog.Action),
				)
			}
		}
	}
}

// shouldAudit 判断是否需要审计此请求
func (am *AuditMiddleware) shouldAudit(c *gin.Context) bool {
	// 只审计管理API (POST, PUT, PATCH, DELETE操作)
	method := c.Request.Method
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return false
	}

	// 只审计 /api/v1/admin 路径
	path := c.Request.URL.Path
	if len(path) < 14 || path[:14] != "/api/v1/admin/" {
		return false
	}

	return true
}

// captureBeforeState 捕获操作前状态
func (am *AuditMiddleware) captureBeforeState(c *gin.Context) map[string]interface{} {
	state := make(map[string]interface{})

	// 读取请求body
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			// 恢复body供后续处理使用
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var requestData map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &requestData); err == nil {
				state["request_body"] = requestData
			}
		}
	}

	// 记录路径参数
	params := make(map[string]string)
	for _, param := range c.Params {
		params[param.Key] = param.Value
	}
	if len(params) > 0 {
		state["path_params"] = params
	}

	// 记录查询参数
	query := c.Request.URL.Query()
	if len(query) > 0 {
		state["query_params"] = query
	}

	return state
}

// captureAfterState 捕获操作后状态
func (am *AuditMiddleware) captureAfterState(c *gin.Context, rw *responseBodyWriter) map[string]interface{} {
	state := make(map[string]interface{})

	// 记录HTTP状态码
	state["http_status"] = rw.Status()

	// 记录响应body
	if rw.body.Len() > 0 {
		var responseData interface{}
		if err := json.Unmarshal(rw.body.Bytes(), &responseData); err == nil {
			state["response_body"] = responseData
		}
	}

	// 记录错误信息(如果有)
	if len(c.Errors) > 0 {
		errors := make([]string, len(c.Errors))
		for i, err := range c.Errors {
			errors[i] = err.Error()
		}
		state["errors"] = errors
	}

	return state
}

// createAuditLog 创建审计日志记录
func (am *AuditMiddleware) createAuditLog(
	c *gin.Context,
	beforeState map[string]interface{},
	afterState map[string]interface{},
	startTime time.Time,
) *domain.AuditLog {
	// 提取资源信息
	resourceType, resourceID := am.extractResourceInfo(c)
	if resourceType == "" {
		return nil
	}

	// 提取操作者信息 (TODO: 从认证上下文中获取)
	// 目前使用请求中的信息或默认值
	actorID := am.extractActorID(c)
	organizationID := am.extractOrganizationID(c)

	// 确定操作类型
	action := am.determineAction(c)

	// 序列化状态
	beforeStateJSON, _ := json.Marshal(beforeState)
	afterStateJSON, _ := json.Marshal(afterState)

	// 提取IP地址
	ipAddress := c.ClientIP()

	// 提取User-Agent
	userAgent := c.Request.UserAgent()

	// Convert JSON bytes to JSONB (map[string]interface{})
	var beforeStateJSONB, afterStateJSONB *domain.JSONB
	if len(beforeStateJSON) > 0 {
		var beforeMap domain.JSONB
		if err := json.Unmarshal(beforeStateJSON, &beforeMap); err == nil {
			beforeStateJSONB = &beforeMap
		}
	}
	if len(afterStateJSON) > 0 {
		var afterMap domain.JSONB
		if err := json.Unmarshal(afterStateJSON, &afterMap); err == nil {
			afterStateJSONB = &afterMap
		}
	}

	return &domain.AuditLog{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ActorID:        &actorID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		BeforeState:    beforeStateJSONB,
		AfterState:     afterStateJSONB,
		IPAddress:      &ipAddress,
		UserAgent:      &userAgent,
		CreatedAt:      startTime,
	}
}

// extractResourceInfo 从请求中提取资源类型和ID
func (am *AuditMiddleware) extractResourceInfo(c *gin.Context) (domain.ResourceType, uuid.UUID) {
	path := c.Request.URL.Path

	// 匹配不同的资源路径模式
	// /api/v1/admin/devices/:device_id
	if deviceID := c.Param("device_id"); deviceID != "" {
		if id, err := uuid.Parse(deviceID); err == nil {
			return domain.ResourceTypeDevice, id
		}
	}

	// /api/v1/admin/virtual-networks/:network_id
	if networkID := c.Param("network_id"); networkID != "" {
		if id, err := uuid.Parse(networkID); err == nil {
			return domain.ResourceTypeVirtualNetwork, id
		}
	}

	// /api/v1/admin/alerts/:alert_id
	if alertID := c.Param("alert_id"); alertID != "" {
		if id, err := uuid.Parse(alertID); err == nil {
			return domain.ResourceTypeAlert, id
		}
	}

	// 对于创建操作,可能没有ID参数,从path推断类型
	if c.Request.Method == http.MethodPost {
		if len(path) >= 26 && path[:26] == "/api/v1/admin/virtual-networks" {
			return domain.ResourceTypeVirtualNetwork, uuid.Nil
		}
		if len(path) >= 20 && path[:20] == "/api/v1/admin/devices" {
			return domain.ResourceTypeDevice, uuid.Nil
		}
		if len(path) >= 19 && path[:19] == "/api/v1/admin/alerts" {
			return domain.ResourceTypeAlert, uuid.Nil
		}
	}

	return "", uuid.Nil
}

// extractActorID 提取操作者ID
func (am *AuditMiddleware) extractActorID(c *gin.Context) uuid.UUID {
	// TODO: 从认证上下文中获取当前用户ID
	// 目前返回一个默认值或从请求中提取
	if actorIDStr := c.GetHeader("X-Actor-ID"); actorIDStr != "" {
		if actorID, err := uuid.Parse(actorIDStr); err == nil {
			return actorID
		}
	}

	// 临时使用系统默认ID
	return uuid.MustParse("00000000-0000-0000-0000-000000000000")
}

// extractOrganizationID 提取组织ID
func (am *AuditMiddleware) extractOrganizationID(c *gin.Context) uuid.UUID {
	// TODO: 从认证上下文中获取组织ID
	// 目前从查询参数或请求体中提取
	if orgIDStr := c.Query("organization_id"); orgIDStr != "" {
		if orgID, err := uuid.Parse(orgIDStr); err == nil {
			return orgID
		}
	}

	// 尝试从请求体中提取
	if c.Request.Body != nil {
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var body map[string]interface{}
		if json.Unmarshal(bodyBytes, &body) == nil {
			if orgIDStr, ok := body["organization_id"].(string); ok {
				if orgID, err := uuid.Parse(orgIDStr); err == nil {
					return orgID
				}
			}
		}
	}

	// 临时使用默认组织ID
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

// determineAction 确定操作类型
func (am *AuditMiddleware) determineAction(c *gin.Context) string {
	method := c.Request.Method
	path := c.Request.URL.Path

	// 基于HTTP方法和路径的组合判断操作
	switch method {
	case http.MethodPost:
		if len(path) >= 14 {
			if path[len(path)-12:] == "/acknowledge" {
				return "acknowledge"
			}
		}
		return "create"
	case http.MethodPut:
		return "update"
	case http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// responseBodyWriter 包装ResponseWriter以捕获响应body
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseBodyWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
