package wireguard

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// InterfaceManager WireGuard接口管理器
type InterfaceManager struct {
	interfaceName string
	useUserspace  bool // true: wireguard-go, false: 内核模块
}

// NewInterfaceManager 创建接口管理器
func NewInterfaceManager(interfaceName string) (*InterfaceManager, error) {
	im := &InterfaceManager{
		interfaceName: interfaceName,
	}

	// 检测WireGuard实现方式
	if err := im.detectWireGuardImplementation(); err != nil {
		return nil, fmt.Errorf("failed to detect WireGuard implementation: %w", err)
	}

	return im, nil
}

// detectWireGuardImplementation 检测WireGuard实现方式
func (im *InterfaceManager) detectWireGuardImplementation() error {
	// 优先使用内核模块（性能更好）
	if hasKernelModule() {
		im.useUserspace = false
		return nil
	}

	// 回退到用户空间实现
	if hasWireGuardGo() {
		im.useUserspace = true
		return nil
	}

	return fmt.Errorf("neither kernel WireGuard module nor wireguard-go found")
}

// hasKernelModule 检查内核是否加载了WireGuard模块
func hasKernelModule() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// 检查/sys/module/wireguard是否存在
	if _, err := os.Stat("/sys/module/wireguard"); err == nil {
		return true
	}

	// 尝试加载模块
	cmd := exec.Command("modprobe", "wireguard")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// hasWireGuardGo 检查wireguard-go是否可用
func hasWireGuardGo() bool {
	cmd := exec.Command("wireguard-go", "--version")
	err := cmd.Run()
	return err == nil
}

// CreateInterface 创建WireGuard接口
func (im *InterfaceManager) CreateInterface() error {
	if im.useUserspace {
		return im.createUserSpaceInterface()
	}
	return im.createKernelInterface()
}

// createKernelInterface 创建内核模块接口
func (im *InterfaceManager) createKernelInterface() error {
	// 使用ip link创建WireGuard接口
	cmd := exec.Command("ip", "link", "add", "dev", im.interfaceName, "type", "wireguard")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create kernel interface: %w, output: %s", err, string(output))
	}

	return nil
}

// createUserSpaceInterface 创建用户空间接口
func (im *InterfaceManager) createUserSpaceInterface() error {
	// wireguard-go会自动创建TUN接口
	cmd := exec.Command("wireguard-go", im.interfaceName)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start wireguard-go: %w", err)
	}

	// TODO: 保存进程PID用于后续管理
	return nil
}

// DeleteInterface 删除WireGuard接口
func (im *InterfaceManager) DeleteInterface() error {
	cmd := exec.Command("ip", "link", "delete", "dev", im.interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果接口不存在，忽略错误
		if strings.Contains(string(output), "Cannot find device") {
			return nil
		}
		return fmt.Errorf("failed to delete interface: %w, output: %s", err, string(output))
	}

	return nil
}

// ApplyConfig 应用WireGuard配置
func (im *InterfaceManager) ApplyConfig(configPath string) error {
	if im.useUserspace {
		return im.applyConfigUserSpace(configPath)
	}
	return im.applyConfigKernel(configPath)
}

// applyConfigKernel 应用内核模块配置
func (im *InterfaceManager) applyConfigKernel(configPath string) error {
	cmd := exec.Command("wg", "setconf", im.interfaceName, configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply kernel config: %w, output: %s", err, string(output))
	}

	return nil
}

// applyConfigUserSpace 应用用户空间配置
func (im *InterfaceManager) applyConfigUserSpace(configPath string) error {
	// wireguard-go使用wg工具的UAPI接口
	cmd := exec.Command("wg", "setconf", im.interfaceName, configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply userspace config: %w, output: %s", err, string(output))
	}

	return nil
}

// SetInterfaceUp 启动接口
func (im *InterfaceManager) SetInterfaceUp() error {
	cmd := exec.Command("ip", "link", "set", "up", "dev", im.interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bring interface up: %w, output: %s", err, string(output))
	}

	return nil
}

// SetInterfaceDown 关闭接口
func (im *InterfaceManager) SetInterfaceDown() error {
	cmd := exec.Command("ip", "link", "set", "down", "dev", im.interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bring interface down: %w, output: %s", err, string(output))
	}

	return nil
}

// AddAddress 为接口添加IP地址
func (im *InterfaceManager) AddAddress(cidr string) error {
	cmd := exec.Command("ip", "address", "add", "dev", im.interfaceName, cidr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add address: %w, output: %s", err, string(output))
	}

	return nil
}

// GetInterfaceStats 获取接口统计信息
func (im *InterfaceManager) GetInterfaceStats() (map[string]interface{}, error) {
	cmd := exec.Command("wg", "show", im.interfaceName, "dump")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface stats: %w", err)
	}

	// 解析wg dump输出
	stats := make(map[string]interface{})
	stats["raw_output"] = string(output)
	// TODO: 解析详细的对等设备统计信息

	return stats, nil
}

// IsInterfaceUp 检查接口是否已启动
func (im *InterfaceManager) IsInterfaceUp() (bool, error) {
	cmd := exec.Command("ip", "link", "show", im.interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 接口不存在
		if strings.Contains(string(output), "does not exist") {
			return false, nil
		}
		return false, err
	}

	// 检查输出中是否包含"UP"
	return strings.Contains(string(output), "UP"), nil
}
