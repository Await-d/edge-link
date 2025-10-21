package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
)

// NATCoordinator NAT协调服务
type NATCoordinator struct {
	deviceRepo        repository.DeviceRepository
	stunServerAddress string

	// STUN探测结果缓存
	natCacheMu sync.RWMutex
	natCache   map[uuid.UUID]*NATProbeResult // deviceID -> 探测结果
}

// NATProbeResult STUN探测结果
type NATProbeResult struct {
	DeviceID       uuid.UUID
	NATType        domain.NATType
	PublicEndpoint string // IP:Port
	LocalEndpoint  string // 设备上报的本地端口
	ProbeTime      time.Time
	TTL            time.Duration // 结果有效期
}

// NewNATCoordinator 创建NAT协调器实例
func NewNATCoordinator(
	deviceRepo repository.DeviceRepository,
	stunServerAddress string,
) *NATCoordinator {
	return &NATCoordinator{
		deviceRepo:        deviceRepo,
		stunServerAddress: stunServerAddress,
		natCache:          make(map[uuid.UUID]*NATProbeResult),
	}
}

// ProbeNATType 探测设备的NAT类型
func (nc *NATCoordinator) ProbeNATType(ctx context.Context, deviceID uuid.UUID, localEndpoint string) (*NATProbeResult, error) {
	// 检查缓存
	nc.natCacheMu.RLock()
	cached, exists := nc.natCache[deviceID]
	nc.natCacheMu.RUnlock()

	if exists && time.Since(cached.ProbeTime) < cached.TTL {
		return cached, nil
	}

	// TODO: 实现完整的STUN探测逻辑
	// 当前为简化实现，实际生产需要：
	// 1. 使用RFC 5389 STUN协议进行多次探测
	// 2. 发送STUN Binding Request到多个STUN服务器
	// 3. 分析响应中的MAPPED-ADDRESS和XOR-MAPPED-ADDRESS
	// 4. 根据RFC 5780判断NAT类型（Full Cone, Restricted, Port Restricted, Symmetric）

	result := &NATProbeResult{
		DeviceID:      deviceID,
		NATType:       domain.NATTypeUnknown, // 占位符
		LocalEndpoint: localEndpoint,
		ProbeTime:     time.Now(),
		TTL:           5 * time.Minute, // 5分钟缓存
	}

	// 存入缓存
	nc.natCacheMu.Lock()
	nc.natCache[deviceID] = result
	nc.natCacheMu.Unlock()

	// 更新数据库中的NAT类型
	device, err := nc.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	device.NATType = result.NATType
	device.PublicEndpoint = result.PublicEndpoint
	device.UpdatedAt = time.Now()

	if err := nc.deviceRepo.Update(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to update device NAT info: %w", err)
	}

	return result, nil
}

// CoordinateHolePunching 协调UDP打洞
func (nc *NATCoordinator) CoordinateHolePunching(ctx context.Context, deviceA, deviceB uuid.UUID) (*HolePunchingSession, error) {
	// 1. 获取两个设备的NAT探测结果
	resultA, err := nc.getNATResult(ctx, deviceA)
	if err != nil {
		return nil, fmt.Errorf("failed to get NAT result for device A: %w", err)
	}

	resultB, err := nc.getNATResult(ctx, deviceB)
	if err != nil {
		return nil, fmt.Errorf("failed to get NAT result for device B: %w", err)
	}

	// 2. 判断是否可以直接打洞
	canPunch, method := nc.evaluateHolePunchingFeasibility(resultA.NATType, resultB.NATType)

	session := &HolePunchingSession{
		DeviceA:     deviceA,
		DeviceB:     deviceB,
		Method:      method,
		CanPunch:    canPunch,
		EndpointA:   resultA.PublicEndpoint,
		EndpointB:   resultB.PublicEndpoint,
		CoordinatedAt: time.Now(),
	}

	// 3. 如果不能直接打洞，分配TURN中继
	if !canPunch {
		turnAllocation, err := nc.allocateTURNRelay(ctx, deviceA, deviceB)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate TURN relay: %w", err)
		}
		session.TURNRelay = turnAllocation
	}

	return session, nil
}

// HolePunchingSession UDP打洞会话
type HolePunchingSession struct {
	DeviceA       uuid.UUID
	DeviceB       uuid.UUID
	Method        string // "direct", "stun", "turn"
	CanPunch      bool
	EndpointA     string
	EndpointB     string
	TURNRelay     *TURNAllocation
	CoordinatedAt time.Time
}

// TURNAllocation TURN中继分配
type TURNAllocation struct {
	RelayAddress string        // TURN服务器地址
	Username     string        // TURN认证用户名
	Password     string        // TURN认证密码
	Lifetime     time.Duration // 分配有效期
}

