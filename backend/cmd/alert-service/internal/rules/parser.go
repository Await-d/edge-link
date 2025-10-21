package rules

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Parser 规则解析器
type Parser struct {
	validators []Validator
}

// Validator 规则验证器接口
type Validator interface {
	Validate(rule *Rule) error
}

// NewParser 创建规则解析器
func NewParser() *Parser {
	return &Parser{
		validators: []Validator{
			&BasicValidator{},
			&ConditionsValidator{},
			&ActionsValidator{},
			&TimeRangeValidator{},
		},
	}
}

// ParseFile 从文件解析规则集
func (p *Parser) ParseFile(filepath string) (*RuleSet, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %w", err)
	}

	return p.ParseBytes(data)
}

// ParseBytes 从字节数组解析规则集
func (p *Parser) ParseBytes(data []byte) (*RuleSet, error) {
	var ruleSet RuleSet
	if err := yaml.Unmarshal(data, &ruleSet); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// 验证规则集
	if err := p.ValidateRuleSet(&ruleSet); err != nil {
		return nil, fmt.Errorf("rule validation failed: %w", err)
	}

	// 初始化默认值
	p.initializeDefaults(&ruleSet)

	return &ruleSet, nil
}

// ValidateRuleSet 验证规则集
func (p *Parser) ValidateRuleSet(ruleSet *RuleSet) error {
	if ruleSet.Version == "" {
		return fmt.Errorf("rule set version is required")
	}

	if len(ruleSet.Rules) == 0 {
		return fmt.Errorf("rule set must contain at least one rule")
	}

	// 检查规则ID唯一性
	ruleIDs := make(map[string]bool)
	for i, rule := range ruleSet.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rule at index %d has empty ID", i)
		}

		if ruleIDs[rule.ID] {
			return fmt.Errorf("duplicate rule ID: %s", rule.ID)
		}
		ruleIDs[rule.ID] = true

		// 验证每个规则
		for _, validator := range p.validators {
			if err := validator.Validate(&rule); err != nil {
				return fmt.Errorf("rule '%s' validation failed: %w", rule.ID, err)
			}
		}
	}

	return nil
}

// initializeDefaults 初始化默认值
func (p *Parser) initializeDefaults(ruleSet *RuleSet) {
	now := time.Now()

	for i := range ruleSet.Rules {
		rule := &ruleSet.Rules[i]

		// 默认启用
		if !rule.Enabled {
			rule.Enabled = true
		}

		// 默认优先级
		if rule.Priority == 0 {
			rule.Priority = 100
		}

		// 初始化时间戳
		if rule.CreatedAt.IsZero() {
			rule.CreatedAt = now
		}
		if rule.UpdatedAt.IsZero() {
			rule.UpdatedAt = now
		}

		// 初始化动作默认值
		for j := range rule.Actions {
			action := &rule.Actions[j]
			if !action.Enabled {
				action.Enabled = true
			}

			// 默认重试策略
			if action.RetryPolicy == nil {
				action.RetryPolicy = &RetryPolicy{
					MaxRetries:  3,
					RetryDelay:  5 * time.Second,
					BackoffRate: 2.0,
				}
			}
		}

		// 初始化速率限制默认值
		if rule.RateLimit != nil && rule.RateLimit.Scope == "" {
			rule.RateLimit.Scope = "per_rule"
		}
	}
}

// BasicValidator 基本验证器
type BasicValidator struct{}

func (v *BasicValidator) Validate(rule *Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if rule.Priority < 0 {
		return fmt.Errorf("rule priority must be non-negative")
	}

	if len(rule.Actions) == 0 {
		return fmt.Errorf("rule must have at least one action")
	}

	return nil
}

// ConditionsValidator 条件验证器
type ConditionsValidator struct{}

func (v *ConditionsValidator) Validate(rule *Rule) error {
	return v.validateConditions(&rule.Conditions)
}

