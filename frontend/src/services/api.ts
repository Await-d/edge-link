import apiClient from './apiClient'
import type {
  Device,
  VirtualNetwork,
  Alert,
  AuditLog,
  PaginatedResponse,
} from '@/types/api'

// 设备API
export const deviceApi = {
  // 获取设备列表
  getDevices: async (params?: {
    virtual_network_id?: string
    online?: boolean
    platform?: string
    limit?: number
    offset?: number
  }) => {
    const response = await apiClient.get<PaginatedResponse<Device>>(
      '/v1/admin/devices',
      { params }
    )
    return response.data
  },

  // 删除设备
  deleteDevice: async (deviceId: string) => {
    const response = await apiClient.delete(`/v1/admin/devices/${deviceId}`)
    return response.data
  },
}

// 虚拟网络API
export const networkApi = {
  // 获取虚拟网络列表
  getNetworks: async (organizationId?: string) => {
    const response = await apiClient.get<{
      networks: VirtualNetwork[]
      total: number
    }>('/v1/admin/virtual-networks', {
      params: organizationId ? { organization_id: organizationId } : undefined,
    })
    return response.data
  },

  // 创建虚拟网络
  createNetwork: async (data: {
    organization_id: string
    name: string
    cidr: string
    gateway_ip: string
    dns_servers?: string[]
  }) => {
    const response = await apiClient.post<VirtualNetwork>(
      '/v1/admin/virtual-networks',
      data
    )
    return response.data
  },
}

// 告警API
export const alertApi = {
  // 获取告警列表
  getAlerts: async (params?: {
    device_id?: string
    severity?: string
    status?: string
    type?: string
    limit?: number
    offset?: number
  }) => {
    const response = await apiClient.get<PaginatedResponse<Alert>>(
      '/v1/admin/alerts',
      { params }
    )
    return response.data
  },

  // 获取告警详情
  getAlertById: async (alertId: string) => {
    const response = await apiClient.get<Alert>(`/v1/admin/alerts/${alertId}`)
    return response.data
  },

  // 确认告警
  acknowledgeAlert: async (alertId: string, acknowledgedBy: string) => {
    const response = await apiClient.put(
      `/v1/admin/alerts/${alertId}/acknowledge`,
      { acknowledged_by: acknowledgedBy }
    )
    return response.data
  },

  // 解决告警
  resolveAlert: async (alertId: string) => {
    const response = await apiClient.put(
      `/v1/admin/alerts/${alertId}/resolve`
    )
    return response.data
  },
}

// 审计日志API
export const auditApi = {
  // 获取审计日志
  getAuditLogs: async (params?: {
    organization_id?: string
    actor_id?: string
    action?: string
    resource_type?: string
    resource_id?: string
    start_time?: string
    end_time?: string
    limit?: number
    offset?: number
  }) => {
    const response = await apiClient.get<PaginatedResponse<AuditLog>>(
      '/v1/admin/audit-logs',
      { params }
    )
    return response.data
  },
}
