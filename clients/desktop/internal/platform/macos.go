// +build darwin

package platform

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Platform macOS平台实现
type Platform struct{}

// NewPlatform 创建macOS平台实例
func NewPlatform() *Platform {
	return &Platform{}
}

// GetName 获取平台名称
func (p *Platform) GetName() string {
	return "macos"
}

// GetArch 获取架构
func (p *Platform) GetArch() string {
	return runtime.GOARCH
}

// CreateTunInterface 创建TUN接口
func (p *Platform) CreateTunInterface(name string) error {
	// macOS上使用utun接口
	fmt.Printf("Creating TUN interface '%s' on macOS\n", name)

	// macOS会自动创建utun接口,通常是/dev/utun0, /dev/utun1等
	// wireguard-go会自动处理

	return nil
}

// ConfigureInterface 配置网络接口
func (p *Platform) ConfigureInterface(name, ip, subnet string, mtu int) error {
	fmt.Printf("Configuring interface '%s' with IP %s/%s\n", name, ip, subnet)

	// 设置IP地址
	cmd := exec.Command("ifconfig", name, ip, ip, "netmask", subnet, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure interface: %w", err)
	}

	// 设置MTU
	cmd = exec.Command("ifconfig", name, "mtu", fmt.Sprintf("%d", mtu))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	return nil
}

// AddRoute 添加路由
func (p *Platform) AddRoute(destination, gateway, interfaceName string) error {
	// 使用route add命令
	cmd := exec.Command("route", "add", "-net", destination, "-interface", interfaceName)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}

	return nil
}

// DeleteRoute 删除路由
func (p *Platform) DeleteRoute(destination string) error {
	cmd := exec.Command("route", "delete", "-net", destination)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	return nil
}

// EnableIPForwarding 启用IP转发
func (p *Platform) EnableIPForwarding() error {
	fmt.Println("Enabling IP forwarding on macOS")

	cmd := exec.Command("sysctl", "-w", "net.inet.ip.forwarding=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	return nil
}

// CheckPrivileges 检查是否具有root权限
func (p *Platform) CheckPrivileges() error {
	// macOS需要root权限来创建TUN接口和修改路由
	cmd := exec.Command("id", "-u")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check privileges: %w", err)
	}

	if string(output) != "0\n" {
		return fmt.Errorf("root privileges required - please run with sudo")
	}

	return nil
}

// StartWireGuard 启动WireGuard
func (p *Platform) StartWireGuard(configPath string) error {
	fmt.Printf("Starting WireGuard with config: %s\n", configPath)

	// macOS上使用wireguard-go
	// 实际实现:
	// cmd := exec.Command("wireguard-go", "utun")
	// return cmd.Start()

	return nil
}

// StopWireGuard 停止WireGuard
func (p *Platform) StopWireGuard(interfaceName string) error {
	fmt.Printf("Stopping WireGuard interface: %s\n", interfaceName)

	// 实际实现:
	// 需要kill wireguard-go进程

	return nil
}

// GetInterfaceStats 获取接口统计信息
func (p *Platform) GetInterfaceStats(interfaceName string) (bytesSent, bytesReceived int64, err error) {
	// 使用netstat或ifconfig查询接口统计
	cmd := exec.Command("netstat", "-I", interfaceName, "-b")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get interface stats: %w", err)
	}

	// 解析netstat输出
	// 简化实现返回示例数据
	_ = output
	return 0, 0, nil
}
