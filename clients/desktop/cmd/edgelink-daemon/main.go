package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgelink/client/internal/config"
	"github.com/edgelink/client/internal/metrics"
	"github.com/edgelink/client/internal/wireguard"
)

const (
	defaultConfigPath = "/etc/edgelink/device.conf"
	interfaceName     = "edgelink0"
	metricsInterval   = 30 * time.Second
)

func main() {
	// 检查root权限
	if os.Geteuid() != 0 {
		log.Fatal("EdgeLink daemon must run as root (use sudo)")
	}

	// 读取命令行参数
	configPath := defaultConfigPath
	if len(os.Args) > 2 && os.Args[1] == "--config" {
		configPath = os.Args[2]
	}

	// 提示输入配置密码
	fmt.Print("Enter config password: ")
	var password string
	fmt.Scanln(&password)

	// 加载配置
	fmt.Println("Loading device configuration...")
	configStore := config.NewConfigStore(configPath, password)
	deviceConfig, err := configStore.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Device ID: %s\n", deviceConfig.DeviceID)
	fmt.Printf("Virtual IP: %s\n", deviceConfig.VirtualIP)

	// 创建WireGuard接口管理器
	interfaceManager, err := wireguard.NewInterfaceManager(interfaceName)
	if err != nil {
		log.Fatalf("Failed to create interface manager: %v", err)
	}

	// 创建指标报告器
	metricsReporter := metrics.NewReporter(
		deviceConfig.DeviceID,
		deviceConfig.ControlPlaneURL,
		metricsInterval,
	)

	// 启动守护进程
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runDaemon(ctx, interfaceManager, metricsReporter, deviceConfig); err != nil {
		log.Fatalf("Daemon failed: %v", err)
	}
}

func runDaemon(
	ctx context.Context,
	interfaceManager *wireguard.InterfaceManager,
	metricsReporter *metrics.Reporter,
	deviceConfig *config.DeviceConfig,
) error {
	// 1. 创建WireGuard接口
	fmt.Println("Creating WireGuard interface...")
	if err := interfaceManager.CreateInterface(); err != nil {
		return fmt.Errorf("failed to create interface: %w", err)
	}
	defer func() {
		fmt.Println("Cleaning up WireGuard interface...")
		interfaceManager.DeleteInterface()
	}()

	// 2. 配置虚拟IP
	fmt.Printf("Configuring virtual IP: %s\n", deviceConfig.VirtualIP)
	if err := interfaceManager.AddAddress(deviceConfig.VirtualIP + "/24"); err != nil {
		return fmt.Errorf("failed to add address: %w", err)
	}

	// 3. 启动接口
	fmt.Println("Bringing interface up...")
	if err := interfaceManager.SetInterfaceUp(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w", err)
	}

	// 4. 应用WireGuard配置
	// TODO: 从控制平面获取对等配置并生成wg配置文件
	// configFile := "/tmp/edgelink-wg.conf"
	// if err := interfaceManager.ApplyConfig(configFile); err != nil {
	//     return fmt.Errorf("failed to apply config: %w", err)
	// }

	// 5. 启动指标上报
	fmt.Println("Starting metrics reporter...")
	go metricsReporter.Start()
	defer metricsReporter.Stop()

	// 6. 上报初始心跳
	if err := metricsReporter.ReportHeartbeat(); err != nil {
		log.Printf("Warning: Failed to report initial heartbeat: %v", err)
	}

	// 7. 监控WireGuard连接
	fmt.Println("EdgeLink daemon is running...")
	fmt.Println("Press Ctrl+C to stop")

	// 启动监控循环
	go monitorLoop(ctx, interfaceManager, metricsReporter)

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		fmt.Println("\nReceived shutdown signal")
	case <-ctx.Done():
		fmt.Println("\nContext cancelled")
	}

	return nil
}

func monitorLoop(
	ctx context.Context,
	interfaceManager *wireguard.InterfaceManager,
	metricsReporter *metrics.Reporter,
) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查接口状态
			up, err := interfaceManager.IsInterfaceUp()
			if err != nil {
				log.Printf("Error checking interface: %v", err)
				continue
			}

			if !up {
				log.Println("Interface is down, attempting to bring it up...")
				if err := interfaceManager.SetInterfaceUp(); err != nil {
					log.Printf("Failed to bring interface up: %v", err)
				}
			}

			// 获取接口统计信息
			stats, err := interfaceManager.GetInterfaceStats()
			if err != nil {
				log.Printf("Error getting stats: %v", err)
				continue
			}

			// TODO: 解析统计信息并上报
			_ = stats

		case <-ctx.Done():
			return
		}
	}
}
