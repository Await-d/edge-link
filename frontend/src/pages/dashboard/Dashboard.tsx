import React from 'react'
import { Card, Row, Col, Statistic, Typography } from 'antd'
import {
  CloudServerOutlined,
  CheckCircleOutlined,
  LinkOutlined,
  AlertOutlined,
} from '@ant-design/icons'

const { Title } = Typography

const Dashboard: React.FC = () => {
  // TODO: 从API获取实际数据
  const stats = {
    totalDevices: 0,
    onlineDevices: 0,
    activeSessions: 0,
    activeAlerts: 0,
  }

  return (
    <div>
      <Title level={2}>仪表盘</Title>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="设备总数"
              value={stats.totalDevices}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="在线设备"
              value={stats.onlineDevices}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃会话"
              value={stats.activeSessions}
              prefix={<LinkOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃告警"
              value={stats.activeAlerts}
              prefix={<AlertOutlined />}
              valueStyle={{ color: stats.activeAlerts > 0 ? '#cf1322' : undefined }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={24}>
          <Card title="系统概览">
            <p>EdgeLink端到端直连系统运行正常</p>
            <p>数据加载中...</p>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