// evaluateHolePunchingFeasibility 评估打洞可行性
func (nc *NATCoordinator) evaluateHolePunchingFeasibility(natA, natB domain.NATType) (bool, string) {
	// 根据NAT组合判断打洞策略
	switch {
	case natA == domain.NATTypeNone || natB == domain.NATTypeNone:
		// 至少一方无NAT，可以直接连接
		return true, "direct"

	case natA == domain.NATTypeFullCone || natB == domain.NATTypeFullCone:
		// 至少一方是Full Cone NAT，可以通过STUN打洞
		return true, "stun"

	case natA == domain.NATTypeRestrictedCone && natB == domain.NATTypeRestrictedCone:
		// 两方都是Restricted Cone，可以尝试同步打洞
		return true, "stun"

	case natA == domain.NATTypePortRestrictedCone && natB == domain.NATTypePortRestrictedCone:
		// 两方都是Port Restricted Cone，可以尝试同步打洞（难度较高）
		return true, "stun"

	case natA == domain.NATTypeSymmetric || natB == domain.NATTypeSymmetric:
		// 至少一方是Symmetric NAT，需要TURN中继
		return false, "turn"

	default:
		// 未知类型，默认使用TURN
		return false, "turn"
	}
}

// allocateTURNRelay 分配TURN中继
func (nc *NATCoordinator) allocateTURNRelay(ctx context.Context, deviceA, deviceB uuid.UUID) (*TURNAllocation, error) {
	// TODO: 实现完整的TURN分配逻辑
	// 当前为简化实现，实际生产需要：
	// 1. 从TURN服务器池中选择负载最低的服务器
	// 2. 使用RFC 5766 TURN协议发送Allocate Request
	// 3. 接收Allocation Response，获取中继地址
	// 4. 生成临时认证凭据（username/password）
	// 5. 设置分配生命周期并定期刷新

	allocation := &TURNAllocation{
		RelayAddress: "turn.example.com:3478", // 占位符
		Username:     fmt.Sprintf("turn-%s-%s", deviceA.String()[:8], deviceB.String()[:8]),
		Password:     generateTURNPassword(), // 临时密码
		Lifetime:     10 * time.Minute,       // 10分钟有效期
	}

	return allocation, nil
}

// getNATResult 获取设备的NAT探测结果（带缓存）
func (nc *NATCoordinator) getNATResult(ctx context.Context, deviceID uuid.UUID) (*NATProbeResult, error) {
	nc.natCacheMu.RLock()
	cached, exists := nc.natCache[deviceID]
	nc.natCacheMu.RUnlock()

	if exists && time.Since(cached.ProbeTime) < cached.TTL {
		return cached, nil
	}

	// 缓存未命中，从数据库读取
	device, err := nc.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	result := &NATProbeResult{
		DeviceID:       device.ID,
		NATType:        device.NATType,
		PublicEndpoint: device.PublicEndpoint,
		ProbeTime:      device.UpdatedAt,
		TTL:            5 * time.Minute,
	}

	nc.natCacheMu.Lock()
	nc.natCache[deviceID] = result
	nc.natCacheMu.Unlock()

	return result, nil
}

// UpdatePublicEndpoint 更新设备的公网端点
func (nc *NATCoordinator) UpdatePublicEndpoint(ctx context.Context, deviceID uuid.UUID, endpoint string) error {
	// 验证端点格式
	if _, err := net.ResolveTCPAddr("tcp", endpoint); err != nil {
		return fmt.Errorf("invalid endpoint format: %w", err)
	}

	device, err := nc.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	device.PublicEndpoint = endpoint
	device.UpdatedAt = time.Now()

	if err := nc.deviceRepo.Update(ctx, device); err != nil {
		return fmt.Errorf("failed to update device endpoint: %w", err)
	}

	// 使缓存失效
	nc.natCacheMu.Lock()
	delete(nc.natCache, deviceID)
	nc.natCacheMu.Unlock()

	return nil
}

// generateTURNPassword 生成TURN临时密码
func generateTURNPassword() string {
	// TODO: 使用加密安全的随机生成器
	// 当前为占位符实现
	return uuid.New().String()
}

// CleanupExpiredSessions 清理过期的打洞会话（定期后台任务）
func (nc *NATCoordinator) CleanupExpiredSessions(ctx context.Context) error {
	nc.natCacheMu.Lock()
	defer nc.natCacheMu.Unlock()

	now := time.Now()
	for deviceID, result := range nc.natCache {
		if now.Sub(result.ProbeTime) > result.TTL {
			delete(nc.natCache, deviceID)
		}
	}

	return nil
}
