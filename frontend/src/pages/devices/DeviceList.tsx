import React, { useState } from 'react'
import {
  Table,
  Card,
  Typography,
  Tag,
  Space,
  Button,
  Popconfirm,
  Input,
  Select,
} from 'antd'
import { DeleteOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useDevices, useDeleteDevice } from '@/hooks/useApi'
import type { Device } from '@/types/api'
import dayjs from 'dayjs'

const { Title } = Typography
const { Search } = Input

const DeviceList: React.FC = () => {
  const [searchText, setSearchText] = useState('')
  const [platformFilter, setPlatformFilter] = useState<string>()
  const [onlineFilter, setOnlineFilter] = useState<boolean>()

  const { data, isLoading, refetch } = useDevices({
    platform: platformFilter,
    online: onlineFilter,
  })

  const deleteDevice = useDeleteDevice()

  const handleDelete = async (deviceId: string) => {
    try {
      await deleteDevice.mutateAsync(deviceId)
    } catch (error) {
      // 错误已在mutation中处理
    }
  }

  const columns: ColumnsType<Device> = [
    {
      title: '设备名称',
      dataIndex: 'name',
      key: 'name',
      filteredValue: searchText ? [searchText] : null,
      onFilter: (value, record) =>
        record.name.toLowerCase().includes((value as string).toLowerCase()),
    },
    {
      title: '平台',
      dataIndex: 'platform',
      key: 'platform',
      render: (platform: string) => {
        const colorMap: Record<string, string> = {
          linux: 'blue',
          windows: 'cyan',
          macos: 'purple',
          android: 'green',
          ios: 'orange',
        }
        return <Tag color={colorMap[platform] || 'default'}>{platform.toUpperCase()}</Tag>
      },
    },
    {
      title: '虚拟IP',
      dataIndex: 'virtual_ip',
      key: 'virtual_ip',
    },
    {
      title: '状态',
      dataIndex: 'is_online',
      key: 'is_online',
      render: (isOnline: boolean) => (
        <Tag color={isOnline ? 'success' : 'default'}>
          {isOnline ? '在线' : '离线'}
        </Tag>
      ),
    },
    {
      title: '最后上线',
      dataIndex: 'last_seen_at',
      key: 'last_seen_at',
      render: (time: string) => (time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <Popconfirm
            title="确定删除此设备?"
            description="删除后将无法恢复"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              type="link"
              danger
              icon={<DeleteOutlined />}
              loading={deleteDevice.isPending}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Title level={2}>设备管理</Title>

      <Card style={{ marginTop: 24 }}>
        <Space style={{ marginBottom: 16 }} size="middle">
          <Search
            placeholder="搜索设备名称"
            allowClear
            style={{ width: 250 }}
            onSearch={setSearchText}
            onChange={(e) => !e.target.value && setSearchText('')}
            prefix={<SearchOutlined />}
          />
          <Select
            placeholder="平台过滤"
            allowClear
            style={{ width: 150 }}
            onChange={setPlatformFilter}
            options={[
              { value: 'linux', label: 'Linux' },
              { value: 'windows', label: 'Windows' },
              { value: 'macos', label: 'macOS' },
              { value: 'android', label: 'Android' },
              { value: 'ios', label: 'iOS' },
            ]}
          />
          <Select
            placeholder="在线状态"
            allowClear
            style={{ width: 150 }}
            onChange={setOnlineFilter}
            options={[
              { value: true, label: '在线' },
              { value: false, label: '离线' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>
        </Space>

        <Table
          columns={columns}
          dataSource={data?.data || []}
          rowKey="id"
          loading={isLoading}
          pagination={{
            total: data?.total || 0,
            pageSize: data?.limit || 50,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>
    </div>
  )
}

export default DeviceList
