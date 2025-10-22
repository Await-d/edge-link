import React from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Row,
  Col,
  Tag,
  Button,
  Space,
  Popconfirm,
  Alert,
  Spin,
  Typography,
} from 'antd'
import {
  ArrowLeftOutlined,
  ReloadOutlined,
  PoweroffOutlined,
  DeleteOutlined,
} from '@ant-design/icons'
import { useDevice, useDeleteDevice, useRestartDevice } from '@/hooks/useApi'
import {
  DeviceInfo,
  ConnectionStatus,
  QuickActions,
  PeerList,
  MetricsHistory,
} from './components'

const { Title } = Typography

const DeviceDetail: React.FC = () => {
  const { deviceId } = useParams<{ deviceId: string }>()
  const navigate = useNavigate()
  const { data: device, isLoading, error, refetch } = useDevice(deviceId!)
  const deleteDevice = useDeleteDevice()
  const restartDevice = useRestartDevice()

  const handleDelete = async () => {
    try {
      await deleteDevice.mutateAsync(deviceId!)
      navigate('/devices')
    } catch (error) {
      // 错误已在mutation中处理
    }
  }

  const handleRestart = async () => {
    try {
      await restartDevice.mutateAsync(deviceId!)
      // 等待2秒后刷新数据
      setTimeout(() => {
        refetch()
      }, 2000)
    } catch (error) {
      // 错误已在mutation中处理
    }
  }

  const handleBack = () => {
    navigate('/devices')
  }

  if (error) {
    return (
      <div>
        <Button icon={<ArrowLeftOutlined />} onClick={handleBack} style={{ marginBottom: 16 }}>
          返回设备列表
        </Button>
        <Alert
          message="加载设备详情失败"
          description="设备不存在或您没有访问权限"
          type="error"
          showIcon
        />
      </div>
    )
  }

  if (isLoading || !device) {
    return (
      <div style={{ textAlign: 'center', padding: '100px 0' }}>
        <Spin size="large" tip="加载设备详情..." />
      </div>
    )
  }

  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 24,
          flexWrap: 'wrap',
          gap: 16,
        }}
      >
        <Space align="center">
          <Button icon={<ArrowLeftOutlined />} onClick={handleBack}>
            返回
          </Button>
          <Title level={2} style={{ margin: 0 }}>
            {device.name}
          </Title>
          <Tag color={device.is_online ? 'success' : 'default'}>
            {device.is_online ? '在线' : '离线'}
          </Tag>
        </Space>

        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>
          <Popconfirm
            title="确定重启此设备?"
            description="将向设备发送重启指令"
            onConfirm={handleRestart}
            okText="确定"
            cancelText="取消"
          >
            <Button
              icon={<PoweroffOutlined />}
              loading={restartDevice.isPending}
              disabled={!device.is_online}
            >
              重启
            </Button>
          </Popconfirm>
          <Popconfirm
            title="确定删除此设备?"
            description="删除后将无法恢复,所有关联数据将被清除"
            onConfirm={handleDelete}
            okText="确定"
            cancelText="取消"
          >
            <Button
              danger
              icon={<DeleteOutlined />}
              loading={deleteDevice.isPending}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      </div>

      <Row gutter={[16, 16]}>
        <Col span={24}>
          <DeviceInfo device={device} loading={isLoading} />
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} md={12}>
          <Card title="连接状态" size="small">
            <ConnectionStatus device={device} loading={isLoading} />
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card title="快速操作" size="small">
            <QuickActions
              device={device}
              loading={isLoading}
              onRefresh={() => refetch()}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={24}>
          <Card title="对等设备列表" size="small">
            <PeerList deviceId={device.id} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={24}>
          <Card title="历史指标" size="small">
            <MetricsHistory deviceId={device.id} />
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default DeviceDetail
