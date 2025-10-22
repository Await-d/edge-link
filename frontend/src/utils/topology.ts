import type { TopologyNode, TopologyLink } from '@/types/topology'

/**
 * Calculate node size based on number of connections
 */
export const calculateNodeSize = (node: TopologyNode, links: TopologyLink[]): number => {
  const connectionCount = links.filter(
    link => link.source === node.id || link.target === node.id
  ).length

  // Base size 40, +10 for each connection, max 100
  return Math.min(40 + connectionCount * 10, 100)
}

/**
 * Get line color based on connection quality
 * Green: latency < 50ms, packetLoss < 1%
 * Yellow: latency < 100ms, packetLoss < 5%
 * Red: latency >= 100ms or packetLoss >= 5%
 */
export const getLineColor = (latency?: number, packetLoss?: number): string => {
  // Default gray if no metrics
  if (latency === undefined && packetLoss === undefined) {
    return '#d9d9d9'
  }

  const lat = latency ?? 0
  const loss = packetLoss ?? 0

  if (lat < 50 && loss < 1) {
    return '#52c41a' // green
  } else if (lat < 100 && loss < 5) {
    return '#faad14' // yellow/orange
  } else {
    return '#ff4d4f' // red
  }
}

/**
 * Get line width based on bandwidth
 */
export const getLineWidth = (bandwidth?: number): number => {
  if (!bandwidth) return 2

  // Scale: 1-10 Mbps => 2px, 10-100 Mbps => 3px, 100+ Mbps => 4px
  if (bandwidth < 10) return 2
  if (bandwidth < 100) return 3
  return 4
}

/**
 * Transform API data to topology format
 */
export const transformToTopology = (
  devices: any[],
  peers: any[]
): { nodes: TopologyNode[], links: TopologyLink[] } => {
  // Map devices to nodes
  const nodes: TopologyNode[] = devices.map(device => ({
    id: device.id,
    name: device.name || device.id,
    virtualIP: device.virtual_ip || device.virtualIP || 'N/A',
    platform: device.platform || 'unknown',
    isOnline: device.is_online || device.isOnline || false,
    category: device.virtual_network_id || device.virtualNetworkId || 'default',
  }))

  // Create a set of valid device IDs for validation
  const deviceIds = new Set(nodes.map(n => n.id))

  // Map peer configurations to links
  const links: TopologyLink[] = peers
    .filter(peer => {
      // Only include links where both devices exist
      return deviceIds.has(peer.device_id || peer.deviceId) &&
             deviceIds.has(peer.peer_device_id || peer.peerDeviceId)
    })
    .map(peer => ({
      source: peer.device_id || peer.deviceId,
      target: peer.peer_device_id || peer.peerDeviceId,
      latency: peer.latency,
      packetLoss: peer.packet_loss || peer.packetLoss,
      bandwidth: peer.bandwidth,
    }))

  return { nodes, links }
}

/**
 * Get unique categories from nodes
 */
export const getCategories = (nodes: TopologyNode[]) => {
  const categorySet = new Set(nodes.map(n => n.category).filter(Boolean))
  const colors = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1', '#13c2c2']

  return Array.from(categorySet).map((cat, index) => ({
    name: cat || 'default',
    itemStyle: {
      color: colors[index % colors.length]
    }
  }))
}

/**
 * Filter nodes by virtual network
 */
export const filterByNetwork = (
  data: { nodes: TopologyNode[], links: TopologyLink[] },
  networkId?: string
) => {
  if (!networkId || networkId === 'all') {
    return data
  }

  const filteredNodes = data.nodes.filter(node => node.category === networkId)
  const nodeIds = new Set(filteredNodes.map(n => n.id))
  const filteredLinks = data.links.filter(
    link => nodeIds.has(link.source) && nodeIds.has(link.target)
  )

  return { nodes: filteredNodes, links: filteredLinks }
}

/**
 * Filter nodes by platform
 */
export const filterByPlatform = (
  data: { nodes: TopologyNode[], links: TopologyLink[] },
  platform?: string
) => {
  if (!platform || platform === 'all') {
    return data
  }

  const filteredNodes = data.nodes.filter(node => node.platform === platform)
  const nodeIds = new Set(filteredNodes.map(n => n.id))
  const filteredLinks = data.links.filter(
    link => nodeIds.has(link.source) && nodeIds.has(link.target)
  )

  return { nodes: filteredNodes, links: filteredLinks }
}
