package rules

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/edgelink/backend/cmd/alert-service/internal/notifier"
	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Engine 规则引擎
type Engine struct {
	rules          []Rule
	rulesMutex     sync.RWMutex
	parser         *Parser
	matcher        *Matcher
	executor       *Executor
	rateLimiters   map[string]*RateLimitTracker
	escalations    map[uuid.UUID]*EscalationState
	escalationLock sync.RWMutex
	deviceRepo     repository.DeviceRepository
	logger         *zap.Logger
}

// EngineConfig 引擎配置
type EngineConfig struct {
	RulesFile        string
	ReloadInterval   time.Duration // 规则文件热重载间隔
	EnableHotReload  bool
}

// NewEngine 创建规则引擎
func NewEngine(
	emailNotifier *notifier.EmailNotifier,
	webhookNotifier *notifier.WebhookNotifier,
	deviceRepo repository.DeviceRepository,
	logger *zap.Logger,
) *Engine {
	return &Engine{
		rules:        make([]Rule, 0),
		parser:       NewParser(),
		matcher:      NewMatcher(),
		executor:     NewExecutor(emailNotifier, webhookNotifier, logger),
		rateLimiters: make(map[string]*RateLimitTracker),
		escalations:  make(map[uuid.UUID]*EscalationState),
		deviceRepo:   deviceRepo,
		logger:       logger,
	}
}

// LoadRules 加载规则
func (e *Engine) LoadRules(filepath string) error {
	ruleSet, err := e.parser.ParseFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	e.rulesMutex.Lock()
	e.rules = ruleSet.Rules
	e.rulesMutex.Unlock()

	e.logger.Info("Rules loaded successfully",
		zap.Int("count", len(ruleSet.Rules)),
		zap.String("version", ruleSet.Version),
	)

	return nil
}

// LoadRulesFromBytes 从字节数组加载规则
func (e *Engine) LoadRulesFromBytes(data []byte) error {
	ruleSet, err := e.parser.ParseBytes(data)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	e.rulesMutex.Lock()
	e.rules = ruleSet.Rules
	e.rulesMutex.Unlock()

	e.logger.Info("Rules loaded from bytes",
		zap.Int("count", len(ruleSet.Rules)),
	)

	return nil
}

// ReloadRules 重新加载规则
func (e *Engine) ReloadRules(filepath string) error {
	e.logger.Info("Reloading rules", zap.String("file", filepath))
	return e.LoadRules(filepath)
}

// StartAutoReload 启动自动重载
func (e *Engine) StartAutoReload(ctx context.Context, config EngineConfig) {
	if !config.EnableHotReload {
		return
	}

	ticker := time.NewTicker(config.ReloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Stopping auto reload")
			return
		case <-ticker.C:
			if err := e.ReloadRules(config.RulesFile); err != nil {
				e.logger.Error("Failed to reload rules", zap.Error(err))
			}
		}
	}
}

