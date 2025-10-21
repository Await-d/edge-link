package rules

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 规则引擎HTTP处理器
type Handler struct {
	engine *Engine
	logger *zap.Logger
}

// NewHandler 创建规则处理器
func NewHandler(engine *Engine, logger *zap.Logger) *Handler {
	return &Handler{
		engine: engine,
		logger: logger,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	rules := router.Group("/rules")
	{
		rules.GET("", h.ListRules)
		rules.GET("/:rule_id", h.GetRule)
		rules.POST("/reload", h.ReloadRules)
		rules.POST("/test", h.TestRule)
	}
}

// ListRulesResponse 规则列表响应
type ListRulesResponse struct {
	Rules []Rule `json:"rules"`
	Total int    `json:"total"`
}

// ListRules 列出所有规则
// @Summary 获取所有通知规则
// @Tags rules
// @Accept json
// @Produce json
// @Success 200 {object} ListRulesResponse
// @Router /api/v1/rules [get]
func (h *Handler) ListRules(c *gin.Context) {
	rules := h.engine.GetRules()

	c.JSON(http.StatusOK, ListRulesResponse{
		Rules: rules,
		Total: len(rules),
	})
}

// GetRuleResponse 规则详情响应
type GetRuleResponse struct {
	Rule *Rule `json:"rule"`
}

// GetRule 获取单个规则详情
// @Summary 获取规则详情
// @Tags rules
// @Accept json
// @Produce json
// @Param rule_id path string true "规则ID"
// @Success 200 {object} GetRuleResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/rules/{rule_id} [get]
func (h *Handler) GetRule(c *gin.Context) {
	ruleID := c.Param("rule_id")

	rule, err := h.engine.GetRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Rule not found",
		})
		return
	}

	c.JSON(http.StatusOK, GetRuleResponse{
		Rule: rule,
	})
}

// ReloadRulesRequest 重新加载规则请求
type ReloadRulesRequest struct {
	FilePath string `json:"file_path,omitempty"`
}

// ReloadRulesResponse 重新加载规则响应
type ReloadRulesResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// ReloadRules 重新加载规则
// @Summary 重新加载通知规则
// @Tags rules
// @Accept json
// @Produce json
// @Param request body ReloadRulesRequest false "重载请求"
// @Success 200 {object} ReloadRulesResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/rules/reload [post]
func (h *Handler) ReloadRules(c *gin.Context) {
	var req ReloadRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有请求体，使用默认文件路径
		req.FilePath = ""
	}

	// TODO: 从配置获取规则文件路径
	filePath := req.FilePath
	if filePath == "" {
		filePath = "alert-rules.yaml"
	}

	if err := h.engine.ReloadRules(filePath); err != nil {
		h.logger.Error("Failed to reload rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reload rules",
		})
		return
	}

	rules := h.engine.GetRules()
	c.JSON(http.StatusOK, ReloadRulesResponse{
		Success: true,
		Message: "Rules reloaded successfully",
		Count:   len(rules),
	})
}

// TestRuleRequest 测试规则请求
type TestRuleRequest struct {
	RuleID   string                 `json:"rule_id" binding:"required"`
	Alert    map[string]interface{} `json:"alert" binding:"required"`
	Device   map[string]interface{} `json:"device,omitempty"`
}

// TestRuleResponse 测试规则响应
type TestRuleResponse struct {
	Matched bool   `json:"matched"`
	RuleID  string `json:"rule_id"`
	Message string `json:"message"`
}

// TestRule 测试规则匹配
// @Summary 测试规则匹配
// @Tags rules
// @Accept json
// @Produce json
// @Param request body TestRuleRequest true "测试请求"
// @Success 200 {object} TestRuleResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/rules/test [post]
func (h *Handler) TestRule(c *gin.Context) {
	var req TestRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// TODO: 构建测试用的Alert和Device对象
	// 这里需要实现从map到domain对象的转换
	// 为简化示例，暂时返回未实现的响应

	c.JSON(http.StatusOK, TestRuleResponse{
		Matched: false,
		RuleID:  req.RuleID,
		Message: "Rule testing feature not fully implemented",
	})
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}
