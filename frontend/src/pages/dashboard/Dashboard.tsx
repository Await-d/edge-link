import React from 'react'
import { Card, Row, Col, Statistic, Typography } from 'antd'
import {
  CloudServerOutlined,
  CheckCircleOutlined,
  LinkOutlined,
  AlertOutlined,
} from '@ant-design/icons'
import { useDashboardStats } from '@/hooks/useApi'
import MetricsChart from './components/MetricsChart'
import TrafficChart from './components/TrafficChart'
import DeviceDistribution from './components/DeviceDistribution'
import AlertTrend from './components/AlertTrend'

const { Title } = Typography

/**
 * Dashboard 仪表盘页面
 * 展示系统核心指标和监控数据
 */
const Dashboard: React.FC = () => {
  const { data: stats, isLoading } = useDashboardStats()

  return (
    <div>
      <Title level={2}>仪表盘</Title>

      {/* 统计卡片行 */}
      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="设备总数"
              value={stats?.total_devices ?? 0}
              prefix={<CloudServerOutlined />}
              loading={isLoading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="在线设备"
              value={stats?.online_devices ?? 0}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#3f8600' }}
              loading={isLoading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃会话"
              value={stats?.active_sessions ?? 0}
              prefix={<LinkOutlined />}
              loading={isLoading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃告警"
              value={stats?.active_alerts ?? 0}
              prefix={<AlertOutlined />}
              valueStyle={{
                color: (stats?.active_alerts ?? 0) > 0 ? '#cf1322' : undefined,
              }}
              loading={isLoading}
            />
          </Card>
        </Col>
      </Row>

      {/* 图表行1: 设备趋势 + 设备分布 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card
            title="设备趋势"
            extra={<span style={{ fontSize: 12, color: '#8c8c8c' }}>过去24小时</span>}
          >
            <MetricsChart />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="设备分布">
            <DeviceDistribution />
          </Card>
        </Col>
      </Row>

      {/* 图表行2: 网络流量 + 告警趋势 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card
            title="网络流量"
            extra={<span style={{ fontSize: 12, color: '#8c8c8c' }}>过去24小时</span>}
          >
            <TrafficChart />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title="告警趋势"
            extra={<span style={{ fontSize: 12, color: '#8c8c8c' }}>过去7天</span>}
          >
            <AlertTrend />
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
