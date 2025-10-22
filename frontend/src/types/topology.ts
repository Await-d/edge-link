export interface TopologyNode {
  id: string
  name: string
  virtualIP: string
  platform: string
  isOnline: boolean
  category?: string // For grouping by virtual network
}

export interface TopologyLink {
  source: string // device ID
  target: string // peer device ID
  latency?: number // ms
  packetLoss?: number // percentage
  bandwidth?: number // Mbps
}

export interface TopologyData {
  nodes: TopologyNode[]
  links: TopologyLink[]
}

export interface TopologyCategory {
  name: string
  itemStyle?: {
    color?: string
  }
}
