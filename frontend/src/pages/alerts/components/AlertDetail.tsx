import React from 'react'
import { Modal, Descriptions, Tag, Typography, Space, Alert as AntAlert } from 'antd'
import type { Alert, Severity, AlertStatus } from '@/types/api'
import dayjs from 'dayjs'

const { Title, Text } = Typography

interface AlertDetailProps {
  alert: Alert | null
  open: boolean
  onClose: () => void
}

const severityColorMap: Record<Severity, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'gold',
  low: 'blue',
}

const statusColorMap: Record<AlertStatus, string> = {
  active: 'red',
  acknowledged: 'orange',
  resolved: 'green',
}

const statusTextMap: Record<AlertStatus, string> = {
  active: '活跃',
  acknowledged: '已确认',
  resolved: '已解决',
}

const AlertDetail: React.FC<AlertDetailProps> = ({ alert, open, onClose }) => {
  if (!alert) return null

  return (
    <Modal
      title={
        <Space>
          <Title level={4} style={{ margin: 0 }}>
            告警详情
          </Title>
          <Tag color={severityColorMap[alert.severity]}>
            {alert.severity.toUpperCase()}
          </Tag>
        </Space>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={800}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {/* 基本信息 */}
        <AntAlert
          message={alert.title}
          description={alert.message}
          type={
            alert.severity === 'critical' || alert.severity === 'high'
              ? 'error'
              : alert.severity === 'medium'
              ? 'warning'
              : 'info'
          }
          showIcon
        />

        {/* 详细信息 */}
        <Descriptions bordered column={2} size="small">
          <Descriptions.Item label="告警ID" span={2}>
            <Text copyable>{alert.id}</Text>
          </Descriptions.Item>

          <Descriptions.Item label="告警类型">
            <Tag>{alert.alert_type.replace(/_/g, ' ').toUpperCase()}</Tag>
          </Descriptions.Item>

          <Descriptions.Item label="状态">
            <Tag color={statusColorMap[alert.status]}>
              {statusTextMap[alert.status]}
            </Tag>
          </Descriptions.Item>

          <Descriptions.Item label="严重程度">
            <Tag color={severityColorMap[alert.severity]}>
              {alert.severity.toUpperCase()}
            </Tag>
          </Descriptions.Item>

          <Descriptions.Item label="设备ID">
            {alert.device_id ? (
              <Text copyable>{alert.device_id}</Text>
            ) : (
              <Text type="secondary">无关联设备</Text>
            )}
          </Descriptions.Item>

          <Descriptions.Item label="创建时间" span={2}>
            {dayjs(alert.created_at).format('YYYY-MM-DD HH:mm:ss')}
            <Text type="secondary" style={{ marginLeft: 8 }}>
              ({dayjs(alert.created_at).fromNow()})
            </Text>
          </Descriptions.Item>

          {alert.acknowledged_at && (
            <>
              <Descriptions.Item label="确认时间">
                {dayjs(alert.acknowledged_at).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
              <Descriptions.Item label="确认人">
                {alert.acknowledged_by || '-'}
              </Descriptions.Item>
            </>
          )}

          {alert.resolved_at && (
            <Descriptions.Item label="解决时间" span={2}>
              {dayjs(alert.resolved_at).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
          )}
        </Descriptions>

        {/* 元数据 */}
        {alert.metadata && Object.keys(alert.metadata).length > 0 && (
          <div>
            <Title level={5}>元数据</Title>
            <Descriptions bordered column={1} size="small">
              {Object.entries(alert.metadata).map(([key, value]) => (
                <Descriptions.Item key={key} label={key}>
                  <Text code style={{ wordBreak: 'break-all' }}>
                    {typeof value === 'object'
                      ? JSON.stringify(value, null, 2)
                      : String(value)}
                  </Text>
                </Descriptions.Item>
              ))}
            </Descriptions>
          </div>
        )}
      </Space>
    </Modal>
  )
}

export default AlertDetail
