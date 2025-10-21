package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "edgelink-cli",
	Short: "EdgeLink desktop client CLI",
	Long:  `Command-line interface for managing EdgeLink desktop client.`,
}

func main() {
	// 添加子命令
	rootCmd.AddCommand(registerCmd)
	// TODO: 添加其他命令
	// - status: 查看连接状态
	// - peers: 列出对等设备
	// - disconnect: 断开连接
	// - reconnect: 重新连接

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
