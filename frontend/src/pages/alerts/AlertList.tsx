import React, { useState, useEffect, useRef } from 'react'
import {
  Table,
  Card,
  Typography,
  Tag,
  Space,
  Button,
  Select,
  Popconfirm,
  Input,
  Badge,
  Switch,
  Tooltip,
} from 'antd'
import {
  CheckOutlined,
  ReloadOutlined,
  SearchOutlined,
  EyeOutlined,
  CloseCircleOutlined,
  BellOutlined,
  BellFilled,
} from '@ant-design/icons'
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table'
import { useAlerts, useAcknowledgeAlert, useResolveAlert } from '@/hooks/useApi'
import { useWebSocket } from '@/hooks/useWebSocket'
import type { Alert, Severity, AlertStatus, AlertType } from '@/types/api'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import AlertDetail from './components/AlertDetail'
import AlertStats from './components/AlertStats'
import {
  alertNotification,
  initAlertNotification,
  setAlertNotificationEnabled,
} from '@/utils/alertNotification'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

const { Title } = Typography
const { Search } = Input

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

const AlertList: React.FC = () => {
  // 筛选状态
  const [severityFilter, setSeverityFilter] = useState<Severity>()
  const [statusFilter, setStatusFilter] = useState<AlertStatus>()
  const [typeFilter, setTypeFilter] = useState<AlertType>()

  // 分页状态
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
  })

  // 详情弹窗状态
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null)
  const [detailVisible, setDetailVisible] = useState(false)

  // 音频通知状态
  const [soundEnabled, setSoundEnabled] = useState(alertNotification.isEnabled())

  // 记录已知的告警ID，用于检测新告警
  const alertIdsRef = useRef<Set<string>>(new Set())

  // 初始化音频通知
  useEffect(() => {
    initAlertNotification()
    setSoundEnabled(alertNotification.isEnabled())
  }, [])

  // 查询告警列表
  const { data, isLoading, refetch } = useAlerts({
    severity: severityFilter,
    status: statusFilter,
    type: typeFilter,
    limit: pagination.pageSize,
    offset: (pagination.current - 1) * pagination.pageSize,
  })

  // Mutations
  const acknowledgeAlert = useAcknowledgeAlert()
  const resolveAlert = useResolveAlert()

  // WebSocket实时更新
  useWebSocket({
    onMessage: (message) => {
      if (message.type === 'alert_created' || message.type === 'alert_updated') {
        // 刷新告警列表
        refetch()
      }
    },
  })

  // 检测新告警并播放声音
  useEffect(() => {
    if (!data?.data) return

    const currentAlertIds = new Set(data.data.map((alert) => alert.id))
    const newAlerts = data.data.filter(
      (alert) => !alertIdsRef.current.has(alert.id) && alert.status === 'active'
    )

    // 如果有新的活跃告警，播放声音
    if (newAlerts.length > 0 && alertIdsRef.current.size > 0) {
      // 找出最高严重级别的新告警
      const criticalAlert = newAlerts.find((a) => a.severity === 'critical')
      const highAlert = newAlerts.find((a) => a.severity === 'high')
      const alertToNotify = criticalAlert || highAlert || newAlerts[0]

      if (alertToNotify) {
        alertNotification.play(alertToNotify.severity)
      }
    }

    alertIdsRef.current = currentAlertIds
  }, [data])

  // 处理确认
  const handleAcknowledge = async (alertId: string) => {
    try {
      // TODO: 从认证上下文获取当前用户ID
      const userId = '00000000-0000-0000-0000-000000000000'
      await acknowledgeAlert.mutateAsync({ alertId, acknowledgedBy: userId })
    } catch (error) {
      // 错误已在mutation中处理
    }
  }

  // 处理解决
  const handleResolve = async (alertId: string) => {
    try {
      await resolveAlert.mutateAsync(alertId)
    } catch (error) {
      // 错误已在mutation中处理
    }
  }

  // 查看详情
  const handleViewDetail = (alert: Alert) => {
    setSelectedAlert(alert)
    setDetailVisible(true)
  }

  // 切换音频通知
  const handleSoundToggle = (checked: boolean) => {
    setSoundEnabled(checked)
    setAlertNotificationEnabled(checked)

    if (checked) {
      // 测试播放
      alertNotification.test()
    }
  }

  // 处理表格变化
  const handleTableChange = (newPagination: TablePaginationConfig) => {
    setPagination({
      current: newPagination.current || 1,
      pageSize: newPagination.pageSize || 20,
    })
  }

  // 表格列定义
  const columns: ColumnsType<Alert> = [
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: Severity) => (
        <Tag color={severityColorMap[severity]}>{severity.toUpperCase()}</Tag>
      ),
      sorter: (a, b) => {
        const order = { critical: 4, high: 3, medium: 2, low: 1 }
        return order[a.severity] - order[b.severity]
      },
    },
    {
      title: '类型',
      dataIndex: 'alert_type',
      key: 'alert_type',
      width: 150,
      render: (type: string) => (
        <Tag>{type.replace(/_/g, ' ').toUpperCase()}</Tag>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: AlertStatus) => (
        <Tag color={statusColorMap[status]}>{statusTextMap[status]}</Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => (
        <Tooltip title={dayjs(time).format('YYYY-MM-DD HH:mm:ss')}>
          {dayjs(time).fromNow()}
        </Tooltip>
      ),
      sorter: (a, b) => dayjs(a.created_at).unix() - dayjs(b.created_at).unix(),
      defaultSortOrder: 'descend',
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetail(record)}
            size="small"
          >
            详情
          </Button>
          {record.status === 'active' && (
            <>
              <Popconfirm
                title="确认此告警?"
                onConfirm={() => handleAcknowledge(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Button
                  type="link"
                  icon={<CheckOutlined />}
                  loading={acknowledgeAlert.isPending}
                  size="small"
                >
                  确认
                </Button>
              </Popconfirm>
            </>
          )}
          {record.status === 'acknowledged' && (
            <Popconfirm
              title="解决此告警?"
              onConfirm={() => handleResolve(record.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="link"
                icon={<CloseCircleOutlined />}
                loading={resolveAlert.isPending}
                size="small"
              >
                解决
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  // 活跃告警数量
  const activeAlertCount = data?.data.filter((a) => a.status === 'active').length || 0

  return (
    <div>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>
          <Space>
            <Badge count={activeAlertCount} overflowCount={99}>
              告警中心
            </Badge>
          </Space>
        </Title>
        <Space>
          <Tooltip title={soundEnabled ? '关闭告警声音' : '开启告警声音'}>
            <Switch
              checked={soundEnabled}
              onChange={handleSoundToggle}
              checkedChildren={<BellFilled />}
              unCheckedChildren={<BellOutlined />}
            />
          </Tooltip>
        </Space>
      </Space>

      {/* 统计卡片 */}
      <AlertStats alerts={data?.data || []} loading={isLoading} />

      {/* 筛选和搜索 */}
      <Card style={{ marginTop: 24 }}>
        <Space style={{ marginBottom: 16 }} size="middle" wrap>
          <Select
            placeholder="严重程度"
            allowClear
            style={{ width: 150 }}
            value={severityFilter}
            onChange={setSeverityFilter}
            options={[
              { value: 'critical', label: 'Critical' },
              { value: 'high', label: 'High' },
              { value: 'medium', label: 'Medium' },
              { value: 'low', label: 'Low' },
            ]}
          />
          <Select
            placeholder="状态"
            allowClear
            style={{ width: 150 }}
            value={statusFilter}
            onChange={setStatusFilter}
            options={[
              { value: 'active', label: '活跃' },
              { value: 'acknowledged', label: '已确认' },
              { value: 'resolved', label: '已解决' },
            ]}
          />
          <Select
            placeholder="类型"
            allowClear
            style={{ width: 180 }}
            value={typeFilter}
            onChange={setTypeFilter}
            options={[
              { value: 'device_offline', label: 'Device Offline' },
              { value: 'high_latency', label: 'High Latency' },
              { value: 'connection_failed', label: 'Connection Failed' },
              { value: 'key_expired', label: 'Key Expired' },
              { value: 'system', label: 'System' },
            ]}
          />
          <Search
            placeholder="搜索告警标题或消息"
            allowClear
            style={{ width: 300 }}
            enterButton={<SearchOutlined />}
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
          onChange={handleTableChange}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: data?.total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            pageSizeOptions: [10, 20, 50, 100],
          }}
          scroll={{ x: 1200 }}
          rowClassName={(record) => {
            if (record.status === 'active' && record.severity === 'critical') {
              return 'alert-row-critical'
            }
            return ''
          }}
        />
      </Card>

      {/* 告警详情弹窗 */}
      <AlertDetail
        alert={selectedAlert}
        open={detailVisible}
        onClose={() => {
          setDetailVisible(false)
          setSelectedAlert(null)
        }}
      />

      <style>{`
        .alert-row-critical {
          background-color: #fff1f0 !important;
        }
        .alert-row-critical:hover td {
          background-color: #ffccc7 !important;
        }
      `}</style>
    </div>
  )
}

export default AlertList
