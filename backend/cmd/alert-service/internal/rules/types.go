package rules

import (
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
)

// Rule 通知规则定义
type Rule struct {
	ID          string       `yaml:"id" json:"id"`
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description,omitempty" json:"description,omitempty"`
	Enabled     bool         `yaml:"enabled" json:"enabled"`
	Priority    int          `yaml:"priority" json:"priority"` // 数字越小优先级越高
	Conditions  Conditions   `yaml:"conditions" json:"conditions"`
	Actions     []Action     `yaml:"actions" json:"actions"`
	RateLimit   *RateLimit   `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	Escalation  *Escalation  `yaml:"escalation,omitempty" json:"escalation,omitempty"`
	Silence     *SilenceRule `yaml:"silence,omitempty" json:"silence,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt   time.Time    `yaml:"created_at" json:"created_at"`
	UpdatedAt   time.Time    `yaml:"updated_at" json:"updated_at"`
}

// Conditions 匹配条件
type Conditions struct {
	Severity     []domain.Severity   `yaml:"severity,omitempty" json:"severity,omitempty"`
	AlertTypes   []domain.AlertType  `yaml:"alert_types,omitempty" json:"alert_types,omitempty"`
	DeviceIDs    []string            `yaml:"device_ids,omitempty" json:"device_ids,omitempty"`
	DeviceTags   []string            `yaml:"device_tags,omitempty" json:"device_tags,omitempty"` // 设备标签匹配
	TimeRange    *TimeRange          `yaml:"time_range,omitempty" json:"time_range,omitempty"`
	MessageMatch string              `yaml:"message_match,omitempty" json:"message_match,omitempty"` // 正则表达式匹配消息
	Metadata     map[string]string   `yaml:"metadata,omitempty" json:"metadata,omitempty"` // 元数据键值对匹配
	AllOf        []Conditions        `yaml:"all_of,omitempty" json:"all_of,omitempty"` // AND逻辑
	AnyOf        []Conditions        `yaml:"any_of,omitempty" json:"any_of,omitempty"` // OR逻辑
	NoneOf       []Conditions        `yaml:"none_of,omitempty" json:"none_of,omitempty"` // NOT逻辑
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime string   `yaml:"start" json:"start"` // HH:MM格式
	EndTime   string   `yaml:"end" json:"end"`     // HH:MM格式
	Timezone  string   `yaml:"timezone" json:"timezone"`
	Weekdays  []string `yaml:"weekdays,omitempty" json:"weekdays,omitempty"` // Monday, Tuesday, etc.
}

// Action 通知动作
type Action struct {
	Type     ActionType             `yaml:"type" json:"type"`
	Enabled  bool                   `yaml:"enabled" json:"enabled"`
	Config   map[string]interface{} `yaml:"config" json:"config"`
	RetryPolicy *RetryPolicy        `yaml:"retry_policy,omitempty" json:"retry_policy,omitempty"`
}

// ActionType 动作类型
type ActionType string

const (
	ActionTypeEmail      ActionType = "email"
	ActionTypeWebhook    ActionType = "webhook"
	ActionTypeSlack      ActionType = "slack"
	ActionTypePagerDuty  ActionType = "pagerduty"
	ActionTypeDingTalk   ActionType = "dingtalk"
	ActionTypeWeChat     ActionType = "wechat"
	ActionTypeTelegram   ActionType = "telegram"
	ActionTypeCustom     ActionType = "custom"
)

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries  int           `yaml:"max_retries" json:"max_retries"`
	RetryDelay  time.Duration `yaml:"retry_delay" json:"retry_delay"`
	BackoffRate float64       `yaml:"backoff_rate" json:"backoff_rate"` // 指数退避倍率
}

// RateLimit 速率限制
type RateLimit struct {
	MaxNotifications int           `yaml:"max_notifications" json:"max_notifications"`
	Window           time.Duration `yaml:"window" json:"window"`
	Scope            string        `yaml:"scope" json:"scope"` // "global", "per_device", "per_rule"
}

// Escalation 告警升级
type Escalation struct {
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	WaitDuration    time.Duration `yaml:"wait_duration" json:"wait_duration"` // 未确认等待时长
	EscalateTo      []Action      `yaml:"escalate_to" json:"escalate_to"` // 升级后的通知动作
	RepeatInterval  time.Duration `yaml:"repeat_interval,omitempty" json:"repeat_interval,omitempty"` // 重复通知间隔
	MaxRepeat       int           `yaml:"max_repeat,omitempty" json:"max_repeat,omitempty"` // 最大重复次数
}

// SilenceRule 静默规则
type SilenceRule struct {
	Enabled    bool       `yaml:"enabled" json:"enabled"`
	TimeRanges []TimeRange `yaml:"time_ranges" json:"time_ranges"` // 支持多个时间段
	Comment    string     `yaml:"comment,omitempty" json:"comment,omitempty"`
}

// RuleSet 规则集合
type RuleSet struct {
	Version string  `yaml:"version" json:"version"`
	Rules   []Rule  `yaml:"rules" json:"rules"`
}

// MatchContext 匹配上下文
type MatchContext struct {
	Alert     *domain.Alert
	Device    *domain.Device // 可选的设备信息
	Timestamp time.Time
	Metadata  map[string]interface{} // 额外的上下文信息
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	Alert        *domain.Alert
	Device       *domain.Device
	Rule         *Rule
	Timestamp    time.Time
	PreviousTries int // 之前的尝试次数
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	RuleID       string
	ActionType   ActionType
	Success      bool
	Error        error
	ExecutedAt   time.Time
	Duration     time.Duration
	Metadata     map[string]interface{}
}

// RateLimitKey 速率限制键
type RateLimitKey struct {
	RuleID   string
	DeviceID *uuid.UUID
	Scope    string
}

// EscalationState 升级状态
type EscalationState struct {
	AlertID        uuid.UUID
	RuleID         string
	FirstTriggered time.Time
	LastNotified   time.Time
	RepeatCount    int
	Acknowledged   bool
}
