package service

import (
	"context"
	"fmt"
	"time"

	"github.com/edgelink/backend/internal/auth"
	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/google/uuid"
)

// DeviceService 设备服务
type DeviceService struct {
	deviceRepo        repository.DeviceRepository
	virtualNetworkRepo repository.VirtualNetworkRepository
	pskRepo           repository.PreSharedKeyRepository
	pskAuth           *auth.PSKAuthenticator
}

// NewDeviceService 创建设备服务实例
func NewDeviceService(
	deviceRepo repository.DeviceRepository,
	vnRepo repository.VirtualNetworkRepository,
	pskRepo repository.PreSharedKeyRepository,
	pskAuth *auth.PSKAuthenticator,
) *DeviceService {
	return &DeviceService{
		deviceRepo:        deviceRepo,
		virtualNetworkRepo: vnRepo,
		pskRepo:           pskRepo,
		pskAuth:           pskAuth,
	}
}

// RegisterDeviceRequest 设备注册请求
type RegisterDeviceRequest struct {
	PublicKey        string `json:"public_key"`
	Platform         string `json:"platform"`
	DeviceName       string `json:"device_name"`
	OrganizationSlug string `json:"organization_slug"`
	VirtualNetworkID string `json:"virtual_network_id"`
	PreSharedKey     string `json:"-"` // 从Header提取，不在JSON body中
}

// RegisterDeviceResponse 设备注册响应
type RegisterDeviceResponse struct {
	DeviceID         uuid.UUID              `json:"device_id"`
	VirtualIP        string                 `json:"virtual_ip"`
	VirtualNetworkID uuid.UUID              `json:"virtual_network_id"`
	CreatedAt        time.Time              `json:"created_at"`
}

// RegisterDevice 注册新设备
func (s *DeviceService) RegisterDevice(ctx context.Context, req *RegisterDeviceRequest) (*RegisterDeviceResponse, error) {
	// 1. 验证PSK
	pskHash := s.pskAuth.HashPSK(req.PreSharedKey)
	psk, err := s.pskRepo.FindByKeyHash(ctx, pskHash)
	if err != nil {
		return nil, fmt.Errorf("invalid pre-shared key: %w", err)
	}

	// 2. 检查PSK是否有效
	if !psk.IsValid() {
		return nil, fmt.Errorf("pre-shared key is expired or exhausted")
	}

	// 3. 检查公钥是否已注册
	existingDevice, err := s.deviceRepo.FindByPublicKey(ctx, req.PublicKey)
	if err == nil && existingDevice != nil {
		return nil, fmt.Errorf("device with this public key already registered")
	}

	// 4. 验证虚拟网络
	vnID, err := uuid.Parse(req.VirtualNetworkID)
	if err != nil {
		return nil, fmt.Errorf("invalid virtual network ID: %w", err)
	}

	vn, err := s.virtualNetworkRepo.FindByID(ctx, vnID)
	if err != nil {
		return nil, fmt.Errorf("virtual network not found: %w", err)
	}

	// 5. 分配虚拟IP（简化版本 - 生产环境需要IP池管理）
	// TODO: 实现完整的IP分配逻辑
	virtualIP := s.allocateVirtualIP(vn)

	// 6. 创建设备记录
	device := &domain.Device{
		ID:               uuid.New(),
		VirtualNetworkID: vnID,
		Name:             req.DeviceName,
		VirtualIP:        virtualIP,
		PublicKey:        req.PublicKey,
		Platform:         domain.Platform(req.Platform),
		NATType:          domain.NATTypeUnknown,
		Online:           false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	// 7. 更新PSK使用次数
	if err := s.pskRepo.IncrementUsedCount(ctx, psk.ID); err != nil {
		// 记录日志但不失败
		fmt.Printf("warning: failed to increment PSK used count: %v\n", err)
	}

	return &RegisterDeviceResponse{
		DeviceID:         device.ID,
		VirtualIP:        device.VirtualIP,
		VirtualNetworkID: device.VirtualNetworkID,
		CreatedAt:        device.CreatedAt,
	}, nil
}

// GetDeviceConfig 获取设备配置
func (s *DeviceService) GetDeviceConfig(ctx context.Context, deviceID uuid.UUID) (*domain.Device, error) {
	device, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	return device, nil
}

// UpdateDeviceStatus 更新设备在线状态
func (s *DeviceService) UpdateDeviceStatus(ctx context.Context, deviceID uuid.UUID, online bool) error {
	return s.deviceRepo.UpdateOnlineStatus(ctx, deviceID, online)
}

// RevokeDevice 撤销设备
func (s *DeviceService) RevokeDevice(ctx context.Context, deviceID uuid.UUID) error {
	// 1. 标记设备为离线
	if err := s.deviceRepo.UpdateOnlineStatus(ctx, deviceID, false); err != nil {
		return fmt.Errorf("failed to mark device offline: %w", err)
	}

	// 2. 删除设备记录（级联删除会处理相关数据）
	if err := s.deviceRepo.Delete(ctx, deviceID); err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

// allocateVirtualIP 分配虚拟IP（简化实现）
// TODO: 实现完整的IP池管理，支持CIDR范围内的动态分配
func (s *DeviceService) allocateVirtualIP(vn *domain.VirtualNetwork) string {
	// 临时实现：生成随机IP（仅用于MVP演示）
	// 生产环境需要：
	// 1. 解析CIDR范围
	// 2. 查询已分配IP
	// 3. 从可用池中选择未使用IP
	// 4. 处理IP回收和重用
	return "10.100.1.100" // 占位符
}