// Process 处理告警
func (e *Engine) Process(ctx context.Context, alert *domain.Alert) error {
	// 获取设备信息（如果有）
	var device *domain.Device
	if alert.DeviceID != nil {
		dev, err := e.deviceRepo.FindByID(ctx, *alert.DeviceID)
		if err == nil {
			device = dev
		}
	}

	// 构建匹配上下文
	matchCtx := &MatchContext{
		Alert:     alert,
		Device:    device,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// 匹配规则
	e.rulesMutex.RLock()
	matchedRules := e.matcher.MatchMultiple(e.rules, matchCtx)
	e.rulesMutex.RUnlock()

	if len(matchedRules) == 0 {
		e.logger.Debug("No rules matched for alert",
			zap.String("alert_id", alert.ID.String()),
		)
		return nil
	}

	e.logger.Info("Rules matched",
		zap.String("alert_id", alert.ID.String()),
		zap.Int("count", len(matchedRules)),
	)

	// 执行匹配的规则
	for _, rule := range matchedRules {
		if err := e.executeRule(ctx, &rule, alert, device); err != nil {
			e.logger.Error("Failed to execute rule",
				zap.String("rule_id", rule.ID),
				zap.Error(err),
			)
		}
	}

	// 处理告警升级
	e.processEscalations(ctx, alert, device)

	return nil
}

// executeRule 执行单个规则
func (e *Engine) executeRule(ctx context.Context, rule *Rule, alert *domain.Alert, device *domain.Device) error {
	// 检查速率限制
	if rule.RateLimit != nil {
		if !e.checkRateLimit(rule, alert) {
			e.logger.Info("Rate limit exceeded, skipping rule",
				zap.String("rule_id", rule.ID),
				zap.String("alert_id", alert.ID.String()),
			)
			return nil
		}
	}

	// 执行所有启用的动作
	execCtx := &ExecutionContext{
		Alert:     alert,
		Device:    device,
		Rule:      rule,
		Timestamp: time.Now(),
	}

	for i := range rule.Actions {
		action := &rule.Actions[i]
		if !action.Enabled {
			continue
		}

		result := e.executor.ExecuteWithRetry(ctx, action, execCtx)
		if !result.Success {
			e.logger.Error("Action execution failed after retries",
				zap.String("rule_id", rule.ID),
				zap.String("action_type", string(action.Type)),
				zap.Error(result.Error),
			)
		}
	}

	// 设置升级状态
	if rule.Escalation != nil && rule.Escalation.Enabled {
		e.setEscalationState(alert.ID, rule.ID)
	}

	return nil
}

// checkRateLimit 检查速率限制
func (e *Engine) checkRateLimit(rule *Rule, alert *domain.Alert) bool {
	key := e.getRateLimitKey(rule, alert)

	tracker, exists := e.rateLimiters[key]
	if !exists {
		tracker = &RateLimitTracker{
			maxCount: rule.RateLimit.MaxNotifications,
			window:   rule.RateLimit.Window,
			tokens:   make([]time.Time, 0),
		}
		e.rateLimiters[key] = tracker
	}

	return tracker.Allow()
}

// getRateLimitKey 获取速率限制键
func (e *Engine) getRateLimitKey(rule *Rule, alert *domain.Alert) string {
	switch rule.RateLimit.Scope {
	case "global":
		return "global"
	case "per_device":
		if alert.DeviceID != nil {
			return fmt.Sprintf("device:%s", alert.DeviceID.String())
		}
		return "global"
	case "per_rule":
		return fmt.Sprintf("rule:%s", rule.ID)
	default:
		return fmt.Sprintf("rule:%s", rule.ID)
	}
}

// setEscalationState 设置升级状态
func (e *Engine) setEscalationState(alertID uuid.UUID, ruleID string) {
	e.escalationLock.Lock()
	defer e.escalationLock.Unlock()

	if _, exists := e.escalations[alertID]; !exists {
		e.escalations[alertID] = &EscalationState{
			AlertID:        alertID,
			RuleID:         ruleID,
			FirstTriggered: time.Now(),
			LastNotified:   time.Now(),
			RepeatCount:    0,
			Acknowledged:   false,
		}
	}
}

// processEscalations 处理告警升级
func (e *Engine) processEscalations(ctx context.Context, alert *domain.Alert, device *domain.Device) {
	e.escalationLock.RLock()
	state, exists := e.escalations[alert.ID]
	e.escalationLock.RUnlock()

	if !exists {
		return
	}

	// 如果告警已确认，不再升级
	if alert.Status == domain.AlertStatusAcknowledged || alert.Status == domain.AlertStatusResolved {
		e.escalationLock.Lock()
		delete(e.escalations, alert.ID)
		e.escalationLock.Unlock()
		return
	}

	// 找到对应的规则
	e.rulesMutex.RLock()
	var rule *Rule
	for i := range e.rules {
		if e.rules[i].ID == state.RuleID {
			rule = &e.rules[i]
			break
		}
	}
	e.rulesMutex.RUnlock()

	if rule == nil || rule.Escalation == nil || !rule.Escalation.Enabled {
		return
	}

	escalation := rule.Escalation
	now := time.Now()

	// 检查是否需要升级
	if now.Sub(state.FirstTriggered) < escalation.WaitDuration {
		return // 还未到升级时间
	}

	// 检查是否需要重复通知
	if escalation.RepeatInterval > 0 {
		if now.Sub(state.LastNotified) < escalation.RepeatInterval {
			return // 还未到重复通知时间
		}

		// 检查重复次数限制
		if escalation.MaxRepeat > 0 && state.RepeatCount >= escalation.MaxRepeat {
			return // 已达到最大重复次数
		}
	}

	// 执行升级动作
	execCtx := &ExecutionContext{
		Alert:     alert,
		Device:    device,
		Rule:      rule,
		Timestamp: now,
	}

	for i := range escalation.EscalateTo {
		action := &escalation.EscalateTo[i]
		result := e.executor.ExecuteWithRetry(ctx, action, execCtx)
		if !result.Success {
			e.logger.Error("Escalation action failed",
				zap.String("rule_id", rule.ID),
				zap.String("action_type", string(action.Type)),
				zap.Error(result.Error),
			)
		}
	}

	// 更新升级状态
	e.escalationLock.Lock()
	state.LastNotified = now
	state.RepeatCount++
	e.escalationLock.Unlock()

	e.logger.Info("Alert escalated",
		zap.String("alert_id", alert.ID.String()),
		zap.String("rule_id", rule.ID),
		zap.Int("repeat_count", state.RepeatCount),
	)
}

// TestRule 测试规则匹配
func (e *Engine) TestRule(ruleID string, alert *domain.Alert, device *domain.Device) (bool, error) {
	e.rulesMutex.RLock()
	defer e.rulesMutex.RUnlock()

	var rule *Rule
	for i := range e.rules {
		if e.rules[i].ID == ruleID {
			rule = &e.rules[i]
			break
		}
	}

	if rule == nil {
		return false, fmt.Errorf("rule not found: %s", ruleID)
	}

	matchCtx := &MatchContext{
		Alert:     alert,
		Device:    device,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	return e.matcher.Match(rule, matchCtx), nil
}

// GetRules 获取所有规则
func (e *Engine) GetRules() []Rule {
	e.rulesMutex.RLock()
	defer e.rulesMutex.RUnlock()

	rules := make([]Rule, len(e.rules))
	copy(rules, e.rules)
	return rules
}

// GetRule 获取单个规则
func (e *Engine) GetRule(ruleID string) (*Rule, error) {
	e.rulesMutex.RLock()
	defer e.rulesMutex.RUnlock()

	for i := range e.rules {
		if e.rules[i].ID == ruleID {
			rule := e.rules[i]
			return &rule, nil
		}
	}

	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

// RateLimitTracker 速率限制跟踪器
type RateLimitTracker struct {
	maxCount int
	window   time.Duration
	tokens   []time.Time
	mutex    sync.Mutex
}

// Allow 检查是否允许
func (t *RateLimitTracker) Allow() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-t.window)

	// 清理过期的token
	validTokens := make([]time.Time, 0)
	for _, token := range t.tokens {
		if token.After(cutoff) {
			validTokens = append(validTokens, token)
		}
	}
	t.tokens = validTokens

	// 检查是否超出限制
	if len(t.tokens) >= t.maxCount {
		return false
	}

	// 添加新token
	t.tokens = append(t.tokens, now)
	return true
}
