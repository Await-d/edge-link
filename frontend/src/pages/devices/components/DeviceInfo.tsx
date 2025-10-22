import React from 'react'
import { Card, Descriptions, Tag, Typography } from 'antd'
import type { DeviceDetail } from '@/types/api'
import dayjs from 'dayjs'

interface DeviceInfoProps {
  device: DeviceDetail
  loading?: boolean
}

const DeviceInfo: React.FC<DeviceInfoProps> = ({ device, loading }) => {
  const platformColorMap: Record<string, string> = {
    linux: 'blue',
    windows: 'cyan',
    macos: 'purple',
    android: 'green',
    ios: 'orange',
  }

  return (
    <Card loading={loading}>
      <Descriptions bordered column={{ xs: 1, sm: 1, md: 2 }}>
        <Descriptions.Item label="设备ID">{device.id}</Descriptions.Item>
        <Descriptions.Item label="设备名称">{device.name}</Descriptions.Item>
        <Descriptions.Item label="虚拟IP">{device.virtual_ip}</Descriptions.Item>
        <Descriptions.Item label="平台">
          <Tag color={platformColorMap[device.platform] || 'default'}>
            {device.platform.toUpperCase()}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label="公钥" span={2}>
          <Typography.Text copyable code style={{ fontSize: 12 }}>
            {device.public_key}
          </Typography.Text>
        </Descriptions.Item>
        <Descriptions.Item label="所属虚拟网络">
          {device.virtual_network_name || device.virtual_network_id}
        </Descriptions.Item>
        <Descriptions.Item label="外部端点">
          {device.endpoint || 'N/A'}
        </Descriptions.Item>
        <Descriptions.Item label="创建时间">
          {dayjs(device.created_at).format('YYYY-MM-DD HH:mm:ss')}
        </Descriptions.Item>
        <Descriptions.Item label="最后上线">
          {device.last_seen_at
            ? dayjs(device.last_seen_at).format('YYYY-MM-DD HH:mm:ss')
            : '-'}
        </Descriptions.Item>
        {device.tags && device.tags.length > 0 && (
          <Descriptions.Item label="标签" span={2}>
            {device.tags.map((tag) => (
              <Tag key={tag}>{tag}</Tag>
            ))}
          </Descriptions.Item>
        )}
      </Descriptions>
    </Card>
  )
}

export default DeviceInfo
