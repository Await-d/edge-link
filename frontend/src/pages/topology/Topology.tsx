import React, { useState, useMemo } from 'react'
import {
  Card,
  Typography,
  Empty,
  Space,
  Button,
  Select,
  Switch,
  Drawer,
  Descriptions,
  Tag,
  Skeleton,
  message,
} from 'antd'
import {
  ReloadOutlined,
  FullscreenOutlined,
  DownloadOutlined,
  FilterOutlined,
} from '@ant-design/icons'
import TopologyGraph from './components/TopologyGraph'
import { useTopologyData, useDevice } from '@/hooks/useApi'
import { filterByNetwork, filterByPlatform } from '@/utils/topology'

const { Title } = Typography

const Topology: React.FC = () => {
  const { data, isLoading, refetch } = useTopologyData()
  const [selectedDeviceId, setSelectedDeviceId] = useState<string | null>(null)
  const [showLabels, setShowLabels] = useState(true)
  const [selectedNetwork, setSelectedNetwork] = useState<string>('all')
  const [selectedPlatform, setSelectedPlatform] = useState<string>('all')

  const { data: deviceDetail } = useDevice(selectedDeviceId || '')

  // Apply filters
  const filteredData = useMemo(() => {
    if (!data) return null

    let result = data

    if (selectedNetwork !== 'all') {
      result = filterByNetwork(result, selectedNetwork)
    }

    if (selectedPlatform !== 'all') {
      result = filterByPlatform(result, selectedPlatform)
    }

    return result
  }, [data, selectedNetwork, selectedPlatform])

  // Get unique networks and platforms for filters
  const networks = useMemo(() => {
    if (!data) return []
    const networkSet = new Set(data.nodes.map(n => n.category).filter(Boolean))
    return Array.from(networkSet)
  }, [data])

  const platforms = useMemo(() => {
    if (!data) return []
    const platformSet = new Set(data.nodes.map(n => n.platform))
    return Array.from(platformSet)
  }, [data])

  const handleRefresh = () => {
    refetch()
    message.success('拓扑数据已刷新')
  }

  const handleExport = () => {
    message.info('导出功能开发中')
  }

  const handleFullscreen = () => {
    const elem = document.documentElement
    if (!document.fullscreenElement) {
      elem.requestFullscreen().catch((err) => {
        message.error(`全屏失败: ${err.message}`)
      })
    } else {
      document.exitFullscreen()
    }
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>网络拓扑</Title>

        <Space>
          <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
            刷新
          </Button>
          <Button icon={<DownloadOutlined />} onClick={handleExport}>
            导出
          </Button>
          <Button icon={<FullscreenOutlined />} onClick={handleFullscreen}>
            全屏
          </Button>
        </Space>
      </div>

      {/* Filter controls */}
      <Card
        size="small"
        style={{ marginTop: 16 }}
        bodyStyle={{ padding: '12px 16px' }}
      >
        <Space size="middle" wrap>
          <Space>
            <FilterOutlined />
            <span>虚拟网络:</span>
            <Select
              value={selectedNetwork}
              onChange={setSelectedNetwork}
              style={{ width: 150 }}
              options={[
                { label: '全部', value: 'all' },
                ...networks.map(n => ({ label: n, value: n })),
              ]}
            />
          </Space>

          <Space>
            <span>平台:</span>
            <Select
              value={selectedPlatform}
              onChange={setSelectedPlatform}
              style={{ width: 150 }}
              options={[
                { label: '全部', value: 'all' },
                ...platforms.map(p => ({ label: p, value: p })),
              ]}
            />
          </Space>

          <Space>
            <span>显示标签:</span>
            <Switch checked={showLabels} onChange={setShowLabels} />
          </Space>

          <Space style={{ marginLeft: 'auto' }}>
            <span style={{ color: '#52c41a' }}>在线</span>
            <span>|</span>
            <span style={{ color: '#d9d9d9' }}>离线</span>
            <span>|</span>
            <span style={{ color: '#52c41a' }}>优质</span>
            <span style={{ color: '#faad14' }}>一般</span>
            <span style={{ color: '#ff4d4f' }}>较差</span>
          </Space>
        </Space>
      </Card>

      {/* Topology graph */}
      <Card
        style={{ marginTop: 16, height: 'calc(100vh - 280px)', minHeight: 500 }}
        bodyStyle={{ padding: 16, height: '100%' }}
      >
        {isLoading ? (
          <Skeleton active paragraph={{ rows: 10 }} />
        ) : filteredData && filteredData.nodes.length > 0 ? (
          <TopologyGraph
            data={filteredData}
            onNodeClick={setSelectedDeviceId}
            showLabels={showLabels}
          />
        ) : (
          <Empty
            description={
              data && data.nodes.length === 0
                ? '暂无设备数据'
                : '没有符合筛选条件的设备'
            }
          />
        )}
      </Card>

      {/* Device details drawer */}
      <Drawer
        title="设备详情"
        placement="right"
        width={500}
        open={!!selectedDeviceId}
        onClose={() => setSelectedDeviceId(null)}
      >
        {deviceDetail && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="设备名称">
              {deviceDetail.name || deviceDetail.id}
            </Descriptions.Item>
            <Descriptions.Item label="设备ID">
              {deviceDetail.id}
            </Descriptions.Item>
            <Descriptions.Item label="虚拟IP">
              {deviceDetail.virtual_ip || 'N/A'}
            </Descriptions.Item>
            <Descriptions.Item label="平台">
              {deviceDetail.platform || 'unknown'}
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={deviceDetail.is_online ? 'green' : 'default'}>
                {deviceDetail.is_online ? '在线' : '离线'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="虚拟网络">
              {deviceDetail.virtual_network_id || 'default'}
            </Descriptions.Item>
            {deviceDetail.public_key && (
              <Descriptions.Item label="公钥">
                <code style={{ fontSize: 10, wordBreak: 'break-all' }}>
                  {deviceDetail.public_key}
                </code>
              </Descriptions.Item>
            )}
            {deviceDetail.last_seen_at && (
              <Descriptions.Item label="最后在线">
                {new Date(deviceDetail.last_seen_at).toLocaleString('zh-CN')}
              </Descriptions.Item>
            )}
            {deviceDetail.created_at && (
              <Descriptions.Item label="创建时间">
                {new Date(deviceDetail.created_at).toLocaleString('zh-CN')}
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Drawer>
    </div>
  )
}

export default Topology
