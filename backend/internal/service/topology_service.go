package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/edgelink/backend/internal/crypto"
	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
)

// TopologyService 拓扑服务
type TopologyService struct {
	virtualNetworkRepo repository.VirtualNetworkRepository
	deviceRepo         repository.DeviceRepository

	// IP池管理（内存缓存）
	ipPoolMu sync.RWMutex
	ipPools  map[uuid.UUID]*IPPool // 按虚拟网络ID索引
}

// IPPool IP地址池
type IPPool struct {
	CIDR        string
	Network     *net.IPNet
	AllocatedIPs map[string]bool // IP -> 是否已分配
	mu          sync.RWMutex
}

// NewTopologyService 创建拓扑服务实例
func NewTopologyService(
	vnRepo repository.VirtualNetworkRepository,
	deviceRepo repository.DeviceRepository,
) *TopologyService {
	return &TopologyService{
		virtualNetworkRepo: vnRepo,
		deviceRepo:         deviceRepo,
		ipPools:            make(map[uuid.UUID]*IPPool),
	}
}

// AllocateVirtualIP 为设备分配虚拟IP
func (s *TopologyService) AllocateVirtualIP(ctx context.Context, virtualNetworkID uuid.UUID) (string, error) {
	// 1. 获取虚拟网络信息
	vn, err := s.virtualNetworkRepo.FindByID(ctx, virtualNetworkID)
	if err != nil {
		return "", fmt.Errorf("virtual network not found: %w", err)
	}

	// 2. 获取或初始化IP池
	pool, err := s.getOrInitIPPool(ctx, vn)
	if err != nil {
		return "", fmt.Errorf("failed to initialize IP pool: %w", err)
	}

	// 3. 从池中分配IP
	ip, err := pool.Allocate()
	if err != nil {
		return "", fmt.Errorf("failed to allocate IP: %w", err)
	}

	return ip, nil
}

// GetPeerConfigurations 获取设备的对等配置
func (s *TopologyService) GetPeerConfigurations(ctx context.Context, deviceID uuid.UUID) ([]crypto.WireGuardPeerConfig, error) {
	// 1. 获取设备信息
	device, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// 2. 获取同一虚拟网络下的所有其他设备
	onlineFilter := true
	peers, err := s.deviceRepo.FindByVirtualNetwork(ctx, device.VirtualNetworkID, &onlineFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch peers: %w", err)
	}

	// 3. 构建对等配置列表
	peerConfigs := make([]crypto.WireGuardPeerConfig, 0, len(peers))
	for _, peer := range peers {
		// 跳过自己
		if peer.ID == deviceID {
			continue
		}

		// 只包含在线设备
		if !peer.Online {
			continue
		}

		peerConfig := crypto.WireGuardPeerConfig{
			PublicKey:  peer.PublicKey,
			AllowedIPs: []string{fmt.Sprintf("%s/32", peer.VirtualIP)},
		}

		// 如果对等设备有公网端点，添加到配置
		if peer.PublicEndpoint != "" {
			peerConfig.Endpoint = peer.PublicEndpoint
		}

		// 对于NAT后的设备，启用持久保活
		if peer.NATType != domain.NATTypeFullCone && peer.NATType != domain.NATTypeNone {
			peerConfig.PersistentKeepalive = 25 // 25秒
		}

		peerConfigs = append(peerConfigs, peerConfig)
	}

	return peerConfigs, nil
}

