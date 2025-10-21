import React from 'react'
import { Card, Typography, Empty } from 'antd'

const { Title } = Typography

const Topology: React.FC = () => {
  return (
    <div>
      <Title level={2}>网络拓扑</Title>
      <Card style={{ marginTop: 24 }}>
        <Empty description="网络拓扑图功能开发中..." />
        <p style={{ textAlign: 'center', marginTop: 16, color: '#999' }}>
          将使用ECharts展示设备间的连接关系和网络拓扑结构
        </p>
      </Card>
    </div>
  )
}

export default Topology
