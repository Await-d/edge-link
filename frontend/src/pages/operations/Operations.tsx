import React from 'react'
import { Card, Typography, Empty } from 'antd'

const { Title } = Typography

const Operations: React.FC = () => {
  return (
    <div>
      <Title level={2}>运维工具</Title>
      <Card style={{ marginTop: 24 }}>
        <Empty description="运维工具功能开发中..." />
        <p style={{ textAlign: 'center', marginTop: 16, color: '#999' }}>
          将提供诊断包下载、设备健康检查、密钥轮换等运维功能
        </p>
      </Card>
    </div>
  )
}

export default Operations
