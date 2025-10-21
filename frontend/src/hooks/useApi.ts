import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { deviceApi, networkApi, alertApi, auditApi } from '@/services/api'
import { message } from 'antd'

// 设备相关hooks
export const useDevices = (params?: Parameters<typeof deviceApi.getDevices>[0]) => {
  return useQuery({
    queryKey: ['devices', params],
    queryFn: () => deviceApi.getDevices(params),
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
