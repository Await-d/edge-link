export interface WebSocketMessage {
  type: string
  timestamp: string
  data?: any
}

export interface WebSocketConfig {
  url: string
  reconnectInterval?: number
  maxReconnectAttempts?: number
}

export class WebSocketClient {
  private ws: WebSocket | null = null
  private config: Required<WebSocketConfig>
  private reconnectAttempts = 0
  private isManualClose = false
  private subscribers = new Map<string, Set<(data: any) => void>>()

  constructor(config: WebSocketConfig) {
    this.config = {
      url: config.url,
      reconnectInterval: config.reconnectInterval || 5000,
      maxReconnectAttempts: config.maxReconnectAttempts || 10,
    }
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.config.url)
        this.isManualClose = false

        this.ws.onopen = () => {
          console.log('WebSocket connected')
          this.reconnectAttempts = 0
          resolve()
        }

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data)
        }

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason)
          if (!this.isManualClose && this.reconnectAttempts < this.config.maxReconnectAttempts) {
            this.scheduleReconnect()
          }
        }

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error)
          reject(error)
        }
      } catch (error) {
        reject(error)
      }
    })
  }

  disconnect(): void {
    this.isManualClose = true
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  subscribe(eventType: string, callback: (data: any) => void): () => void {
    if (!this.subscribers.has(eventType)) {
      this.subscribers.set(eventType, new Set())
    }

    const subscribers = this.subscribers.get(eventType)!
    subscribers.add(callback)

    // 发送订阅消息
    this.send({
      type: 'subscribe',
      event_types: [eventType],
    })

    // 返回取消订阅函数
    return () => {
      subscribers.delete(callback)
      if (subscribers.size === 0) {
        this.subscribers.delete(eventType)
        // 发送取消订阅消息
        this.send({
          type: 'unsubscribe',
          event_types: [eventType],
        })
      }
    }
  }

  private handleMessage(data: string): void {
    try {
      const message: WebSocketMessage = JSON.parse(data)

      // 处理pong消息
      if (message.type === 'pong') {
        return
      }

      // 通知订阅者
      const subscribers = this.subscribers.get(message.type)
      if (subscribers) {
        subscribers.forEach(callback => {
          try {
            callback(message.data)
          } catch (error) {
            console.error('Error in WebSocket callback:', error)
          }
        })
      }
    } catch (error) {
      console.error('Error parsing WebSocket message:', error)
    }
  }

  private send(message: any): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    }
  }

  private scheduleReconnect(): void {
    this.reconnectAttempts++
    console.log(`Scheduling reconnect attempt ${this.reconnectAttempts}/${this.config.maxReconnectAttempts}`)

    setTimeout(() => {
      if (!this.isManualClose) {
        this.connect().catch(error => {
          console.error('Reconnect failed:', error)
        })
      }
    }, this.config.reconnectInterval)
  }

  // 公共方法
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN
  }

  ping(): void {
    this.send({ type: 'ping' })
  }

  // 静态方法创建实例
  static create(baseUrl?: string): WebSocketClient {
    const wsUrl = (baseUrl || window.location.origin).replace(/^http/, 'ws') + '/ws'
    return new WebSocketClient({ url: wsUrl })
  }
}

// 默认WebSocket客户端实例
export const wsClient = WebSocketClient.create()

// 导出常用事件类型
export const WebSocketEventTypes = {
  DEVICE_STATUS: 'device_status',
  ALERT_CREATED: 'alert_created',
  ALERT_UPDATED: 'alert_updated',
  METRICS_UPDATE: 'metrics_update',
  SESSION_UPDATE: 'session_update',
  ERROR: 'error',
} as const