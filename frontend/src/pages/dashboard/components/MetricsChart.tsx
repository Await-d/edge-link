import React from 'react'
import ReactECharts from 'echarts-for-react'
import { Spin } from 'antd'
import { useDeviceTrend } from '@/hooks/useApi'
import dayjs from 'dayjs'
import type { EChartsOption } from 'echarts'

/**
 * 设备趋势图表组件
 * 展示过去24小时内设备总数和在线设备数的变化趋势
 */
const MetricsChart: React.FC = () => {
  const { data, isLoading } = useDeviceTrend('24h')

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '40px 0' }}>
        <Spin />
      </div>
    )
  }

  // 准备图表数据
  const timestamps = data?.data?.map((item) => dayjs(item.timestamp).format('HH:mm')) || []
  const totalDevices = data?.data?.map((item) => item.total_devices) || []
  const onlineDevices = data?.data?.map((item) => item.online_devices) || []

  const option: EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985',
        },
      },
    },
    legend: {
      data: ['总设备', '在线设备'],
      bottom: 0,
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '12%',
      top: '5%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: timestamps,
      axisLabel: {
        rotate: 45,
        interval: Math.floor(timestamps.length / 8), // 显示8个时间标签
      },
    },
    yAxis: {
      type: 'value',
      name: '设备数',
      minInterval: 1,
    },
    series: [
      {
        name: '总设备',
        type: 'line',
        smooth: true,
        data: totalDevices,
        lineStyle: {
          width: 2,
        },
        itemStyle: {
          color: '#1890ff',
        },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(24, 144, 255, 0.3)' },
              { offset: 1, color: 'rgba(24, 144, 255, 0.05)' },
            ],
          },
        },
      },
      {
        name: '在线设备',
        type: 'line',
        smooth: true,
        data: onlineDevices,
        lineStyle: {
          width: 2,
        },
        itemStyle: {
          color: '#52c41a',
        },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(82, 196, 26, 0.3)' },
              { offset: 1, color: 'rgba(82, 196, 26, 0.05)' },
            ],
          },
        },
      },
    ],
  }

  return (
    <ReactECharts
      option={option}
      style={{ height: 300 }}
      opts={{ renderer: 'canvas' }}
    />
  )
}

export default MetricsChart
