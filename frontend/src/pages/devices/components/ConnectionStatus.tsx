import React from 'react'
import { Descriptions, Tag, Skeleton } from 'antd'
import type { DeviceDetail } from '@/types/api'

interface ConnectionStatusProps {
  device: DeviceDetail
  loading?: boolean
}

const ConnectionStatus: React.FC<ConnectionStatusProps> = ({ device, loading }) => {
  if (loading) {
    return <Skeleton active paragraph={{ rows: 4 }} />
  }

  return (
    <Descriptions column={1} size="small">
      <Descriptions.Item label="连接模式">
        {device.connection_mode ? (
          <Tag color={device.connection_mode === 'direct' ? 'blue' : 'orange'}>
            {device.connection_mode === 'direct' ? '直连' : '中继'}
          </Tag>
        ) : (
          <Tag color="default">未知</Tag>
        )}
      </Descriptions.Item>
      <Descriptions.Item label="NAT类型">
        {device.nat_type || 'Unknown'}
      </Descriptions.Item>
      <Descriptions.Item label="公网IP">
        {device.public_ip || 'N/A'}
      </Descriptions.Item>
      <Descriptions.Item label="监听端口">
        {device.listen_port || '-'}
      </Descriptions.Item>
      <Descriptions.Item label="当前带宽">
        上行: {device.current_upload?.toFixed(2) || '0.00'} MB/s | 下行:{' '}
        {device.current_download?.toFixed(2) || '0.00'} MB/s
      </Descriptions.Item>
    </Descriptions>
  )
}

export default ConnectionStatus
