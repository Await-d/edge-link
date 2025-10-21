import React, { useState } from 'react'
import {
  Table,
  Card,
  Typography,
  Tag,
  Space,
  Button,
  Select,
  DatePicker,
} from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useAuditLogs } from '@/hooks/useApi'
import type { AuditLog, ResourceType } from '@/types/api'
import dayjs from 'dayjs'

const { Title } = Typography
const { RangePicker } = DatePicker

const resourceTypeMap: Record<ResourceType, string> = {
  device: '设备',
  virtual_network: '虚拟网络',
  pre_shared_key: '预共享密钥',
  alert: '告警',
  organization: '组织',
}

const AuditLogList: React.FC = () => {
  const [actionFilter, setActionFilter] = useState<string>()
  const [resourceTypeFilter, setResourceTypeFilter] = useState<ResourceType>()
  const [dateRange, setDateRange] = useState<[string, string] | undefined>()

  const { data, isLoading, refetch } = useAuditLogs({
    action: actionFilter,
    resource_type: resourceTypeFilter,
    start_time: dateRange?.[0],
    end_time: dateRange?.[1],
  })

  const columns: ColumnsType<AuditLog> = [
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      width: 100,
      render: (action: string) => {
        const colorMap: Record<string, string> = {
          create: 'green',
          update: 'blue',
          delete: 'red',
          acknowledge: 'orange',
        }
        return <Tag color={colorMap[action] || 'default'}>{action.toUpperCase()}</Tag>
      },
    },
    {
      title: '资源类型',
      dataIndex: 'resource_type',
      key: 'resource_type',
      width: 120,
      render: (type: ResourceType) => resourceTypeMap[type] || type,
    },
    {
      title: '资源ID',
      dataIndex: 'resource_id',
      key: 'resource_id',
      width: 280,
      ellipsis: true,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 140,
    },
    {
      title: 'User-Agent',
      dataIndex: 'user_agent',
      key: 'user_agent',
      ellipsis: true,
    },
  ]

  return (
    <div>
      <Title level={2}>审计日志</Title>

      <Card style={{ marginTop: 24 }}>
        <Space style={{ marginBottom: 16 }} size="middle" wrap>
          <Select
            placeholder="操作类型"
            allowClear
            style={{ width: 150 }}
            onChange={setActionFilter}
            options={[
              { value: 'create', label: '创建' },
              { value: 'update', label: '更新' },
              { value: 'delete', label: '删除' },
              { value: 'acknowledge', label: '确认' },
            ]}
          />
          <Select
            placeholder="资源类型"
            allowClear
            style={{ width: 150 }}
            onChange={setResourceTypeFilter}
            options={[
              { value: 'device', label: '设备' },
              { value: 'virtual_network', label: '虚拟网络' },
              { value: 'alert', label: '告警' },
              { value: 'organization', label: '组织' },
            ]}
          />
          <RangePicker
            showTime
            format="YYYY-MM-DD HH:mm:ss"
            onChange={(dates) => {
              if (dates && dates[0] && dates[1]) {
                setDateRange([
                  dates[0].toISOString(),
                  dates[1].toISOString(),
                ])
              } else {
                setDateRange(undefined)
              }
            }}
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
          expandable={{
            expandedRowRender: (record) => (
              <div>
                <p>
                  <strong>操作前状态:</strong>
                </p>
                <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
                  {record.before_state
                    ? JSON.stringify(record.before_state, null, 2)
                    : '无'}
                </pre>
                <p style={{ marginTop: 16 }}>
                  <strong>操作后状态:</strong>
                </p>
                <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
                  {record.after_state
                    ? JSON.stringify(record.after_state, null, 2)
                    : '无'}
                </pre>
              </div>
            ),
          }}
        />
      </Card>
    </div>
  )
}

export default AuditLogList