// GenerateWireGuardConfig 生成完整的WireGuard配置
func (s *TopologyService) GenerateWireGuardConfig(ctx context.Context, deviceID uuid.UUID, privateKey string) (*crypto.WireGuardConfig, error) {
	// 1. 获取设备信息
	device, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// 2. 获取虚拟网络信息
	vn, err := s.virtualNetworkRepo.FindByID(ctx, device.VirtualNetworkID)
	if err != nil {
		return nil, fmt.Errorf("virtual network not found: %w", err)
	}

	// 3. 获取对等配置
	peers, err := s.GetPeerConfigurations(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get peer configurations: %w", err)
	}

	// 4. 构建接口配置
	_, ipNet, err := net.ParseCIDR(vn.CIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	maskBits, _ := ipNet.Mask.Size()
	interfaceConfig := crypto.WireGuardInterfaceConfig{
		PrivateKey: privateKey,
		Address:    fmt.Sprintf("%s/%d", device.VirtualIP, maskBits),
		ListenPort: 51820, // 默认端口
	}

	// 5. 构建完整配置
	config := &crypto.WireGuardConfig{
		Interface: interfaceConfig,
		Peers:     peers,
	}

	return config, nil
}

// RefreshVirtualNetworkTopology 刷新虚拟网络拓扑（重新计算对等关系）
func (s *TopologyService) RefreshVirtualNetworkTopology(ctx context.Context, virtualNetworkID uuid.UUID) error {
	// 1. 获取虚拟网络下的所有设备
	devices, err := s.deviceRepo.FindByVirtualNetwork(ctx, virtualNetworkID, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch devices: %w", err)
	}

	// 2. 对每个设备，计算并缓存其对等配置
	// （实际生产环境可能需要推送WebSocket通知，触发客户端重新拉取配置）
	for _, device := range devices {
		if !device.Online {
			continue
		}

		// 记录拓扑更新时间（可选）
		device.UpdatedAt = time.Now()
		if err := s.deviceRepo.Update(ctx, &device); err != nil {
			// 记录日志但不失败
			fmt.Printf("warning: failed to update device %s: %v\n", device.ID, err)
		}
	}

	return nil
}

// ReleaseVirtualIP 释放虚拟IP（设备撤销时调用）
func (s *TopologyService) ReleaseVirtualIP(ctx context.Context, virtualNetworkID uuid.UUID, ip string) error {
	s.ipPoolMu.RLock()
	pool, exists := s.ipPools[virtualNetworkID]
	s.ipPoolMu.RUnlock()

	if !exists {
		// IP池不存在，无需释放
		return nil
	}

	pool.Release(ip)
	return nil
}

// getOrInitIPPool 获取或初始化IP池
func (s *TopologyService) getOrInitIPPool(ctx context.Context, vn *domain.VirtualNetwork) (*IPPool, error) {
	s.ipPoolMu.RLock()
	pool, exists := s.ipPools[vn.ID]
	s.ipPoolMu.RUnlock()

	if exists {
		return pool, nil
	}

	// 需要初始化IP池
	s.ipPoolMu.Lock()
	defer s.ipPoolMu.Unlock()

	// 双重检查
	if pool, exists := s.ipPools[vn.ID]; exists {
		return pool, nil
	}

	// 解析CIDR
	_, ipNet, err := net.ParseCIDR(vn.CIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	pool = &IPPool{
		CIDR:         vn.CIDR,
		Network:      ipNet,
		AllocatedIPs: make(map[string]bool),
	}

	// 查询已分配的IP
	devices, err := s.deviceRepo.FindByVirtualNetwork(ctx, vn.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing devices: %w", err)
	}

	for _, device := range devices {
		pool.AllocatedIPs[device.VirtualIP] = true
	}

	s.ipPools[vn.ID] = pool
	return pool, nil
}

// Allocate 从IP池中分配一个可用IP
func (p *IPPool) Allocate() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 遍历网络范围内的所有IP
	for ip := incrementIP(p.Network.IP); p.Network.Contains(ip); ip = incrementIP(ip) {
		ipStr := ip.String()

		// 跳过网络地址和广播地址
		if ip.Equal(p.Network.IP) || isBroadcast(ip, p.Network) {
			continue
		}

		// 检查是否已分配
		if !p.AllocatedIPs[ipStr] {
			p.AllocatedIPs[ipStr] = true
			return ipStr, nil
		}
	}

	return "", fmt.Errorf("IP pool exhausted")
}

// Release 释放IP地址
func (p *IPPool) Release(ip string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.AllocatedIPs, ip)
}

// incrementIP IP地址自增
func incrementIP(ip net.IP) net.IP {
	// 复制IP以避免修改原始值
	newIP := make(net.IP, len(ip))
	copy(newIP, ip)

	// 从最后一个字节开始递增
	for i := len(newIP) - 1; i >= 0; i-- {
		newIP[i]++
		if newIP[i] != 0 {
			break
		}
	}

	return newIP
}

// isBroadcast 检查是否为广播地址
func isBroadcast(ip net.IP, network *net.IPNet) bool {
	broadcast := make(net.IP, len(network.IP))
	for i := range network.IP {
		broadcast[i] = network.IP[i] | ^network.Mask[i]
	}
	return ip.Equal(broadcast)
}