func (v *ConditionsValidator) validateConditions(cond *Conditions) error {
	// 递归验证嵌套条件
	for i := range cond.AllOf {
		if err := v.validateConditions(&cond.AllOf[i]); err != nil {
			return fmt.Errorf("all_of[%d]: %w", i, err)
		}
	}

	for i := range cond.AnyOf {
		if err := v.validateConditions(&cond.AnyOf[i]); err != nil {
			return fmt.Errorf("any_of[%d]: %w", i, err)
		}
	}

	for i := range cond.NoneOf {
		if err := v.validateConditions(&cond.NoneOf[i]); err != nil {
			return fmt.Errorf("none_of[%d]: %w", i, err)
		}
	}

	// 验证时间范围
	if cond.TimeRange != nil {
		if err := validateTimeRange(cond.TimeRange); err != nil {
			return fmt.Errorf("time_range: %w", err)
		}
	}

	return nil
}

// ActionsValidator 动作验证器
type ActionsValidator struct{}

func (v *ActionsValidator) Validate(rule *Rule) error {
	for i, action := range rule.Actions {
		if action.Type == "" {
			return fmt.Errorf("action[%d]: type is required", i)
		}

		// 验证必需的配置项
		if err := v.validateActionConfig(&action); err != nil {
			return fmt.Errorf("action[%d]: %w", i, err)
		}

		// 验证重试策略
		if action.RetryPolicy != nil {
			if action.RetryPolicy.MaxRetries < 0 {
				return fmt.Errorf("action[%d]: max_retries must be non-negative", i)
			}
			if action.RetryPolicy.BackoffRate < 1.0 {
				return fmt.Errorf("action[%d]: backoff_rate must be >= 1.0", i)
			}
		}
	}

	return nil
}

func (v *ActionsValidator) validateActionConfig(action *Action) error {
	switch action.Type {
	case ActionTypeEmail:
		if _, ok := action.Config["recipients"]; !ok {
			return fmt.Errorf("email action requires 'recipients' config")
		}

	case ActionTypeWebhook:
		if _, ok := action.Config["url"]; !ok {
			return fmt.Errorf("webhook action requires 'url' config")
		}

	case ActionTypeSlack:
		if _, ok := action.Config["webhook_url"]; !ok {
			return fmt.Errorf("slack action requires 'webhook_url' config")
		}

	case ActionTypePagerDuty:
		if _, ok := action.Config["service_key"]; !ok {
			return fmt.Errorf("pagerduty action requires 'service_key' config")
		}

	case ActionTypeDingTalk:
		if _, ok := action.Config["webhook_url"]; !ok {
			return fmt.Errorf("dingtalk action requires 'webhook_url' config")
		}

	case ActionTypeWeChat:
		if _, ok := action.Config["webhook_url"]; !ok {
			return fmt.Errorf("wechat action requires 'webhook_url' config")
		}
	}

	return nil
}

// TimeRangeValidator 时间范围验证器
type TimeRangeValidator struct{}

func (v *TimeRangeValidator) Validate(rule *Rule) error {
	if rule.Conditions.TimeRange != nil {
		if err := validateTimeRange(rule.Conditions.TimeRange); err != nil {
			return err
		}
	}

	if rule.Silence != nil && rule.Silence.Enabled {
		for i, tr := range rule.Silence.TimeRanges {
			if err := validateTimeRange(&tr); err != nil {
				return fmt.Errorf("silence time_range[%d]: %w", i, err)
			}
		}
	}

	return nil
}

// validateTimeRange 验证时间范围格式
func validateTimeRange(tr *TimeRange) error {
	if tr.StartTime == "" || tr.EndTime == "" {
		return fmt.Errorf("start and end time are required")
	}

	// 验证时间格式 HH:MM
	if _, err := time.Parse("15:04", tr.StartTime); err != nil {
		return fmt.Errorf("invalid start time format (expected HH:MM): %w", err)
	}

	if _, err := time.Parse("15:04", tr.EndTime); err != nil {
		return fmt.Errorf("invalid end time format (expected HH:MM): %w", err)
	}

	// 验证时区
	if tr.Timezone != "" {
		if _, err := time.LoadLocation(tr.Timezone); err != nil {
			return fmt.Errorf("invalid timezone: %w", err)
		}
	}

	// 验证星期
	validWeekdays := map[string]bool{
		"Monday": true, "Tuesday": true, "Wednesday": true, "Thursday": true,
		"Friday": true, "Saturday": true, "Sunday": true,
	}

	for _, wd := range tr.Weekdays {
		if !validWeekdays[wd] {
			return fmt.Errorf("invalid weekday: %s", wd)
		}
	}

	return nil
}
