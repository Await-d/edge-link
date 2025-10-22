import React from 'react'
import ReactECharts from 'echarts-for-react'
import { Spin } from 'antd'
import { useAlertTrend } from '@/hooks/useApi'
import dayjs from 'dayjs'
import type { EChartsOption } from 'echarts'

/**
 * 告警趋势柱状图组件
 * 展示过去7天内不同严重程度告警的数量变化
 */
const AlertTrend: React.FC = () => {
  const { data, isLoading } = useAlertTrend('7d')

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '40px 0' }}>
        <Spin />
      </div>
    )
  }

  // 准备图表数据
  const dates = data?.data?.map((item) => dayjs(item.date).format('MM-DD')) || []
  const critical = data?.data?.map((item) => item.critical) || []
  const high = data?.data?.map((item) => item.high) || []
  const medium = data?.data?.map((item) => item.medium) || []
  const low = data?.data?.map((item) => item.low) || []

  const option: EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow',
      },
      formatter: (params: any) => {
        if (!Array.isArray(params)) return ''
        let result = `${params[0].axisValue}<br/>`
        let total = 0
        params.forEach((item: any) => {
          result += `${item.marker}${item.seriesName}: ${item.value}<br/>`
          total += item.value
        })
        result += `<b>总计: ${total}</b>`
        return result
      },
    },
    legend: {
      data: ['严重', '高危', '中危', '低危'],
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
      data: dates,
      axisLabel: {
        interval: 0,
      },
    },
    yAxis: {
      type: 'value',
      name: '告警数量',
      minInterval: 1,
    },
    series: [
      {
        name: '严重',
        type: 'bar',
        stack: 'total',
        data: critical,
        itemStyle: {
          color: '#ff4d4f',
        },
      },
      {
        name: '高危',
        type: 'bar',
        stack: 'total',
        data: high,
        itemStyle: {
          color: '#fa8c16',
        },
      },
      {
        name: '中危',
        type: 'bar',
        stack: 'total',
        data: medium,
        itemStyle: {
          color: '#faad14',
        },
      },
      {
        name: '低危',
        type: 'bar',
        stack: 'total',
        data: low,
        itemStyle: {
          color: '#52c41a',
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

export default AlertTrend
