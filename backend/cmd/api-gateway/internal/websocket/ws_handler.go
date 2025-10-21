package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocket 消息类型
const (
	// 客户端 -> 服务器
	MessageTypeSubscribe   = "subscribe"
	MessageTypeUnsubscribe = "unsubscribe"
	MessageTypePing        = "ping"

	// 服务器 -> 客户端
	MessageTypePong            = "pong"
	MessageTypeDeviceStatus    = "device_status"
	MessageTypeAlertCreated    = "alert_created"
	MessageTypeAlertUpdated    = "alert_updated"
	MessageTypeMetricsUpdate   = "metrics_update"
	MessageTypeSessionUpdate   = "session_update"
	MessageTypeError           = "error"
)

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	EventTypes []string `json:"event_types"` // 订阅的事件类型
	DeviceID   *string  `json:"device_id,omitempty"`   // 可选：只订阅特定设备
	OrgID      *string  `json:"org_id,omitempty"`      // 可选：只订阅特定组织
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Client 表示一个WebSocket客户端连接
type Client struct {
	ID             string
	Conn           *websocket.Conn
	Send           chan *WebSocketMessage
	Subscriptions  *SubscriptionManager
	logger         *zap.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	lastPing       time.Time
	lastPong       time.Time
}

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
	logger     *zap.Logger
	upgrader   websocket.Upgrader
}

// NewWebSocketHandler 创建WebSocket处理器
func NewWebSocketHandler(logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *BroadcastMessage, 1024),
		logger:     logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: 生产环境需要检查Origin
				return true
			},
		},
	}
}

// Run 启动WebSocket处理器事件循环
func (h *WebSocketHandler) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("WebSocket handler shutting down")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			h.logger.Info("Client registered",
				zap.String("client_id", client.ID),
				zap.Int("total_clients", len(h.clients)),
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
			h.logger.Info("Client unregistered",
				zap.String("client_id", client.ID),
				zap.Int("total_clients", len(h.clients)),
			)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// HandleWebSocket 处理WebSocket连接升级
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 升级HTTP连接为WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// 创建客户端
	ctx, cancel := context.WithCancel(c.Request.Context())
	client := &Client{
		ID:            uuid.New().String(),
		Conn:          conn,
		Send:          make(chan *WebSocketMessage, 256),
		Subscriptions: NewSubscriptionManager(),
		logger:        h.logger,
		ctx:           ctx,
		cancel:        cancel,
		lastPing:      time.Now(),
		lastPong:      time.Now(),
	}

	// 注册客户端
	h.register <- client

	// 启动读写goroutines
	go client.writePump()
	go client.readPump(h)
}

// readPump 从WebSocket连接读取消息
func (c *Client) readPump(h *WebSocketHandler) {
	defer func() {
		h.unregister <- c
		c.Conn.Close()
		c.cancel()
	}()

	// 设置读取配置
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.lastPong = time.Now()
		c.mu.Unlock()
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg WebSocketMessage
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		// 处理消息
		c.handleMessage(h, &msg)
	}
}

// writePump 向WebSocket连接写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				c.logger.Error("Failed to write message", zap.Error(err))
				return
			}

		case <-ticker.C:
			// 发送ping
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			c.mu.Lock()
			c.lastPing = time.Now()
			c.mu.Unlock()
		}
	}
}

// handleMessage 处理客户端消息
func (c *Client) handleMessage(h *WebSocketHandler, msg *WebSocketMessage) {
	switch msg.Type {
	case MessageTypeSubscribe:
		var req SubscribeRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			c.sendError("invalid_subscribe_request", err.Error())
			return
		}
		c.handleSubscribe(&req)

	case MessageTypeUnsubscribe:
		var req SubscribeRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			c.sendError("invalid_unsubscribe_request", err.Error())
			return
		}
		c.handleUnsubscribe(&req)

	case MessageTypePing:
		c.Send <- &WebSocketMessage{
			Type:      MessageTypePong,
			Timestamp: time.Now(),
		}

	default:
		c.sendError("unknown_message_type", fmt.Sprintf("unknown message type: %s", msg.Type))
	}
}

// handleSubscribe 处理订阅请求
func (c *Client) handleSubscribe(req *SubscribeRequest) {
	for _, eventType := range req.EventTypes {
		filter := &SubscriptionFilter{
			EventType: eventType,
		}
		if req.DeviceID != nil {
			filter.DeviceID = req.DeviceID
		}
		if req.OrgID != nil {
			filter.OrgID = req.OrgID
		}
		c.Subscriptions.Subscribe(filter)
	}

	c.logger.Info("Client subscribed",
		zap.String("client_id", c.ID),
		zap.Strings("event_types", req.EventTypes),
	)
}

// handleUnsubscribe 处理取消订阅请求
func (c *Client) handleUnsubscribe(req *SubscribeRequest) {
	for _, eventType := range req.EventTypes {
		filter := &SubscriptionFilter{
			EventType: eventType,
		}
		if req.DeviceID != nil {
			filter.DeviceID = req.DeviceID
		}
		if req.OrgID != nil {
			filter.OrgID = req.OrgID
		}
		c.Subscriptions.Unsubscribe(filter)
	}

	c.logger.Info("Client unsubscribed",
		zap.String("client_id", c.ID),
		zap.Strings("event_types", req.EventTypes),
	)
}

// sendError 发送错误消息
func (c *Client) sendError(code, message string) {
	errData, _ := json.Marshal(ErrorResponse{
		Code:    code,
		Message: message,
	})

	c.Send <- &WebSocketMessage{
		Type:      MessageTypeError,
		Timestamp: time.Now(),
		Data:      errData,
	}
}

// broadcastMessage 广播消息给匹配的客户端
func (h *WebSocketHandler) broadcastMessage(msg *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sentCount := 0
	for _, client := range h.clients {
		if client.Subscriptions.Matches(msg.EventType, msg.DeviceID, msg.OrgID) {
			select {
			case client.Send <- &WebSocketMessage{
				Type:      msg.EventType,
				Timestamp: time.Now(),
				Data:      msg.Data,
			}:
				sentCount++
			default:
				// 客户端发送缓冲区满，跳过
				h.logger.Warn("Client send buffer full, skipping message",
					zap.String("client_id", client.ID),
				)
			}
		}
	}

	h.logger.Debug("Broadcast message sent",
		zap.String("event_type", msg.EventType),
		zap.Int("recipients", sentCount),
	)
}

// Broadcast 广播消息（公共方法）
func (h *WebSocketHandler) Broadcast(msg *BroadcastMessage) {
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("Broadcast channel full, message dropped",
			zap.String("event_type", msg.EventType),
		)
	}
}

// GetClientCount 获取当前连接的客户端数量
func (h *WebSocketHandler) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
