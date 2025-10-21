package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClientConfig 客户端配置
type ClientConfig struct {
	// 服务器配置
	ServerURL      string `json:"server_url"`
	PreSharedKey   string `json:"pre_shared_key"`

	// 设备信息
	DeviceID       string `json:"device_id,omitempty"`
	DeviceName     string `json:"device_name"`
	PublicKey      string `json:"public_key,omitempty"`
	PrivateKey     string `json:"private_key,omitempty"`

	// 网络配置
	VirtualIP      string `json:"virtual_ip,omitempty"`
	VirtualSubnet  string `json:"virtual_subnet,omitempty"`

	// WireGuard配置
	ListenPort     int    `json:"listen_port"`
	MTU            int    `json:"mtu"`

	// 自动启动
	AutoStart      bool   `json:"auto_start"`
	AutoReconnect  bool   `json:"auto_reconnect"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *ClientConfig {
	hostname, _ := os.Hostname()

	return &ClientConfig{
		ServerURL:     "https://api.edgelink.example.com",
		DeviceName:    hostname,
		ListenPort:    51820,
		MTU:           1420,
		AutoStart:     false,
		AutoReconnect: true,
	}
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// GetConfigDir 获取配置目录
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".edgelink")

	// 确保目录存在
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return configDir, nil
}

// Load 加载配置
func Load() (*ClientConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// 如果配置文件不存在,返回默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// Save 保存配置
func (c *ClientConfig) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate 验证配置
func (c *ClientConfig) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("server URL is required")
	}

	if c.PreSharedKey == "" {
		return fmt.Errorf("pre-shared key is required")
	}

	if c.DeviceName == "" {
		return fmt.Errorf("device name is required")
	}

	return nil
}
