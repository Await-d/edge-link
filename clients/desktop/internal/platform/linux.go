//go:build linux
// +build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// TUNInterface Linux TUN接口管理
type TUNInterface struct {
	name string
	fd   int
}

// CreateTUNInterface 创建TUN接口（需要root权限）
func CreateTUNInterface(name string) (*TUNInterface, error) {
	// 检查root权限
	if os.Geteuid() != 0 {
		return nil, fmt.Errorf("creating TUN interface requires root privileges")
	}

	// 使用ip tuntap命令创建TUN接口
	cmd := exec.Command("ip", "tuntap", "add", "dev", name, "mode", "tun")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %w, output: %s", err, string(output))
	}

	return &TUNInterface{
		name: name,
	}, nil
}

// Delete 删除TUN接口
func (t *TUNInterface) Delete() error {
	cmd := exec.Command("ip", "tuntap", "del", "dev", t.name, "mode", "tun")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete TUN interface: %w, output: %s", err, string(output))
	}

	return nil
}

// SetMTU 设置MTU
func (t *TUNInterface) SetMTU(mtu int) error {
	cmd := exec.Command("ip", "link", "set", "dev", t.name, "mtu", fmt.Sprintf("%d", mtu))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set MTU: %w, output: %s", err, string(output))
	}

	return nil
}

// EnableIPv4Forwarding 启用IPv4转发
func EnableIPv4Forwarding() error {
	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644); err != nil {
		return fmt.Errorf("failed to enable IPv4 forwarding: %w", err)
	}

	return nil
}

// AddRoute 添加路由
func AddRoute(destination, gateway, interfaceName string) error {
	cmd := exec.Command("ip", "route", "add", destination, "via", gateway, "dev", interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add route: %w, output: %s", err, string(output))
	}

	return nil
}

// DeleteRoute 删除路由
func DeleteRoute(destination string) error {
	cmd := exec.Command("ip", "route", "del", destination)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete route: %w, output: %s", err, string(output))
	}

	return nil
}

// SetIPTablesNAT 配置iptables NAT（用于流量转发）
func SetIPTablesNAT(interfaceName, sourceIP string) error {
	// 添加MASQUERADE规则
	cmd := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-s", sourceIP, "-o", interfaceName, "-j", "MASQUERADE")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set iptables NAT: %w, output: %s", err, string(output))
	}

	return nil
}

// RemoveIPTablesNAT 移除iptables NAT规则
func RemoveIPTablesNAT(interfaceName, sourceIP string) error {
	cmd := exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING",
		"-s", sourceIP, "-o", interfaceName, "-j", "MASQUERADE")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove iptables NAT: %w, output: %s", err, string(output))
	}

	return nil
}

// GetSystemInfo 获取系统信息
func GetSystemInfo() (map[string]string, error) {
	info := make(map[string]string)

	// 内核版本
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err == nil {
		info["kernel"] = string(uname.Release[:])
	}

	// 检查WireGuard内核模块
	if _, err := os.Stat("/sys/module/wireguard"); err == nil {
		info["wireguard_kernel"] = "available"
	} else {
		info["wireguard_kernel"] = "not_available"
	}

	// 检查wireguard-go
	if _, err := exec.LookPath("wireguard-go"); err == nil {
		info["wireguard_userspace"] = "available"
	} else {
		info["wireguard_userspace"] = "not_available"
	}

	return info, nil
}

// CheckRootPrivilege 检查是否具有root权限
func CheckRootPrivilege() bool {
	return os.Geteuid() == 0
}

// SetInterfaceOwner 设置接口所有者
func SetInterfaceOwner(interfaceName string, uid, gid int) error {
	// 通过sysfs设置接口所有者
	ownerPath := fmt.Sprintf("/sys/class/net/%s/owner", interfaceName)
	groupPath := fmt.Sprintf("/sys/class/net/%s/group", interfaceName)

	if err := os.WriteFile(ownerPath, []byte(fmt.Sprintf("%d", uid)), 0644); err != nil {
		return fmt.Errorf("failed to set owner: %w", err)
	}

	if err := os.WriteFile(groupPath, []byte(fmt.Sprintf("%d", gid)), 0644); err != nil {
		return fmt.Errorf("failed to set group: %w", err)
	}

	return nil
}
