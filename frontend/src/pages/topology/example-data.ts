/**
 * Example topology data for testing and demonstration
 * This file shows how the topology graph would look with real data
 */

import type { TopologyData } from '@/types/topology'

export const exampleTopologyData: TopologyData = {
  nodes: [
    // Virtual Network A - Production
    {
      id: 'dev-001',
      name: 'Gateway Server',
      virtualIP: '10.0.1.1',
      platform: 'linux',
      isOnline: true,
      category: 'prod-network',
    },
    {
      id: 'dev-002',
      name: 'Web Server 1',
      virtualIP: '10.0.1.10',
      platform: 'linux',
      isOnline: true,
      category: 'prod-network',
    },
    {
      id: 'dev-003',
      name: 'Web Server 2',
      virtualIP: '10.0.1.11',
      platform: 'linux',
      isOnline: true,
      category: 'prod-network',
    },
    {
      id: 'dev-004',
      name: 'Database Server',
      virtualIP: '10.0.1.20',
      platform: 'linux',
      isOnline: true,
      category: 'prod-network',
    },

    // Virtual Network B - Development
    {
      id: 'dev-005',
      name: 'Dev MacBook',
      virtualIP: '10.0.2.5',
      platform: 'darwin',
      isOnline: true,
      category: 'dev-network',
    },
    {
      id: 'dev-006',
      name: 'Dev Desktop',
      virtualIP: '10.0.2.6',
      platform: 'windows',
      isOnline: true,
      category: 'dev-network',
    },
    {
      id: 'dev-007',
      name: 'Test Server',
      virtualIP: '10.0.2.10',
      platform: 'linux',
      isOnline: false, // Offline node
      category: 'dev-network',
    },

    // IoT Network
    {
      id: 'dev-008',
      name: 'IoT Gateway',
      virtualIP: '10.0.3.1',
      platform: 'linux',
      isOnline: true,
      category: 'iot-network',
    },
    {
      id: 'dev-009',
      name: 'Sensor Node 1',
      virtualIP: '10.0.3.10',
      platform: 'linux',
      isOnline: true,
      category: 'iot-network',
    },
    {
      id: 'dev-010',
      name: 'Sensor Node 2',
      virtualIP: '10.0.3.11',
      platform: 'linux',
      isOnline: true,
      category: 'iot-network',
    },
  ],
  links: [
    // Production network connections
    {
      source: 'dev-001', // Gateway
      target: 'dev-002', // Web Server 1
      latency: 12,
      packetLoss: 0.1,
      bandwidth: 100,
    },
    {
      source: 'dev-001', // Gateway
      target: 'dev-003', // Web Server 2
      latency: 15,
      packetLoss: 0.2,
      bandwidth: 100,
    },
    {
      source: 'dev-002', // Web Server 1
      target: 'dev-004', // Database
      latency: 8,
      packetLoss: 0.0,
      bandwidth: 1000,
    },
    {
      source: 'dev-003', // Web Server 2
      target: 'dev-004', // Database
      latency: 9,
      packetLoss: 0.0,
      bandwidth: 1000,
    },

    // Development network connections
    {
      source: 'dev-005', // Dev MacBook
      target: 'dev-006', // Dev Desktop
      latency: 45,
      packetLoss: 0.5,
      bandwidth: 50,
    },
    {
      source: 'dev-005', // Dev MacBook
      target: 'dev-007', // Test Server (offline)
      latency: 150,
      packetLoss: 8.0,
      bandwidth: 10,
    },
    {
      source: 'dev-006', // Dev Desktop
      target: 'dev-007', // Test Server (offline)
      latency: 140,
      packetLoss: 7.5,
      bandwidth: 10,
    },

    // IoT network connections
    {
      source: 'dev-008', // IoT Gateway
      target: 'dev-009', // Sensor 1
      latency: 25,
      packetLoss: 1.0,
      bandwidth: 10,
    },
    {
      source: 'dev-008', // IoT Gateway
      target: 'dev-010', // Sensor 2
      latency: 28,
      packetLoss: 1.2,
      bandwidth: 10,
    },

    // Cross-network connections
    {
      source: 'dev-001', // Gateway (prod)
      target: 'dev-005', // Dev MacBook
      latency: 85,
      packetLoss: 2.5,
      bandwidth: 50,
    },
    {
      source: 'dev-001', // Gateway (prod)
      target: 'dev-008', // IoT Gateway
      latency: 65,
      packetLoss: 1.5,
      bandwidth: 20,
    },
  ],
}

/**
 * Graph Characteristics:
 *
 * Total Nodes: 10 devices
 * - Production: 4 (all online)
 * - Development: 3 (1 offline)
 * - IoT: 3 (all online)
 *
 * Total Links: 12 P2P connections
 * - Green (excellent): 6 links (latency < 50ms, packetLoss < 1%)
 * - Yellow (good): 3 links (latency < 100ms, packetLoss < 5%)
 * - Red (poor): 3 links (latency >= 100ms or packetLoss >= 5%)
 *
 * Topology Features Demonstrated:
 * 1. Multi-network grouping with categories
 * 2. Different platforms (Linux, macOS, Windows)
 * 3. Online/offline status visualization
 * 4. Connection quality color coding
 * 5. Hub-and-spoke patterns (Gateway, IoT Gateway)
 * 6. Cross-network connections
 * 7. Various latency/packet loss scenarios
 *
 * Interactive Features:
 * - Zoom in/out to see details
 * - Drag nodes to rearrange layout
 * - Hover over nodes to see device info
 * - Hover over edges to see connection metrics
 * - Click nodes to open details drawer
 * - Filter by network or platform
 * - Toggle label visibility
 */
