/**
 * 告警管理组件使用示例
 *
 * 本文件演示了如何在其他页面中集成告警相关组件
 */

import React, { useState } from 'react'
import { Button, Badge } from 'antd'
import { BellOutlined } from '@ant-design/icons'
import { useAlerts } from '@/hooks/useApi'
import { AlertDetail, AlertStats } from '@/pages/alerts/components'
import type { Alert } from '@/types/api'

/**
 * 示例1: 在Dashboard中显示活跃告警徽章
 */
export const DashboardAlertBadge: React.FC = () => {
  const { data } = useAlerts({ status: 'active', limit: 100 })
  const activeCount = data?.data.length || 0

  return (
    <Badge count={activeCount} overflowCount={99}>
      <Button icon={<BellOutlined />} type="text">
        告警
      </Button>
    </Badge>
  )
}

/**
 * 示例2: 在设备详情页显示设备相关告警统计
 */
interface DeviceAlertsProps {
  deviceId: string
}

export const DeviceAlertsSummary: React.FC<DeviceAlertsProps> = ({ deviceId }) => {
  const { data, isLoading } = useAlerts({ device_id: deviceId, limit: 100 })

  return (
    <AlertStats alerts={data?.data || []} loading={isLoading} />
  )
}

/**
 * 示例3: 快速查看最近的Critical告警
 */
export const CriticalAlertsWidget: React.FC = () => {
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null)
  const [detailVisible, setDetailVisible] = useState(false)

  const { data, isLoading } = useAlerts({
    severity: 'critical',
    status: 'active',
    limit: 5,
  })

  const handleViewDetail = (alert: Alert) => {
    setSelectedAlert(alert)
    setDetailVisible(true)
  }

  return (
    <>
      <div>
        <h3>Critical告警 ({data?.data.length || 0})</h3>
        {isLoading ? (
          <p>加载中...</p>
        ) : data?.data.length === 0 ? (
          <p>暂无Critical告警</p>
        ) : (
          <ul>
            {data?.data.map((alert) => (
              <li key={alert.id}>
                <Button
                  type="link"
                  onClick={() => handleViewDetail(alert)}
                >
                  {alert.title}
                </Button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <AlertDetail
        alert={selectedAlert}
        open={detailVisible}
        onClose={() => {
          setDetailVisible(false)
          setSelectedAlert(null)
        }}
      />
    </>
  )
}

/**
 * 示例4: 自定义告警表格(简化版)
 */
export const SimpleAlertTable: React.FC = () => {
  const { data, isLoading } = useAlerts({ limit: 10 })

  if (isLoading) return <div>加载中...</div>

  return (
    <table>
      <thead>
        <tr>
          <th>严重程度</th>
          <th>标题</th>
          <th>状态</th>
          <th>创建时间</th>
        </tr>
      </thead>
      <tbody>
        {data?.data.map((alert) => (
          <tr key={alert.id}>
            <td>{alert.severity}</td>
            <td>{alert.title}</td>
            <td>{alert.status}</td>
            <td>{new Date(alert.created_at).toLocaleString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

/**
 * 示例5: 使用音频通知
 */
import {
  alertNotification,
  setAlertNotificationEnabled,
} from '@/utils/alertNotification'
import { Switch } from 'antd'

export const AlertSoundToggle: React.FC = () => {
  const [enabled, setEnabled] = React.useState(alertNotification.isEnabled())

  const handleToggle = (checked: boolean) => {
    setEnabled(checked)
    setAlertNotificationEnabled(checked)

    if (checked) {
      // 播放测试音
      alertNotification.test()
    }
  }

  return (
    <div>
      <Switch
        checked={enabled}
        onChange={handleToggle}
        checkedChildren="音频开"
        unCheckedChildren="音频关"
      />
      <span style={{ marginLeft: 8 }}>
        告警音频通知
      </span>
    </div>
  )
}

/**
 * 示例6: 手动触发告警操作
 */
import { useAcknowledgeAlert, useResolveAlert } from '@/hooks/useApi'

export const ManualAlertActions: React.FC<{ alertId: string }> = ({ alertId }) => {
  const acknowledgeAlert = useAcknowledgeAlert()
  const resolveAlert = useResolveAlert()

  const handleAcknowledge = async () => {
    try {
      await acknowledgeAlert.mutateAsync({
        alertId,
        acknowledgedBy: 'current-user-id',
      })
      console.log('告警已确认')
    } catch (error) {
      console.error('确认失败:', error)
    }
  }

  const handleResolve = async () => {
    try {
      await resolveAlert.mutateAsync(alertId)
      console.log('告警已解决')
    } catch (error) {
      console.error('解决失败:', error)
    }
  }

  return (
    <div>
      <Button
        onClick={handleAcknowledge}
        loading={acknowledgeAlert.isPending}
      >
        确认告警
      </Button>
      <Button
        onClick={handleResolve}
        loading={resolveAlert.isPending}
        style={{ marginLeft: 8 }}
      >
        解决告警
      </Button>
    </div>
  )
}

/**
 * 示例7: WebSocket实时监听告警事件
 */
import { useWebSocket } from '@/hooks/useWebSocket'
import { useQueryClient } from '@tanstack/react-query'

export const AlertRealtimeListener: React.FC = () => {
  const queryClient = useQueryClient()

  useWebSocket({
    onMessage: (message) => {
      if (message.type === 'alert_created') {
        console.log('新告警创建:', message.data)

        // 刷新告警列表
        queryClient.invalidateQueries({ queryKey: ['alerts'] })

        // 播放音频通知
        if (message.data?.severity) {
          alertNotification.play(message.data.severity)
        }
      } else if (message.type === 'alert_updated') {
        console.log('告警更新:', message.data)

        // 刷新告警列表
        queryClient.invalidateQueries({ queryKey: ['alerts'] })
      }
    },
  })

  return null // 这是一个纯监听组件,不渲染任何UI
}
