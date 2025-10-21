package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgelink/clients/desktop/internal/api"
	"github.com/edgelink/clients/desktop/internal/config"
	"github.com/edgelink/clients/desktop/internal/platform"
)

const (
	version = "0.1.0"
)

func main() {
	// 命令行参数
	var (
		configFile    = flag.String("config", "", "配置文件路径")
		serverURL     = flag.String("server", "", "EdgeLink服务器URL")
		preSharedKey  = flag.String("key", "", "预共享密钥")
		deviceName    = flag.String("name", "", "设备名称")
		showVersion   = flag.Bool("version", false, "显示版本信息")
		register      = flag.Bool("register", false, "注册设备")
		connect       = flag.Bool("connect", false, "连接到网络")
		disconnect    = flag.Bool("disconnect", false, "断开连接")
		status        = flag.Bool("status", false, "显示连接状态")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("EdgeLink Lite Client v%s\n", version)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := loadConfig(*configFile, *serverURL, *preSharedKey, *deviceName)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建平台实例
	plat := platform.NewPlatform()

	// 创建API客户端
	apiClient := api.NewClient(cfg.ServerURL)

	// 执行操作
	if *register {
		if err := registerDevice(apiClient, cfg, plat); err != nil {
			log.Fatalf("Registration failed: %v", err)
		}
		fmt.Println("Device registered successfully")
		return
	}

	if *connect {
		if err := connectToNetwork(apiClient, cfg, plat); err != nil {
			log.Fatalf("Connection failed: %v", err)
		}
		fmt.Println("Connected to EdgeLink network")

		// 等待中断信号
		waitForShutdown(plat, cfg)
		return
	}

	if *disconnect {
		if err := disconnectFromNetwork(plat, cfg); err != nil {
			log.Fatalf("Disconnection failed: %v", err)
		}
		fmt.Println("Disconnected from EdgeLink network")
		return
	}

	if *status {
		showStatus(cfg, plat)
		return
	}

	// 默认显示帮助
	flag.Usage()
}

// loadConfig 加载配置
func loadConfig(configFile, serverURL, preSharedKey, deviceName string) (*config.ClientConfig, error) {
	var cfg *config.ClientConfig
	var err error

	if configFile != "" {
		// 从指定文件加载
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	} else {
		// 从默认位置加载
		cfg, err = config.Load()
		if err != nil {
			return nil, err
		}
	}

	// 命令行参数覆盖配置文件
	if serverURL != "" {
		cfg.ServerURL = serverURL
	}
	if preSharedKey != "" {
		cfg.PreSharedKey = preSharedKey
	}
	if deviceName != "" {
		cfg.DeviceName = deviceName
	}

	return cfg, nil
}

// registerDevice 注册设备
func registerDevice(apiClient *api.Client, cfg *config.ClientConfig, plat *platform.Platform) error {
	// 生成WireGuard密钥对
	// 简化实现:使用占位符
	publicKey := "generated_public_key"
	privateKey := "generated_private_key"

	req := &api.RegisterDeviceRequest{
		PreSharedKey: cfg.PreSharedKey,
		Name:         cfg.DeviceName,
		Platform:     plat.GetName(),
		PublicKey:    publicKey,
	}

	resp, err := apiClient.RegisterDevice(req)
	if err != nil {
		return err
	}

	// 保存配置
	cfg.DeviceID = resp.DeviceID
	cfg.VirtualIP = resp.VirtualIP
	cfg.VirtualSubnet = resp.VirtualSubnet
	cfg.PublicKey = publicKey
	cfg.PrivateKey = privateKey

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// connectToNetwork 连接到网络
func connectToNetwork(apiClient *api.Client, cfg *config.ClientConfig, plat *platform.Platform) error {
	// 检查权限
	if err := plat.CheckPrivileges(); err != nil {
		return err
	}

	// 创建TUN接口
	interfaceName := "edgelink0"
	if err := plat.CreateTunInterface(interfaceName); err != nil {
		return fmt.Errorf("failed to create TUN interface: %w", err)
	}

	// 配置接口
	if err := plat.ConfigureInterface(interfaceName, cfg.VirtualIP, cfg.VirtualSubnet, cfg.MTU); err != nil {
		return fmt.Errorf("failed to configure interface: %w", err)
	}

	// 获取最新配置(对等列表)
	deviceConfig, err := apiClient.GetDeviceConfig(cfg.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device config: %w", err)
	}

	// 生成WireGuard配置文件
	// 简化实现:实际应该生成完整的wg配置

	// 启动WireGuard
	configPath := "/tmp/edgelink.conf"
	if err := plat.StartWireGuard(configPath); err != nil {
		return fmt.Errorf("failed to start WireGuard: %w", err)
	}

	// 启动指标上报
	go reportMetrics(apiClient, cfg, plat)

	_ = deviceConfig // 使用配置

	return nil
}

// disconnectFromNetwork 断开网络连接
func disconnectFromNetwork(plat *platform.Platform, cfg *config.ClientConfig) error {
	interfaceName := "edgelink0"

	if err := plat.StopWireGuard(interfaceName); err != nil {
		return fmt.Errorf("failed to stop WireGuard: %w", err)
	}

	return nil
}

// showStatus 显示连接状态
func showStatus(cfg *config.ClientConfig, plat *platform.Platform) {
	fmt.Println("EdgeLink Client Status")
	fmt.Println("=====================")
	fmt.Printf("Server: %s\n", cfg.ServerURL)
	fmt.Printf("Device ID: %s\n", cfg.DeviceID)
	fmt.Printf("Device Name: %s\n", cfg.DeviceName)
	fmt.Printf("Virtual IP: %s\n", cfg.VirtualIP)
	fmt.Printf("Platform: %s/%s\n", plat.GetName(), plat.GetArch())

	// 获取接口统计
	bytesSent, bytesReceived, _ := plat.GetInterfaceStats("edgelink0")
	fmt.Printf("Bytes Sent: %d\n", bytesSent)
	fmt.Printf("Bytes Received: %d\n", bytesReceived)
}

// reportMetrics 定期上报指标
func reportMetrics(apiClient *api.Client, cfg *config.ClientConfig, plat *platform.Platform) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		bytesSent, bytesReceived, err := plat.GetInterfaceStats("edgelink0")
		if err != nil {
			log.Printf("Failed to get interface stats: %v", err)
			continue
		}

		metrics := &api.MetricsRequest{
			BytesSent:     bytesSent,
			BytesReceived: bytesReceived,
		}

		if err := apiClient.SubmitMetrics(cfg.DeviceID, metrics); err != nil {
			log.Printf("Failed to submit metrics: %v", err)
		}
	}
}

// waitForShutdown 等待关闭信号
func waitForShutdown(plat *platform.Platform, cfg *config.ClientConfig) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh

	fmt.Println("\nShutting down...")

	// 断开连接
	if err := disconnectFromNetwork(plat, cfg); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
