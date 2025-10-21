package stun

import (
	"fmt"
	"net"
	"time"
)

// NATType NAT类型枚举
type NATType int

const (
	NATTypeUnknown NATType = iota
	NATTypeNone
	NATTypeFullCone
	NATTypeRestrictedCone
	NATTypePortRestrictedCone
	NATTypeSymmetric
)

// String 返回NAT类型的字符串表示
func (t NATType) String() string {
	switch t {
	case NATTypeNone:
		return "None"
	case NATTypeFullCone:
		return "Full Cone"
	case NATTypeRestrictedCone:
		return "Restricted Cone"
	case NATTypePortRestrictedCone:
		return "Port Restricted Cone"
	case NATTypeSymmetric:
		return "Symmetric"
	default:
		return "Unknown"
	}
}

// STUNClient STUN客户端
type STUNClient struct {
	primaryServer   string
	secondaryServer string
	timeout         time.Duration
}

// NewSTUNClient 创建STUN客户端
func NewSTUNClient(primaryServer, secondaryServer string) *STUNClient {
	return &STUNClient{
		primaryServer:   primaryServer,
		secondaryServer: secondaryServer,
		timeout:         5 * time.Second,
	}
}

// ProbeNATType 探测NAT类型
func (c *STUNClient) ProbeNATType() (NATType, string, error) {
	// TODO: 实现完整的RFC 5780 STUN NAT类型检测算法
	// 当前为简化实现，生产环境需要：
	// 1. 发送STUN Binding Request到主STUN服务器
	// 2. 解析STUN Binding Response，获取MAPPED-ADDRESS和XOR-MAPPED-ADDRESS
	// 3. 比较本地地址和映射地址，判断是否存在NAT
	// 4. 发送到不同IP/端口的STUN服务器，判断NAT类型
	// 5. 根据RFC 5780的决策树确定NAT类型

	// 临时实现：仅检测是否存在NAT
	localAddr, err := getLocalAddress()
	if err != nil {
		return NATTypeUnknown, "", fmt.Errorf("failed to get local address: %w", err)
	}

	mappedAddr, err := c.getMappedAddress()
	if err != nil {
		return NATTypeUnknown, "", fmt.Errorf("failed to get mapped address: %w", err)
	}

	// 如果本地地址和映射地址相同，说明没有NAT
	if localAddr == mappedAddr {
		return NATTypeNone, mappedAddr, nil
	}

	// 默认返回未知类型（需要完整的STUN探测）
	return NATTypeUnknown, mappedAddr, nil
}

// getMappedAddress 获取公网映射地址
func (c *STUNClient) getMappedAddress() (string, error) {
	// TODO: 实现完整的STUN Binding Request/Response
	// 当前为占位符实现

	// 连接到STUN服务器
	conn, err := net.DialTimeout("udp", c.primaryServer, c.timeout)
	if err != nil {
		return "", fmt.Errorf("failed to connect to STUN server: %w", err)
	}
	defer conn.Close()

	// 设置读写超时
	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return "", fmt.Errorf("failed to set deadline: %w", err)
	}

	// TODO: 构造STUN Binding Request消息
	// STUN消息格式：
	// 0                   1                   2                   3
	// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |0 0|     STUN Message Type     |         Message Length        |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                         Magic Cookie                          |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                                                               |
	// |                     Transaction ID (96 bits)                  |
	// |                                                               |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

	// 临时：返回远程地址作为映射地址
	remoteAddr := conn.RemoteAddr().(*net.UDPAddr)
	return remoteAddr.IP.String(), nil
}

// getLocalAddress 获取本地IP地址
func getLocalAddress() (string, error) {
	// 通过连接外部地址来获取本地使用的IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// GetPublicEndpoint 获取公网端点（IP:Port）
func (c *STUNClient) GetPublicEndpoint(localPort int) (string, error) {
	// TODO: 实现完整的STUN Binding请求
	// 当前返回映射地址+本地端口作为占位符

	mappedIP, err := c.getMappedAddress()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", mappedIP, localPort), nil
}

// TestReachability 测试到对等设备的可达性
func (c *STUNClient) TestReachability(peerEndpoint string) (bool, time.Duration, error) {
	start := time.Now()

	// 尝试连接对等设备端点
	conn, err := net.DialTimeout("udp", peerEndpoint, c.timeout)
	if err != nil {
		return false, 0, err
	}
	defer conn.Close()

	// 发送探测包
	testMessage := []byte("PING")
	if _, err := conn.Write(testMessage); err != nil {
		return false, 0, err
	}

	// 等待响应
	conn.SetReadDeadline(time.Now().Add(c.timeout))
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, 0, err
	}

	latency := time.Since(start)

	// 验证响应
	if string(buffer[:n]) == "PONG" {
		return true, latency, nil
	}

	return false, latency, fmt.Errorf("unexpected response")
}
