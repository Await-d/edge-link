package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/edgelink/client/internal/config"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register device with EdgeLink control plane",
	Long:  `Register a new device using a pre-shared key and receive virtual network configuration.`,
	RunE:  runRegister,
}

var (
	controlPlaneURL  string
	preSharedKey     string
	deviceName       string
	organizationSlug string
	virtualNetworkID string
	configPath       string
	configPassword   string
)

func init() {
	registerCmd.Flags().StringVarP(&controlPlaneURL, "control-plane", "c", "https://control.edgelink.example.com", "Control plane URL")
	registerCmd.Flags().StringVarP(&preSharedKey, "psk", "k", "", "Pre-shared key (required)")
	registerCmd.Flags().StringVarP(&deviceName, "name", "n", "", "Device name (default: hostname)")
	registerCmd.Flags().StringVarP(&organizationSlug, "org", "o", "", "Organization slug (required)")
	registerCmd.Flags().StringVarP(&virtualNetworkID, "network", "N", "", "Virtual network ID (required)")
	registerCmd.Flags().StringVarP(&configPath, "config", "f", "/etc/edgelink/device.conf", "Config file path")
	registerCmd.Flags().StringVarP(&configPassword, "password", "p", "", "Config encryption password")

	registerCmd.MarkFlagRequired("psk")
	registerCmd.MarkFlagRequired("org")
	registerCmd.MarkFlagRequired("network")
}

func runRegister(cmd *cobra.Command, args []string) error {
	// 1. 生成设备密钥对
	fmt.Println("Generating device keypair...")
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}

	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)
	privateKeyB64 := base64.StdEncoding.EncodeToString(privateKey)

	// 2. 获取设备名称
	if deviceName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get hostname: %w", err)
		}
		deviceName = hostname
	}

	// 3. 构建注册请求
	registerReq := map[string]interface{}{
		"public_key":         publicKeyB64,
		"platform":           runtime.GOOS,
		"device_name":        deviceName,
		"organization_slug":  organizationSlug,
		"virtual_network_id": virtualNetworkID,
	}

	reqData, err := json.Marshal(registerReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 4. 发送注册请求
	fmt.Println("Registering device with control plane...")
	url := fmt.Sprintf("%s/api/v1/device/register", controlPlaneURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pre-Shared-Key", preSharedKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 5. 解析响应
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var registerResp struct {
		DeviceID         string `json:"device_id"`
		VirtualIP        string `json:"virtual_ip"`
		VirtualNetworkID string `json:"virtual_network_id"`
		CreatedAt        string `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("✅ Device registered successfully!\n")
	fmt.Printf("   Device ID: %s\n", registerResp.DeviceID)
	fmt.Printf("   Virtual IP: %s\n", registerResp.VirtualIP)

	// 6. 保存配置
	if configPassword == "" {
		// 提示用户输入密码
		fmt.Print("Enter config encryption password: ")
		fmt.Scanln(&configPassword)
	}

	deviceConfig := &config.DeviceConfig{
		DeviceID:         registerResp.DeviceID,
		DeviceName:       deviceName,
		PrivateKey:       privateKeyB64,
		PublicKey:        publicKeyB64,
		VirtualIP:        registerResp.VirtualIP,
		VirtualNetworkID: registerResp.VirtualNetworkID,
		ControlPlaneURL:  controlPlaneURL,
		ListenPort:       51820,
	}

	configStore := config.NewConfigStore(configPath, configPassword)
	if err := configStore.Save(deviceConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✅ Configuration saved to: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Start the EdgeLink daemon: sudo edgelink-daemon start")
	fmt.Println("  2. Check connection status: edgelink-cli status")

	return nil
}
