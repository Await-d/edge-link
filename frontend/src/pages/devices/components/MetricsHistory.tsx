import React, { useState } from 'react'
import { Alert, Select, Space, Spin } from 'antd'
import ReactECharts from 'echarts-for-react'
import { useDeviceMetrics } from '@/hooks/useApi'
import type { EChartsOption } from 'echarts'

interface MetricsHistoryProps {
  deviceId: string
}

const MetricsHistory: React.FC<MetricsHistoryProps> = ({ deviceId }) => {
  const [timeRange, setTimeRange] = useState('24h')
  const { data, isLoading, error } = useDeviceMetrics(deviceId, timeRange)

  const getTrafficChartOption = (): EChartsOption => {
    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
        },
      },
      legend: {
        data: ['入站流量', '出站流量'],
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: data?.timestamps || [],
      },
      yAxis: {
        type: 'value',
        name: 'MB/s',
      },
      series: [
        {
          name: '入站流量',
          type: 'line',
          smooth: true,
          data: data?.inbound_bandwidth || [],
          areaStyle: {
            opacity: 0.3,
          },
          itemStyle: {
            color: '#1890ff',
          },
        },
        {
          name: '出站流量',
          type: 'line',
          smooth: true,
          data: data?.outbound_bandwidth || [],
          areaStyle: {
            opacity: 0.3,
          },
          itemStyle: {
            color: '#52c41a',
          },
        },
      ],
    }
  }

  const getLatencyChartOption = (): EChartsOption => {
    return {
      tooltip: {
        trigger: 'axis',
        formatter: '{b}<br />{a}: {c}ms',
      },
      legend: {
        data: ['平均延迟'],
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: data?.timestamps || [],
      },
      yAxis: {
        type: 'value',
        name: '延迟 (ms)',
      },
      series: [
        {
          name: '平均延迟',
          type: 'line',
          smooth: true,
          data: data?.avg_latency || [],
          itemStyle: {
            color: '#faad14',
          },
        },
      ],
    }
  }

  if (error) {
    return <Alert message="加载指标数据失败" type="error" showIcon />
  }

  return (
    <Spin spinning={isLoading}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        <div style={{ textAlign: 'right' }}>
          <Select
            value={timeRange}
            onChange={setTimeRange}
            style={{ width: 120 }}
            options={[
              { value: '1h', label: '最近1小时' },
              { value: '6h', label: '最近6小时' },
              { value: '24h', label: '最近24小时' },
              { value: '7d', label: '最近7天' },
              { value: '30d', label: '最近30天' },
            ]}
          />
        </div>

        <div>
          <h4 style={{ marginBottom: 16 }}>流量趋势</h4>
          <ReactECharts
            option={getTrafficChartOption()}
            style={{ height: 300 }}
            notMerge={true}
            lazyUpdate={true}
          />
        </div>

        <div>
          <h4 style={{ marginBottom: 16 }}>延迟趋势</h4>
          <ReactECharts
            option={getLatencyChartOption()}
            style={{ height: 300 }}
            notMerge={true}
            lazyUpdate={true}
          />
        </div>
      </Space>
    </Spin>
  )
}

export default MetricsHistory
