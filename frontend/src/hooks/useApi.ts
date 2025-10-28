import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useCallback } from 'react'
import { deviceApi, networkApi, alertApi, auditApi, statsApi, topologyApi } from '@/services/api'
import { wsClient, WebSocketEventTypes } from '@/services/websocketClient'
import { message } from 'antd'
import { transformToTopology } from '@/utils/topology'

// 设备相关hooks
export const useDevices = (params?: Parameters<typeof deviceApi.getDevices>[0]) => {
  return useQuery({
    queryKey: ['devices', params],
    queryFn: () => deviceApi.getDevices(params),
  })
}

export const useDevice = (deviceId: string) => {
  return useQuery({
    queryKey: ['device', deviceId],
    queryFn: () => deviceApi.getDeviceById(deviceId),
    enabled: !!deviceId,
  })
}

export const useDevicePeers = (deviceId: string) => {
  return useQuery({
    queryKey: ['device-peers', deviceId],
    queryFn: () => deviceApi.getDevicePeers(deviceId),
    enabled: !!deviceId,
  })
}

export const useDeviceMetrics = (deviceId: string, timeRange: string = '24h') => {
  return useQuery({
    queryKey: ['device-metrics', deviceId, timeRange],
    queryFn: () => deviceApi.getDeviceMetrics(deviceId, timeRange),
    enabled: !!deviceId,
  })
}

export const useDeleteDevice = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deviceApi.deleteDevice,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      message.success('设备删除成功')
    },
    onError: (error: any) => {
      message.error(error.response?.data?.message || '设备删除失败')
    },
  })
}

export const useRestartDevice = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deviceApi.restartDevice,
    onSuccess: (_, deviceId) => {
      queryClient.invalidateQueries({ queryKey: ['device', deviceId] })
      message.success('设备重启指令已发送')
    },
    onError: (error: any) => {
      message.error(error.response?.data?.message || '设备重启失败')
    },
  })
}

// 虚拟网络相关hooks
export const useNetworks = (organizationId?: string) => {
  return useQuery({
    queryKey: ['networks', organizationId],
    queryFn: () => networkApi.getNetworks(organizationId),
  })
}

export const useCreateNetwork = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: networkApi.createNetwork,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['networks'] })
      message.success('虚拟网络创建成功')
    },
    onError: (error: any) => {
      message.error(error.response?.data?.message || '虚拟网络创建失败')
    },
  })
}

// 告警相关hooks
export const useAlerts = (params?: Parameters<typeof alertApi.getAlerts>[0]) => {
  return useQuery({
    queryKey: ['alerts', params],
    queryFn: () => alertApi.getAlerts(params),
    refetchInterval: 30000, // 自动每30秒刷新一次
  })
}

export const useAlertById = (alertId: string) => {
  return useQuery({
    queryKey: ['alert', alertId],
    queryFn: () => alertApi.getAlertById(alertId),
    enabled: !!alertId,
  })
}

export const useAcknowledgeAlert = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ alertId, acknowledgedBy }: { alertId: string; acknowledgedBy: string }) =>
      alertApi.acknowledgeAlert(alertId, acknowledgedBy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
      message.success('告警已确认')
    },
    onError: (error: any) => {
      message.error(error.response?.data?.message || '告警确认失败')
    },
  })
}

export const useResolveAlert = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (alertId: string) => alertApi.resolveAlert(alertId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
      message.success('告警已解决')
    },
    onError: (error: any) => {
      message.error(error.response?.data?.message || '告警解决失败')
    },
  })
}

// 审计日志相关hooks
export const useAuditLogs = (params?: Parameters<typeof auditApi.getAuditLogs>[0]) => {
  return useQuery({
    queryKey: ['audit-logs', params],
    queryFn: () => auditApi.getAuditLogs(params),
  })
}

// 统计数据相关hooks
export const useDashboardStats = () => {
  return useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: () => statsApi.getDashboardStats(),
    refetchInterval: 30000, // 每30秒刷新
    refetchIntervalInBackground: true,
  })
}

export const useDeviceTrend = (timeRange: string = '24h') => {
  return useQuery({
    queryKey: ['device-trend', timeRange],
    queryFn: () => statsApi.getDeviceTrend(timeRange),
    refetchInterval: 30000,
    refetchIntervalInBackground: true,
  })
}

export const useTrafficStats = (timeRange: string = '24h') => {
  return useQuery({
    queryKey: ['traffic-stats', timeRange],
    queryFn: () => statsApi.getTrafficStats(timeRange),
    refetchInterval: 30000,
    refetchIntervalInBackground: true,
  })
}

export const usePlatformDistribution = () => {
  return useQuery({
    queryKey: ['platform-distribution'],
    queryFn: () => statsApi.getPlatformDistribution(),
    refetchInterval: 30000,
    refetchIntervalInBackground: true,
  })
}

export const useAlertTrend = (timeRange: string = '7d') => {
  return useQuery({
    queryKey: ['alert-trend', timeRange],
    queryFn: () => statsApi.getAlertTrend(timeRange),
    refetchInterval: 30000,
    refetchIntervalInBackground: true,
  })
}

