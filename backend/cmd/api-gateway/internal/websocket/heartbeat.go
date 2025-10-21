package websocket

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const (
	// 心跳间隔
	HeartbeatInterval = 30 * time.Second

	// 最大未响应时间（超过此时间未收到pong认为连接已断开）
	MaxUnresponsiveTime = 90 * time.Second
)

// HeartbeatMonitor 心跳监控器
type HeartbeatMonitor struct {
	handler *WebSocketHandler
	logger  *zap.Logger
}

// NewHeartbeatMonitor 创建心跳监控器
func NewHeartbeatMonitor(handler *WebSocketHandler, logger *zap.Logger) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		handler: handler,
		logger:  logger,
	}
}

// Start 启动心跳监控
func (hm *HeartbeatMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	hm.logger.Info("Heartbeat monitor started",
		zap.Duration("interval", HeartbeatInterval),
		zap.Duration("max_unresponsive", MaxUnresponsiveTime),
	)

	for {
		select {
		case <-ctx.Done():
			hm.logger.Info("Heartbeat monitor shutting down")
			return

		case <-ticker.C:
			hm.checkClients()
		}
	}
}

// checkClients 检查所有客户端的心跳状态
func (hm *HeartbeatMonitor) checkClients() {
	hm.handler.mu.RLock()
	clients := make([]*Client, 0, len(hm.handler.clients))
	for _, client := range hm.handler.clients {
		clients = append(clients, client)
	}
	hm.handler.mu.RUnlock()

	now := time.Now()
	deadClients := 0

	for _, client := range clients {
		client.mu.RLock()
		lastPong := client.lastPong
		client.mu.RUnlock()

		// 检查是否超时未响应
		if now.Sub(lastPong) > MaxUnresponsiveTime {
			hm.logger.Warn("Client unresponsive, disconnecting",
				zap.String("client_id", client.ID),
				zap.Duration("unresponsive_time", now.Sub(lastPong)),
			)

			// 关闭客户端连接
			client.cancel()
			deadClients++
		}
	}

	if deadClients > 0 {
		hm.logger.Info("Heartbeat check completed",
			zap.Int("total_clients", len(clients)),
			zap.Int("dead_clients", deadClients),
		)
	}
}

// GetClientHealthStatus 获取客户端健康状态
func (hm *HeartbeatMonitor) GetClientHealthStatus(clientID string) *ClientHealthStatus {
	hm.handler.mu.RLock()
	client, exists := hm.handler.clients[clientID]
	hm.handler.mu.RUnlock()

	if !exists {
		return nil
	}

	client.mu.RLock()
	defer client.mu.RUnlock()

	now := time.Now()
	return &ClientHealthStatus{
		ClientID:          client.ID,
		LastPing:          client.lastPing,
		LastPong:          client.lastPong,
		UnresponsiveTime:  now.Sub(client.lastPong),
		IsHealthy:         now.Sub(client.lastPong) < MaxUnresponsiveTime,
		SubscriptionCount: client.Subscriptions.GetSubscriptionCount(),
	}
}

// GetAllClientHealthStatus 获取所有客户端健康状态
func (hm *HeartbeatMonitor) GetAllClientHealthStatus() []*ClientHealthStatus {
	hm.handler.mu.RLock()
	clients := make([]*Client, 0, len(hm.handler.clients))
	for _, client := range hm.handler.clients {
		clients = append(clients, client)
	}
	hm.handler.mu.RUnlock()

	statuses := make([]*ClientHealthStatus, 0, len(clients))
	for _, client := range clients {
		if status := hm.GetClientHealthStatus(client.ID); status != nil {
			statuses = append(statuses, status)
		}
	}

	return statuses
}

// ClientHealthStatus 客户端健康状态
type ClientHealthStatus struct {
	ClientID          string        `json:"client_id"`
	LastPing          time.Time     `json:"last_ping"`
	LastPong          time.Time     `json:"last_pong"`
	UnresponsiveTime  time.Duration `json:"unresponsive_time"`
	IsHealthy         bool          `json:"is_healthy"`
	SubscriptionCount int           `json:"subscription_count"`
}
