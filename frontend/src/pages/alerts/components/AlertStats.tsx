import React, { useMemo } from 'react'
import { Card, Row, Col, Statistic, Badge, Space } from 'antd'
import {
  ExclamationCircleOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  FireOutlined,
} from '@ant-design/icons'
import type { Alert, Severity } from '@/types/api'

interface AlertStatsProps {
  alerts: Alert[]
  loading?: boolean
}

const AlertStats: React.FC<AlertStatsProps> = ({ alerts, loading = false }) => {
  // 计算统计数据
  const stats = useMemo(() => {
    const activeCount = alerts.filter((a) => a.status === 'active').length
    const acknowledgedCount = alerts.filter((a) => a.status === 'acknowledged').length
    const resolvedCount = alerts.filter((a) => a.status === 'resolved').length

    const severityCounts: Record<Severity, number> = {
      critical: 0,
      high: 0,
      medium: 0,
      low: 0,
    }

    alerts.forEach((alert) => {
      if (alert.status === 'active') {
        severityCounts[alert.severity]++
      }
    })

    return {
      total: alerts.length,
      active: activeCount,
      acknowledged: acknowledgedCount,
      resolved: resolvedCount,
      critical: severityCounts.critical,
      high: severityCounts.high,
      medium: severityCounts.medium,
      low: severityCounts.low,
    }
  }, [alerts])

  return (
    <Row gutter={[16, 16]}>
      {/* 活跃告警 */}
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title={
              <Space>
                <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
                活跃告警
              </Space>
            }
            value={stats.active}
            valueStyle={{ color: stats.active > 0 ? '#ff4d4f' : undefined }}
            loading={loading}
          />
        </Card>
      </Col>

      {/* 已确认告警 */}
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title={
              <Space>
                <ClockCircleOutlined style={{ color: '#faad14' }} />
                已确认
              </Space>
            }
            value={stats.acknowledged}
            valueStyle={{ color: stats.acknowledged > 0 ? '#faad14' : undefined }}
            loading={loading}
          />
        </Card>
      </Col>

      {/* 已解决告警 */}
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title={
              <Space>
                <CheckCircleOutlined style={{ color: '#52c41a' }} />
                已解决
              </Space>
            }
            value={stats.resolved}
            valueStyle={{ color: '#52c41a' }}
            loading={loading}
          />
        </Card>
      </Col>

      {/* 总告警数 */}
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic
            title="总告警数"
            value={stats.total}
            loading={loading}
          />
        </Card>
      </Col>

      {/* 严重程度统计 */}
      <Col span={24}>
        <Card
          title={
            <Space>
              <FireOutlined />
              按严重程度分类（仅活跃告警）
            </Space>
          }
        >
          <Row gutter={16}>
            <Col xs={12} sm={6}>
              <Statistic
                title={<Badge status="error" text="Critical" />}
                value={stats.critical}
                valueStyle={{ color: stats.critical > 0 ? '#f5222d' : undefined }}
                loading={loading}
              />
            </Col>
            <Col xs={12} sm={6}>
              <Statistic
                title={<Badge status="warning" text="High" />}
                value={stats.high}
                valueStyle={{ color: stats.high > 0 ? '#fa8c16' : undefined }}
                loading={loading}
              />
            </Col>
            <Col xs={12} sm={6}>
              <Statistic
                title={<Badge status="processing" text="Medium" />}
                value={stats.medium}
                valueStyle={{ color: stats.medium > 0 ? '#faad14' : undefined }}
                loading={loading}
              />
            </Col>
            <Col xs={12} sm={6}>
              <Statistic
                title={<Badge status="default" text="Low" />}
                value={stats.low}
                loading={loading}
              />
            </Col>
          </Row>
        </Card>
      </Col>
    </Row>
  )
}

export default AlertStats
