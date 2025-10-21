package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/scrypt"
)

// DeviceConfig 设备配置
type DeviceConfig struct {
	DeviceID         string   `json:"device_id"`
	DeviceName       string   `json:"device_name"`
	PrivateKey       string   `json:"private_key"`
	PublicKey        string   `json:"public_key"`
	VirtualIP        string   `json:"virtual_ip"`
	VirtualNetworkID string   `json:"virtual_network_id"`
	ControlPlaneURL  string   `json:"control_plane_url"`
	ListenPort       int      `json:"listen_port"`
	DNS              []string `json:"dns,omitempty"`
}

// ConfigStore 配置存储（加密）
type ConfigStore struct {
	configPath string
	password   string
}

// NewConfigStore 创建配置存储
func NewConfigStore(configPath, password string) *ConfigStore {
	return &ConfigStore{
		configPath: configPath,
		password:   password,
	}
}

// Save 保存配置（加密）
func (cs *ConfigStore) Save(config *DeviceConfig) error {
	// 序列化配置
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 加密数据
	encrypted, err := cs.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(cs.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 写入文件（权限600，仅所有者可读写）
	if err := os.WriteFile(cs.configPath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Load 加载配置（解密）
func (cs *ConfigStore) Load() (*DeviceConfig, error) {
	// 读取加密文件
	encrypted, err := os.ReadFile(cs.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解密数据
	data, err := cs.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}

	// 反序列化配置
	var config DeviceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// Exists 检查配置文件是否存在
func (cs *ConfigStore) Exists() bool {
	_, err := os.Stat(cs.configPath)
	return err == nil
}

// Delete 删除配置文件
func (cs *ConfigStore) Delete() error {
	if err := os.Remove(cs.configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file: %w", err)
	}
	return nil
}

// encrypt 使用AES-256-GCM加密数据
func (cs *ConfigStore) encrypt(plaintext []byte) ([]byte, error) {
	// 使用scrypt派生密钥（更安全的密钥派生函数）
	key, salt, err := cs.deriveKey()
	if err != nil {
		return nil, err
	}

	// 创建AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用GCM模式（提供认证加密）
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	// 将salt和密文拼接（salt在前32字节）
	result := append(salt, ciphertext...)

	return result, nil
}

// decrypt 使用AES-256-GCM解密数据
func (cs *ConfigStore) decrypt(ciphertext []byte) ([]byte, error) {
	// 提取salt（前32字节）
	if len(ciphertext) < 32 {
		return nil, fmt.Errorf("ciphertext too short")
	}

	salt := ciphertext[:32]
	encrypted := ciphertext[32:]

	// 使用相同的salt派生密钥
	key, err := cs.deriveKeyWithSalt(salt)
	if err != nil {
		return nil, err
	}

	// 创建AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用GCM模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 提取nonce
	nonceSize := aesGCM.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short for nonce")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	// 解密数据
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// deriveKey 派生加密密钥
func (cs *ConfigStore) deriveKey() ([]byte, []byte, error) {
	// 生成随机salt
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key, err := cs.deriveKeyWithSalt(salt)
	if err != nil {
		return nil, nil, err
	}

	return key, salt, nil
}

// deriveKeyWithSalt 使用指定salt派生密钥
func (cs *ConfigStore) deriveKeyWithSalt(salt []byte) ([]byte, error) {
	// scrypt参数：
	// N=32768 (CPU/内存成本参数)
	// r=8 (块大小)
	// p=1 (并行参数)
	// keyLen=32 (256位密钥)
	key, err := scrypt.Key([]byte(cs.password), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}
