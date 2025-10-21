package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	EventType string          `json:"event_type"`
	DeviceID  *string         `json:"device_id,omitempty"`
	OrgID     *string         `json:"org_id,omitempty"`
	Data      json.RawMessage `json:"data"`
}

// Broadcaster 事件广播器
type Broadcaster struct {
	redisClient *redis.Client
	wsHandler   *WebSocketHandler
	logger      *zap.Logger
	channelName string
}

// NewBroadcaster 创建事件广播器
func NewBroadcaster(redisClient *redis.Client, wsHandler *WebSocketHandler, logger *zap.Logger) *Broadcaster {
	return &Broadcaster{
		redisClient: redisClient,
		wsHandler:   wsHandler,
		logger:      logger,
		channelName: "edgelink:events",
	}
}

// Start 启动广播器（订阅Redis频道）
func (b *Broadcaster) Start(ctx context.Context) error {
	// 订阅Redis频道
	pubsub := b.redisClient.Subscribe(ctx, b.channelName)
	defer pubsub.Close()

	// 等待订阅确认
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to Redis channel: %w", err)
	}

	b.logger.Info("Broadcaster started",
		zap.String("channel", b.channelName),
	)

	// 接收消息
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			b.logger.Info("Broadcaster shutting down")
			return ctx.Err()

		case msg := <-ch:
			b.handleRedisMessage(msg)
		}
	}
}

// handleRedisMessage 处理从Redis接收的消息
func (b *Broadcaster) handleRedisMessage(msg *redis.Message) {
	var broadcastMsg BroadcastMessage
	if err := json.Unmarshal([]byte(msg.Payload), &broadcastMsg); err != nil {
		b.logger.Error("Failed to unmarshal Redis message",
			zap.Error(err),
			zap.String("payload", msg.Payload),
		)
		return
	}

	// 通过WebSocket广播给客户端
	b.wsHandler.Broadcast(&broadcastMsg)
}

// Publish 发布事件到Redis（供其他服务调用）
func (b *Broadcaster) Publish(ctx context.Context, msg *BroadcastMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %w", err)
	}

	if err := b.redisClient.Publish(ctx, b.channelName, data).Err(); err != nil {
		return fmt.Errorf("failed to publish to Redis: %w", err)
	}

	b.logger.Debug("Event published to Redis",
		zap.String("event_type", msg.EventType),
	)

	return nil
}

// PublishDeviceStatus 发布设备状态变化事件
func (b *Broadcaster) PublishDeviceStatus(ctx context.Context, deviceID, orgID string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return b.Publish(ctx, &BroadcastMessage{
		EventType: MessageTypeDeviceStatus,
		DeviceID:  &deviceID,
		OrgID:     &orgID,
		Data:      jsonData,
	})
}

// PublishAlertCreated 发布告警创建事件
func (b *Broadcaster) PublishAlertCreated(ctx context.Context, orgID string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return b.Publish(ctx, &BroadcastMessage{
		EventType: MessageTypeAlertCreated,
		OrgID:     &orgID,
		Data:      jsonData,
	})
}

// PublishAlertUpdated 发布告警更新事件
func (b *Broadcaster) PublishAlertUpdated(ctx context.Context, orgID string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return b.Publish(ctx, &BroadcastMessage{
		EventType: MessageTypeAlertUpdated,
		OrgID:     &orgID,
		Data:      jsonData,
	})
}

// PublishMetricsUpdate 发布指标更新事件
func (b *Broadcaster) PublishMetricsUpdate(ctx context.Context, deviceID, orgID string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return b.Publish(ctx, &BroadcastMessage{
		EventType: MessageTypeMetricsUpdate,
		DeviceID:  &deviceID,
		OrgID:     &orgID,
		Data:      jsonData,
	})
}

// PublishSessionUpdate 发布会话更新事件
func (b *Broadcaster) PublishSessionUpdate(ctx context.Context, orgID string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return b.Publish(ctx, &BroadcastMessage{
		EventType: MessageTypeSessionUpdate,
		OrgID:     &orgID,
		Data:      jsonData,
	})
}
