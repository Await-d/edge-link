// 设备相关类型
export interface Device {
  id: string
  virtual_network_id: string
  name: string
  platform: 'linux' | 'windows' | 'macos' | 'android' | 'ios'
  public_key: string
  virtual_ip: string
  endpoint?: string
  nat_type?: string
  is_online: boolean
  last_seen_at: string
  created_at: string
  updated_at: string
}

// 虚拟网络类型
export interface VirtualNetwork {
  id: string
  organization_id: string
  name: string
  cidr: string
  gateway_ip: string
  dns_servers: string[]
  created_at: string
  updated_at: string
}

// 告警类型
export type Severity = 'critical' | 'high' | 'medium' | 'low'
export type AlertStatus = 'active' | 'acknowledged' | 'resolved'
export type AlertType = 'device_offline' | 'high_latency' | 'connection_failed' | 'key_expired' | 'system'

export interface Alert {
  id: string
  device_id?: string
  severity: Severity
  alert_type: AlertType
  title: string
  message: string
  status: AlertStatus
  metadata?: Record<string, any>
  acknowledged_at?: string
  acknowledged_by?: string
  resolved_at?: string
  created_at: string
}

// 审计日志类型
export type ResourceType = 'device' | 'virtual_network' | 'pre_shared_key' | 'alert' | 'organization'

export interface AuditLog {
  id: string
  organization_id: string
  actor_id?: string
  action: string
  resource_type: ResourceType
  resource_id: string
  before_state?: Record<string, any>
  after_state?: Record<string, any>
  ip_address?: string
  user_agent?: string
  created_at: string
}

// 会话类型
export type ConnectionType = 'p2p_direct' | 'turn_relay'
export type SessionStatus = 'active' | 'ended'

export interface Session {
  id: string
  device_a_id: string
  device_b_id: string
  connection_type: ConnectionType
  status: SessionStatus
  started_at: string
  ended_at?: string
  bytes_sent: number
  bytes_received: number
  avg_latency_ms?: number
}

// 分页响应
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  limit: number
  offset: number
}

// API错误响应
export interface ErrorResponse {
  error: string
  message: string
}

// 统计数据
export interface DashboardStats {
  total_devices: number
  online_devices: number
  active_sessions: number
  active_alerts: number
  p2p_success_rate: number
}

export interface DeviceMetrics {
  device_id: string
  timestamp: string
  cpu_usage?: number
  memory_usage?: number
  network_tx_bytes: number
  network_rx_bytes: number
  latency_ms?: number
}
