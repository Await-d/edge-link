package websocket

import (
	"sync"
)

// SubscriptionFilter 订阅过滤器
type SubscriptionFilter struct {
	EventType string   // 事件类型
	DeviceID  *string  // 可选：设备ID过滤
	OrgID     *string  // 可选：组织ID过滤
}

// SubscriptionManager 订阅管理器
type SubscriptionManager struct {
	subscriptions map[string]*SubscriptionFilter // key: eventType
	mu            sync.RWMutex
}

// NewSubscriptionManager 创建订阅管理器
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string]*SubscriptionFilter),
	}
}

// Subscribe 添加订阅
func (sm *SubscriptionManager) Subscribe(filter *SubscriptionFilter) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.makeKey(filter)
	sm.subscriptions[key] = filter
}

// Unsubscribe 取消订阅
func (sm *SubscriptionManager) Unsubscribe(filter *SubscriptionFilter) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.makeKey(filter)
	delete(sm.subscriptions, key)
}

// Matches 检查消息是否匹配任何订阅
func (sm *SubscriptionManager) Matches(eventType string, deviceID, orgID *string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, filter := range sm.subscriptions {
		if sm.filterMatches(filter, eventType, deviceID, orgID) {
			return true
		}
	}
	return false
}

// filterMatches 检查单个过滤器是否匹配
func (sm *SubscriptionManager) filterMatches(filter *SubscriptionFilter, eventType string, deviceID, orgID *string) bool {
	// 检查事件类型
	if filter.EventType != eventType {
		return false
	}

	// 检查设备ID过滤（如果有）
	if filter.DeviceID != nil {
		if deviceID == nil || *filter.DeviceID != *deviceID {
			return false
		}
	}

	// 检查组织ID过滤（如果有）
	if filter.OrgID != nil {
		if orgID == nil || *filter.OrgID != *orgID {
			return false
		}
	}

	return true
}

// makeKey 生成订阅键
func (sm *SubscriptionManager) makeKey(filter *SubscriptionFilter) string {
	key := filter.EventType
	if filter.DeviceID != nil {
		key += ":device:" + *filter.DeviceID
	}
	if filter.OrgID != nil {
		key += ":org:" + *filter.OrgID
	}
	return key
}

// GetSubscriptionCount 获取订阅数量
func (sm *SubscriptionManager) GetSubscriptionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.subscriptions)
}

// GetSubscriptions 获取所有订阅（用于调试）
func (sm *SubscriptionManager) GetSubscriptions() []SubscriptionFilter {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]SubscriptionFilter, 0, len(sm.subscriptions))
	for _, filter := range sm.subscriptions {
		result = append(result, *filter)
	}
	return result
}
