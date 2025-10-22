import React from 'react'
import { Space, Button, Skeleton } from 'antd'
import {
  SyncOutlined,
  ApiOutlined,
  BarChartOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import type { DeviceDetail } from '@/types/api'

interface QuickActionsProps {
  device: DeviceDetail
  loading?: boolean
  onRefresh?: () => void
}

const QuickActions: React.FC<QuickActionsProps> = ({
  device,
  loading,
  onRefresh,
}) => {
  if (loading) {
    return <Skeleton active paragraph={{ rows: 2 }} />
  }

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Button
        type="default"
        icon={<SyncOutlined />}
        block
        onClick={onRefresh}
      >
        同步配置
      </Button>
      <Button
        type="default"
        icon={<ApiOutlined />}
        block
        disabled={!device.is_online}
      >
        测试连通性
      </Button>
      <Button type="default" icon={<BarChartOutlined />} block>
        查看完整指标
      </Button>
      <Button type="default" icon={<SettingOutlined />} block>
        高级设置
      </Button>
    </Space>
  )
}

export default QuickActions
