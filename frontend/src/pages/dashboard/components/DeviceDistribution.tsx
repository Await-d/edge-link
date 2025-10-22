import React from 'react'
import ReactECharts from 'echarts-for-react'
import { Spin } from 'antd'
import { usePlatformDistribution } from '@/hooks/useApi'
import type { EChartsOption } from 'echarts'

/**
 * 设备分布饼图组件
 * 展示不同平台设备的数量分布
 */
const DeviceDistribution: React.FC = () => {
  const { data, isLoading } = usePlatformDistribution()

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '40px 0' }}>
        <Spin />
      </div>
    )
  }

  // 平台颜色映射
  const platformColors: Record<string, string> = {
    linux: '#1890ff',
    windows: '#13c2c2',
    macos: '#722ed1',
    android: '#52c41a',
    ios: '#fa8c16',
  }

  // 平台名称映射
  const platformNames: Record<string, string> = {
    linux: 'Linux',
    windows: 'Windows',
    macos: 'macOS',
    android: 'Android',
    ios: 'iOS',
  }

  // 准备图表数据
  const chartData =
    data?.data?.map((item) => ({
      value: item.count,
      name: platformNames[item.platform] || item.platform,
      itemStyle: {
        color: platformColors[item.platform] || '#d9d9d9',
      },
    })) || []

  const option: EChartsOption = {
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => {
        return `${params.marker}${params.name}: ${params.value} 台 (${params.percent}%)`
      },
    },
    legend: {
      orient: 'vertical',
      right: '10%',
      top: 'center',
      textStyle: {
        fontSize: 12,
      },
    },
    series: [
      {
        name: '设备分布',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2,
        },
        label: {
          show: false,
          position: 'center',
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 18,
            fontWeight: 'bold',
            formatter: (params: any) => {
              return `${params.name}\n${params.value} 台`
            },
          },
        },
        labelLine: {
          show: false,
        },
        data: chartData,
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

export default DeviceDistribution
