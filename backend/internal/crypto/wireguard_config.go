package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/curve25519"
)

// WireGuardKeyPair WireGuard密钥对（Curve25519）
type WireGuardKeyPair struct {
	PrivateKey [32]byte
	PublicKey  [32]byte
}

// WireGuardPeerConfig WireGuard对等设备配置
type WireGuardPeerConfig struct {
	PublicKey           string   `json:"public_key"`
	AllowedIPs          []string `json:"allowed_ips"`
	Endpoint            string   `json:"endpoint,omitempty"`
	PersistentKeepalive int      `json:"persistent_keepalive,omitempty"`
}

// WireGuardInterfaceConfig WireGuard接口配置
type WireGuardInterfaceConfig struct {
	PrivateKey string   `json:"private_key"`
	Address    string   `json:"address"` // CIDR格式，如 10.100.1.42/16
	ListenPort int      `json:"listen_port"`
	DNS        []string `json:"dns,omitempty"`
}

// WireGuardConfig 完整的WireGuard配置
type WireGuardConfig struct {
	Interface WireGuardInterfaceConfig `json:"interface"`
	Peers     []WireGuardPeerConfig    `json:"peers"`
}

// GenerateWireGuardKeyPair 生成WireGuard密钥对（Curve25519）
func GenerateWireGuardKeyPair() (*WireGuardKeyPair, error) {
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// 修正私钥格式（WireGuard规范）
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// 计算公钥
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return &WireGuardKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// PrivateKeyBase64 返回Base64编码的私钥
func (kp *WireGuardKeyPair) PrivateKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PrivateKey[:])
}

// PublicKeyBase64 返回Base64编码的公钥
func (kp *WireGuardKeyPair) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PublicKey[:])
}

// ToWGQuickFormat 将配置转换为wg-quick格式的配置文件
func (wgc *WireGuardConfig) ToWGQuickFormat() string {
	var sb strings.Builder

	// [Interface] section
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", wgc.Interface.PrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", wgc.Interface.Address))
	if wgc.Interface.ListenPort > 0 {
		sb.WriteString(fmt.Sprintf("ListenPort = %d\n", wgc.Interface.ListenPort))
	}
	if len(wgc.Interface.DNS) > 0 {
		sb.WriteString(fmt.Sprintf("DNS = %s\n", strings.Join(wgc.Interface.DNS, ", ")))
	}
	sb.WriteString("\n")

	// [Peer] sections
	for _, peer := range wgc.Peers {
		sb.WriteString("[Peer]\n")
		sb.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(peer.AllowedIPs, ", ")))
		if peer.Endpoint != "" {
			sb.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint))
		}
		if peer.PersistentKeepalive > 0 {
			sb.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepalive))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ParseWireGuardPublicKey 从Base64字符串解析WireGuard公钥
func ParseWireGuardPublicKey(pub string) ([32]byte, error) {
	var publicKey [32]byte
	decoded, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		return publicKey, fmt.Errorf("invalid public key: %w", err)
	}

	if len(decoded) != 32 {
		return publicKey, fmt.Errorf("invalid public key length: expected 32, got %d", len(decoded))
	}

	copy(publicKey[:], decoded)
	return publicKey, nil
}

// ValidateWireGuardConfig 验证WireGuard配置的完整性
func ValidateWireGuardConfig(cfg *WireGuardConfig) error {
	if cfg.Interface.PrivateKey == "" {
		return fmt.Errorf("interface private key is required")
	}
	if cfg.Interface.Address == "" {
		return fmt.Errorf("interface address is required")
	}

	// 验证私钥格式
	if _, err := base64.StdEncoding.DecodeString(cfg.Interface.PrivateKey); err != nil {
		return fmt.Errorf("invalid private key format: %w", err)
	}

	// 验证对等设备配置
	for i, peer := range cfg.Peers {
		if peer.PublicKey == "" {
			return fmt.Errorf("peer %d: public key is required", i)
		}
		if len(peer.AllowedIPs) == 0 {
			return fmt.Errorf("peer %d: at least one allowed IP is required", i)
		}
		if _, err := base64.StdEncoding.DecodeString(peer.PublicKey); err != nil {
			return fmt.Errorf("peer %d: invalid public key format: %w", i, err)
		}
	}

	return nil
}
