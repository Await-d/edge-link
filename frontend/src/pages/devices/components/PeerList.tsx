import React from 'react'
import { Table, Tag, Alert } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useDevicePeers } from '@/hooks/useApi'
import type { DevicePeer } from '@/types/api'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

interface PeerListProps {
  deviceId: string
}

const PeerList: React.FC<PeerListProps> = ({ deviceId }) => {
  const { data, isLoading, error } = useDevicePeers(deviceId)

  const columns: ColumnsType<DevicePeer> = [
    {
      title: '对等设备',
      dataIndex: 'peer_name',
      key: 'peer_name',
    },
    {
      title: '对等IP',
      dataIndex: 'peer_ip',
      key: 'peer_ip',
    },
    {
      title: '连接状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'connected' ? 'success' : 'default'}>
          {status === 'connected' ? '已连接' : '未连接'}
        </Tag>
      ),
    },
    {
      title: '延迟',
      dataIndex: 'latency',
      key: 'latency',
      render: (ms?: number) => (ms ? `${ms}ms` : '-'),
    },
    {
      title: '丢包率',
      dataIndex: 'packet_loss',
      key: 'packet_loss',
      render: (rate?: number) => (rate !== undefined ? `${rate.toFixed(2)}%` : '-'),
    },
    {
      title: '最后握手',
      dataIndex: 'last_handshake',
      key: 'last_handshake',
      render: (time?: string) => (time ? dayjs(time).fromNow() : '-'),
    },
    {
      title: '发送/接收',
      key: 'traffic',
      render: (_, record) => {
        const formatBytes = (bytes: number) => {
          if (bytes < 1024) return `${bytes} B`
          if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
          if (bytes < 1024 * 1024 * 1024)
            return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
          return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
        }
        return `${formatBytes(record.bytes_sent)} / ${formatBytes(record.bytes_received)}`
      },
    },
  ]

  if (error) {
    return <Alert message="加载对等设备列表失败" type="error" showIcon />
  }

  return (
    <Table
      columns={columns}
      dataSource={data?.peers || []}
      rowKey="id"
      loading={isLoading}
      pagination={false}
      locale={{
        emptyText: '暂无对等设备',
      }}
    />
  )
}

export default PeerList