// 拓扑相关hooks
export const useTopologyData = () => {
  return useQuery({
    queryKey: ['topology'],
    queryFn: async () => {
      // 并行获取设备和对等配置数据
      const [devicesResponse, peersResponse] = await Promise.all([
        topologyApi.getAllDevices(),
        topologyApi.getAllPeerConfigurations(),
      ])

      // 转换为拓扑格式
      const devices = Array.isArray(devicesResponse)
        ? devicesResponse
        : devicesResponse.data || []
      return transformToTopology(devices, peersResponse.peers || [])
    },
    refetchInterval: 10000, // 每10秒刷新
    refetchIntervalInBackground: true,
  })
}

// WebSocket实时更新hooks
export const useWebSocketConnection = () => {
  useEffect(() => {
    // 组件挂载时连接WebSocket
    wsClient.connect().catch(console.error)

    // 组件卸载时断开连接
    return () => {
      wsClient.disconnect()
    }
  }, [])
}

export const useDeviceStatusUpdates = (onDeviceStatusUpdate: (data: any) => void) => {
  useWebSocketConnection()

  useEffect(() => {
    const unsubscribe = wsClient.subscribe(
      WebSocketEventTypes.DEVICE_STATUS,
      onDeviceStatusUpdate
    )

    return unsubscribe
  }, [onDeviceStatusUpdate])
}

export const useAlertUpdates = (onAlertCreated: (data: any) => void, onAlertUpdated?: (data: any) => void) => {
  useWebSocketConnection()

  useEffect(() => {
    const unsubscribes: (() => void)[] = []

    unsubscribes.push(
      wsClient.subscribe(WebSocketEventTypes.ALERT_CREATED, onAlertCreated)
    )

    if (onAlertUpdated) {
      unsubscribes.push(
        wsClient.subscribe(WebSocketEventTypes.ALERT_UPDATED, onAlertUpdated)
      )
    }

    return () => {
      unsubscribes.forEach(unsubscribe => unsubscribe())
    }
  }, [onAlertCreated, onAlertUpdated])
}

export const useMetricsUpdates = (onMetricsUpdate: (data: any) => void) => {
  useWebSocketConnection()

  useEffect(() => {
    const unsubscribe = wsClient.subscribe(
      WebSocketEventTypes.METRICS_UPDATE,
      onMetricsUpdate
    )

    return unsubscribe
  }, [onMetricsUpdate])
}

export const useSessionUpdates = (onSessionUpdate: (data: any) => void) => {
  useWebSocketConnection()

  useEffect(() => {
    const unsubscribe = wsClient.subscribe(
      WebSocketEventTypes.SESSION_UPDATE,
      onSessionUpdate
    )

    return unsubscribe
  }, [onSessionUpdate])
}

// 实时数据hooks - 结合React Query和WebSocket
export const useRealTimeDevices = () => {
  const queryClient = useQueryClient()

  // 监听设备状态更新
  useDeviceStatusUpdates((data) => {
    // 更新查询缓存
    queryClient.setQueryData(['devices'], (oldData: any) => {
      if (!oldData?.devices) return oldData

      const updatedDevices = oldData.devices.map((device: any) =>
        device.id === data.device_id ? { ...device, ...data } : device
      )

      return { ...oldData, devices: updatedDevices }
    })
  })

  return useDevices()
}

export const useRealTimeAlerts = () => {
  const queryClient = useQueryClient()

  // 监听告警创建和更新
  useAlertUpdates(
    (data) => {
      // 新告警创建，添加到列表
      queryClient.setQueryData(['alerts'], (oldData: any) => {
        if (!oldData?.alerts) return oldData
        return {
          ...oldData,
          alerts: [data, ...oldData.alerts],
          total: oldData.total + 1,
        }
      })

      // 显示通知
      message.warning(`新告警: ${data.message}`)
    },
    (data) => {
      // 告警更新，修改现有数据
      queryClient.setQueryData(['alerts'], (oldData: any) => {
        if (!oldData?.alerts) return oldData

        const updatedAlerts = oldData.alerts.map((alert: any) =>
          alert.id === data.id ? { ...alert, ...data } : alert
        )

        return { ...oldData, alerts: updatedAlerts }
      })

      // 如果告警被确认或解决，显示通知
      if (data.status === 'acknowledged' || data.status === 'resolved') {
        message.success(`告警已${data.status === 'acknowledged' ? '确认' : '解决'}`)
      }
    }
  )

  return useAlerts()
}

export const useRealTimeTopology = () => {
  const queryClient = useQueryClient()

  // 监听会话更新
  useSessionUpdates((data) => {
    // 更新拓扑数据
    queryClient.setQueryData(['topology'], (oldData: any) => {
      if (!oldData) return oldData

      // 这里可以根据会话更新数据来调整拓扑图
      // 例如更新连接状态、延迟等信息
      return oldData
    })
  })

  return useTopologyData()
}
