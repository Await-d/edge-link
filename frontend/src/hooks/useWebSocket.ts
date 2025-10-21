import { useEffect, useRef, useCallback, useState } from 'react'
import { message } from 'antd'

export interface WebSocketMessage {
  type: string
  timestamp: string
  data?: any
}

export interface WebSocketHookOptions {
  url?: string
  onMessage?: (message: WebSocketMessage) => void
  onConnect?: () => void
  onDisconnect?: () => void
  onError?: (error: Event) => void
  autoReconnect?: boolean
  reconnectInterval?: number
}

export const useWebSocket = (options: WebSocketHookOptions = {}) => {
  const {
    url = import.meta.env.VITE_WS_URL || 'ws://localhost:8080',
    onMessage,
    onConnect,
    onDisconnect,
    onError,
    autoReconnect = true,
    reconnectInterval = 5000,
  } = options

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<NodeJS.Timeout>()
  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null)

  const connect = useCallback(() => {
    try {
      const wsUrl = `${url}/ws`
      wsRef.current = new WebSocket(wsUrl)

      wsRef.current.onopen = () => {
        console.log('WebSocket connected')
        setIsConnected(true)
        onConnect?.()

        // 连接成功后发送ping保持心跳
        const pingInterval = setInterval(() => {
          if (wsRef.current?.readyState === WebSocket.OPEN) {
            send({ type: 'ping' })
          } else {
            clearInterval(pingInterval)
          }
        }, 30000) // 每30秒发送一次ping
      }

      wsRef.current.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          setLastMessage(message)
          onMessage?.(message)

          // 处理pong响应
          if (message.type === 'pong') {
            console.log('Received pong from server')
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }

      wsRef.current.onerror = (error) => {
        console.error('WebSocket error:', error)
        onError?.(error)
        message.error('WebSocket连接错误')
      }

      wsRef.current.onclose = () => {
        console.log('WebSocket disconnected')
        setIsConnected(false)
        onDisconnect?.()

        // 自动重连
        if (autoReconnect) {
          reconnectTimerRef.current = setTimeout(() => {
            console.log('Attempting to reconnect WebSocket...')
            connect()
          }, reconnectInterval)
        }
      }
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error)
    }
  }, [url, onMessage, onConnect, onDisconnect, onError, autoReconnect, reconnectInterval])

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
  }, [])

  const send = useCallback((message: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
    } else {
      console.warn('WebSocket is not connected')
    }
  }, [])

  const subscribe = useCallback((eventTypes: string[], deviceId?: string, orgId?: string) => {
    send({
      type: 'subscribe',
      data: {
        event_types: eventTypes,
        device_id: deviceId,
        org_id: orgId,
      },
    })
  }, [send])

  const unsubscribe = useCallback((eventTypes: string[], deviceId?: string, orgId?: string) => {
    send({
      type: 'unsubscribe',
      data: {
        event_types: eventTypes,
        device_id: deviceId,
        org_id: orgId,
      },
    })
  }, [send])

  useEffect(() => {
    connect()

    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  return {
    isConnected,
    lastMessage,
    send,
    subscribe,
    unsubscribe,
    connect,
    disconnect,
  }
}
