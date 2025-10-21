package rules

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/edgelink/backend/internal/domain"
)

// Matcher 条件匹配器
type Matcher struct {
	regexCache map[string]*regexp.Regexp
}

// NewMatcher 创建条件匹配器
func NewMatcher() *Matcher {
	return &Matcher{
		regexCache: make(map[string]*regexp.Regexp),
	}
}

// Match 匹配规则条件
func (m *Matcher) Match(rule *Rule, ctx *MatchContext) bool {
	// 检查规则是否启用
	if !rule.Enabled {
		return false
	}

	// 检查静默规则
	if rule.Silence != nil && rule.Silence.Enabled {
		if m.isInSilencePeriod(rule.Silence, ctx.Timestamp) {
			return false
		}
	}

	// 匹配条件
	return m.matchConditions(&rule.Conditions, ctx)
}

// matchConditions 匹配条件组
func (m *Matcher) matchConditions(cond *Conditions, ctx *MatchContext) bool {
	alert := ctx.Alert

	// 处理逻辑组合
	if len(cond.AllOf) > 0 {
		for _, subCond := range cond.AllOf {
			if !m.matchConditions(&subCond, ctx) {
				return false // AND逻辑，任一不满足则失败
			}
		}
	}

	if len(cond.AnyOf) > 0 {
		matched := false
		for _, subCond := range cond.AnyOf {
			if m.matchConditions(&subCond, ctx) {
				matched = true
				break // OR逻辑，任一满足即成功
			}
		}
		if !matched {
			return false
		}
	}

	if len(cond.NoneOf) > 0 {
		for _, subCond := range cond.NoneOf {
			if m.matchConditions(&subCond, ctx) {
				return false // NOT逻辑，任一满足则失败
			}
		}
	}

	// 匹配严重程度
	if len(cond.Severity) > 0 {
		if !m.matchSeverity(cond.Severity, alert.Severity) {
			return false
		}
	}

	// 匹配告警类型
	if len(cond.AlertTypes) > 0 {
		if !m.matchAlertType(cond.AlertTypes, alert.Type) {
			return false
		}
	}

	// 匹配设备ID
	if len(cond.DeviceIDs) > 0 && alert.DeviceID != nil {
		if !m.matchDeviceID(cond.DeviceIDs, alert.DeviceID.String()) {
			return false
		}
	}

	// 匹配设备标签
	if len(cond.DeviceTags) > 0 && ctx.Device != nil {
		if !m.matchDeviceTags(cond.DeviceTags, ctx.Device) {
			return false
		}
	}

	// 匹配时间范围
	if cond.TimeRange != nil {
		if !m.matchTimeRange(cond.TimeRange, ctx.Timestamp) {
			return false
		}
	}

	// 匹配消息内容
	if cond.MessageMatch != "" {
		if !m.matchMessage(cond.MessageMatch, alert.Message) {
			return false
		}
	}

	// 匹配元数据
	if len(cond.Metadata) > 0 {
		if !m.matchMetadata(cond.Metadata, alert.Metadata) {
			return false
		}
	}

	return true
}

// matchSeverity 匹配严重程度
func (m *Matcher) matchSeverity(allowed []domain.Severity, severity domain.Severity) bool {
	for _, s := range allowed {
		if s == severity {
			return true
		}
	}
	return false
}

// matchAlertType 匹配告警类型
func (m *Matcher) matchAlertType(allowed []domain.AlertType, alertType domain.AlertType) bool {
	for _, t := range allowed {
		if t == alertType {
			return true
		}
	}
	return false
}

// matchDeviceID 匹配设备ID
func (m *Matcher) matchDeviceID(allowed []string, deviceID string) bool {
	for _, id := range allowed {
		if id == deviceID || id == "*" {
			return true
		}
	}
	return false
}

// matchDeviceTags 匹配设备标签
func (m *Matcher) matchDeviceTags(requiredTags []string, device *domain.Device) bool {
	// 将设备标签转换为map便于查找
	deviceTags := make(map[string]bool)
	for _, tag := range device.Tags {
		deviceTags[tag] = true
	}

	// 检查所有必需标签是否存在
	for _, tag := range requiredTags {
		if !deviceTags[tag] {
			return false
		}
	}

	return true
}

// matchTimeRange 匹配时间范围
func (m *Matcher) matchTimeRange(tr *TimeRange, timestamp time.Time) bool {
	// 加载时区
	loc := time.Local
	if tr.Timezone != "" {
		if l, err := time.LoadLocation(tr.Timezone); err == nil {
			loc = l
		}
	}

	// 转换到指定时区
	t := timestamp.In(loc)

	// 检查星期
	if len(tr.Weekdays) > 0 {
		weekday := t.Weekday().String()
		matched := false
		for _, wd := range tr.Weekdays {
			if wd == weekday {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查时间范围
	currentTime := t.Format("15:04")

	// 处理跨天情况 (如 22:00-02:00)
	if tr.StartTime <= tr.EndTime {
		// 正常范围 (如 09:00-18:00)
		return currentTime >= tr.StartTime && currentTime <= tr.EndTime
	} else {
		// 跨天范围
		return currentTime >= tr.StartTime || currentTime <= tr.EndTime
	}
}

// matchMessage 匹配消息内容
func (m *Matcher) matchMessage(pattern, message string) bool {
	// 获取或编译正则表达式
	regex, ok := m.regexCache[pattern]
	if !ok {
		var err error
		regex, err = regexp.Compile(pattern)
		if err != nil {
			return false // 正则表达式无效，不匹配
		}
		m.regexCache[pattern] = regex
	}

	return regex.MatchString(message)
}

// matchMetadata 匹配元数据
func (m *Matcher) matchMetadata(required map[string]string, metadata domain.JSONB) bool {
	for key, expectedValue := range required {
		actualValue, ok := metadata[key]
		if !ok {
			return false // 键不存在
		}

		// 转换为字符串进行比较
		actualStr := ""
		switch v := actualValue.(type) {
		case string:
			actualStr = v
		case float64:
			actualStr = formatFloat(v)
		case bool:
			if v {
				actualStr = "true"
			} else {
				actualStr = "false"
			}
		default:
			actualStr = ""
		}

		if actualStr != expectedValue {
			return false // 值不匹配
		}
	}

	return true
}

// isInSilencePeriod 检查是否在静默期
func (m *Matcher) isInSilencePeriod(silence *SilenceRule, timestamp time.Time) bool {
	for _, tr := range silence.TimeRanges {
		if m.matchTimeRange(&tr, timestamp) {
			return true
		}
	}
	return false
}

// MatchMultiple 匹配多个规则并按优先级排序
func (m *Matcher) MatchMultiple(rules []Rule, ctx *MatchContext) []Rule {
	var matched []Rule

	for _, rule := range rules {
		if m.Match(&rule, ctx) {
			matched = append(matched, rule)
		}
	}

	// 按优先级排序 (数字越小优先级越高)
	for i := 0; i < len(matched); i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[j].Priority < matched[i].Priority {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}

	return matched
}

// formatFloat 格式化浮点数为字符串
func formatFloat(v float64) string {
	s := fmt.Sprintf("%.10f", v)
	// 去除尾随的0
	s = strings.TrimRight(s, "0")
	// 如果最后是小数点，也去掉
	s = strings.TrimRight(s, ".")
	return s
}

