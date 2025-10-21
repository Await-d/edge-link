#!/bin/sh
set -e

# EdgeLink Sidecar入口脚本

# 环境变量配置
EDGELINK_SERVER=${EDGELINK_SERVER:-"https://api.edgelink.example.com"}
EDGELINK_PSK=${EDGELINK_PSK:-""}
EDGELINK_DEVICE_NAME=${EDGELINK_DEVICE_NAME:-$(hostname)}

# 检查必需的环境变量
if [ -z "$EDGELINK_PSK" ]; then
  echo "Error: EDGELINK_PSK environment variable is required"
  exit 1
fi

# 如果设备未注册,先注册
if [ ! -f "/etc/edgelink/config.json" ]; then
  echo "Registering device..."
  edgelink-lite \
    --server="$EDGELINK_SERVER" \
    --key="$EDGELINK_PSK" \
    --name="$EDGELINK_DEVICE_NAME" \
    --register

  if [ $? -ne 0 ]; then
    echo "Device registration failed"
    exit 1
  fi
  echo "Device registered successfully"
fi

# 连接到EdgeLink网络
echo "Connecting to EdgeLink network..."
exec edgelink-lite "$@"
