import React from 'react'
import ReactECharts from 'echarts-for-react'
import { Spin } from 'antd'
import { useTrafficStats } from '@/hooks/useApi'
import dayjs from 'dayjs'
import type { EChartsOption } from 'echarts'

/**
 * 网络流量图表组件
 * 展示过去24小时内入站和出站流量的变化趋势
 */
const TrafficChart: React.FC = () => {
  const { data, isLoading } = useTrafficStats('24h')

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '40px 0' }}>
        <Spin />
      </div>
    )
  }

  // 准备图表数据
  const timestamps = data?.data?.map((item) => dayjs(item.timestamp).format('HH:mm')) || []
  const inbound = data?.data?.map((item) => item.inbound) || []
  const outbound = data?.data?.map((item) => item.outbound) || []

  const option: EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
      },
      formatter: (params: any) => {
        if (!Array.isArray(params)) return ''
        let result = `${params[0].axisValue}<br/>`
        params.forEach((item: any) => {
          result += `${item.marker}${item.seriesName}: ${item.value.toFixed(2)} MB/s<br/>`
        })
        return result
      },
    },
    legend: {
      data: ['入站流量', '出站流量'],
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
        interval: Math.floor(timestamps.length / 8),
      },
    },
    yAxis: {
      type: 'value',
      name: 'MB/s',
      axisLabel: {
        formatter: '{value}',
      },
    },
    series: [
      {
        name: '入站流量',
        type: 'line',
        smooth: true,
        data: inbound,
        stack: 'traffic',
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
              { offset: 0, color: 'rgba(82, 196, 26, 0.4)' },
              { offset: 1, color: 'rgba(82, 196, 26, 0.1)' },
            ],
          },
        },
      },
      {
        name: '出站流量',
        type: 'line',
        smooth: true,
        data: outbound,
        stack: 'traffic',
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
              { offset: 0, color: 'rgba(24, 144, 255, 0.4)' },
              { offset: 1, color: 'rgba(24, 144, 255, 0.1)' },
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

export default TrafficChart
