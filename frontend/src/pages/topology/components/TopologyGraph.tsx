import React, { useEffect, useRef, useMemo } from 'react'
import * as echarts from 'echarts'
import type { ECharts } from 'echarts'
import type { TopologyData } from '@/types/topology'
import {
  calculateNodeSize,
  getLineColor,
  getLineWidth,
  getCategories,
} from '@/utils/topology'

interface TopologyGraphProps {
  data: TopologyData
  onNodeClick?: (nodeId: string) => void
  showLabels?: boolean
}

const TopologyGraph: React.FC<TopologyGraphProps> = ({
  data,
  onNodeClick,
  showLabels = true,
}) => {
  const chartRef = useRef<HTMLDivElement>(null)
  const chartInstance = useRef<ECharts | null>(null)

  // Calculate categories for legend
  const categories = useMemo(() => getCategories(data.nodes), [data.nodes])

  useEffect(() => {
    if (!chartRef.current) return

    // Initialize chart
    if (!chartInstance.current) {
      chartInstance.current = echarts.init(chartRef.current)
    }

    const chart = chartInstance.current

    // Prepare node data for ECharts
    const graphNodes = data.nodes.map(node => ({
      id: node.id,
      name: node.name,
      value: node.virtualIP,
      symbolSize: calculateNodeSize(node, data.links),
      itemStyle: {
        color: node.isOnline ? '#52c41a' : '#d9d9d9',
        borderColor: '#fff',
        borderWidth: 2,
      },
      category: node.category || 'default',
      // Store original data for tooltip
      originalData: node,
    }))

    // Prepare link data for ECharts
    const graphLinks = data.links.map(link => ({
      source: link.source,
      target: link.target,
      lineStyle: {
        color: getLineColor(link.latency, link.packetLoss),
        width: getLineWidth(link.bandwidth),
        curveness: 0.2,
      },
      label: {
        show: false,
        formatter: link.latency ? `${link.latency}ms` : '',
      },
      // Store original data for tooltip
      originalData: link,
    }))

    const option: echarts.EChartsOption = {
      title: {
        show: false,
      },
      tooltip: {
        trigger: 'item',
        formatter: (params: any) => {
          if (params.dataType === 'node') {
            const node = params.data.originalData
            return `
              <div style="padding: 8px;">
                <strong>${node.name}</strong><br/>
                虚拟IP: ${node.virtualIP}<br/>
                平台: ${node.platform}<br/>
                状态: <span style="color: ${node.isOnline ? '#52c41a' : '#d9d9d9'}">
                  ${node.isOnline ? '在线' : '离线'}
                </span><br/>
                虚拟网络: ${node.category || 'default'}
              </div>
            `
          } else if (params.dataType === 'edge') {
            const link = params.data.originalData
            const parts: string[] = []

            if (link.latency !== undefined) {
              parts.push(`延迟: ${link.latency}ms`)
            }
            if (link.packetLoss !== undefined) {
              parts.push(`丢包率: ${link.packetLoss}%`)
            }
            if (link.bandwidth !== undefined) {
              parts.push(`带宽: ${link.bandwidth}Mbps`)
            }

            return parts.length > 0
              ? `<div style="padding: 8px;">${parts.join('<br/>')}</div>`
              : '连接信息暂无'
          }
          return ''
        },
        backgroundColor: 'rgba(255, 255, 255, 0.95)',
        borderColor: '#d9d9d9',
        borderWidth: 1,
        textStyle: {
          color: '#000',
        },
      },
      legend: [
        {
          data: categories.map(c => c.name),
          orient: 'vertical',
          left: 10,
          top: 20,
          textStyle: {
            color: '#666',
          },
        },
      ],
      series: [
        {
          type: 'graph',
          layout: 'force',
          data: graphNodes,
          links: graphLinks,
          categories: categories,
          roam: true, // Enable zoom and pan
          draggable: true,
          label: {
            show: showLabels,
            position: 'right',
            formatter: '{b}',
            fontSize: 12,
            color: '#333',
          },
          force: {
            repulsion: 1000,
            edgeLength: [100, 200],
            gravity: 0.1,
            friction: 0.6,
          },
          emphasis: {
            focus: 'adjacency',
            label: {
              show: true,
              fontSize: 14,
              fontWeight: 'bold',
            },
            lineStyle: {
              width: 4,
            },
          },
          lineStyle: {
            opacity: 0.8,
          },
        },
      ],
      animationDuration: 1500,
      animationEasingUpdate: 'quinticInOut',
    }

    chart.setOption(option)

    // Handle node click events
    if (onNodeClick) {
      chart.off('click')
      chart.on('click', (params: any) => {
        if (params.dataType === 'node') {
          onNodeClick(params.data.id)
        }
      })
    }

    // Handle window resize
    const handleResize = () => {
      chart.resize()
    }
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      chart.off('click')
    }
  }, [data, categories, onNodeClick, showLabels])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (chartInstance.current) {
        chartInstance.current.dispose()
        chartInstance.current = null
      }
    }
  }, [])

  return (
    <div
      ref={chartRef}
      style={{
        width: '100%',
        height: '100%',
        minHeight: '500px',
      }}
    />
  )
}

export default TopologyGraph
