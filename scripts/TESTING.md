# EdgeLink集成测试指南

本文档描述如何验证Edge-Link系统的完整注册流程和P2P隧道建立。

## 前置条件

- Docker和Docker Compose已安装
- 两台测试机器或虚拟机（用于测试设备互联）
- Linux环境（用于WireGuard测试）
- root权限（用于创建TUN接口）

## 1. 启动开发环境

```bash
# 进入项目根目录
cd /home/await/project/edge-link

# 运行开发环境设置脚本
./scripts/dev-setup.sh
```

预期输出：
- ✓ Docker和Docker Compose已就绪
- ✓ PostgreSQL已就绪
- ✓ API Gateway健康检查通过
- 所有服务显示为"Up"状态

## 2. 创建种子数据

```bash
# 生成测试组织、虚拟网络和PSK
./scripts/seed-data.sh
```

预期输出：
- ✓ 组织创建完成: demo-org
- ✓ 虚拟网络创建完成: Demo VPN Network (CIDR: 10.100.0.0/16)
- ✓ 预共享密钥创建完成

记录以下信息（会保存在 `/tmp/edgelink-test-config.env`）：
- ORGANIZATION_SLUG
- VIRTUAL_NETWORK_ID
- PRE_SHARED_KEY

## 3. 注册第一个设备

在测试机器A上：

```bash
# 编译桌面客户端（如果未编译）
cd clients/desktop
go build -o edgelink-cli ./cmd/edgelink-cli
go build -o edgelink-daemon ./cmd/edgelink-daemon

# 注册设备
sudo ./edgelink-cli register \
  --control-plane http://<CONTROL_PLANE_IP>:8080 \
  --psk <PRE_SHARED_KEY> \
  --org demo-org \
  --network <VIRTUAL_NETWORK_ID> \
  --name device-a \
  --config /etc/edgelink/device-a.conf
```

预期输出：
```
Generating device keypair...
Registering device with control plane...
✅ Device registered successfully!
   Device ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
   Virtual IP: 10.100.1.1
✅ Configuration saved to: /etc/edgelink/device-a.conf
```

## 4. 注册第二个设备

在测试机器B上：

```bash
# 使用相同的PSK注册第二个设备
sudo ./edgelink-cli register \
  --control-plane http://<CONTROL_PLANE_IP>:8080 \
  --psk <PRE_SHARED_KEY> \
  --org demo-org \
  --network <VIRTUAL_NETWORK_ID> \
  --name device-b \
  --config /etc/edgelink/device-b.conf
```

预期输出：
```
✅ Device registered successfully!
   Device ID: yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy
   Virtual IP: 10.100.1.2
```

## 5. 启动守护进程

**在设备A上：**

```bash
sudo ./edgelink-daemon --config /etc/edgelink/device-a.conf
```

预期输出：
```
Loading device configuration...
Device ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Virtual IP: 10.100.1.1
Creating WireGuard interface...
Configuring virtual IP: 10.100.1.1
Bringing interface up...
Starting metrics reporter...
EdgeLink daemon is running...
```

**在设备B上：**

```bash
sudo ./edgelink-daemon --config /etc/edgelink/device-b.conf
```

## 6. 验证连通性

### 6.1 检查WireGuard接口

在任一设备上：

```bash
# 查看WireGuard接口状态
sudo wg show edgelink0

# 预期输出包含：
# - interface: edgelink0
# - public key: <设备公钥>
# - listening port: 51820
# - peer: <对等设备公钥>
# - allowed ips: <对等设备虚拟IP>/32
```

### 6.2 Ping测试

**从设备A ping设备B：**

```bash
ping -c 4 10.100.1.2
```

预期输出：
```
PING 10.100.1.2 (10.100.1.2) 56(84) bytes of data.
64 bytes from 10.100.1.2: icmp_seq=1 ttl=64 time=XX ms
64 bytes from 10.100.1.2: icmp_seq=2 ttl=64 time=XX ms
64 bytes from 10.100.1.2: icmp_seq=3 ttl=64 time=XX ms
64 bytes from 10.100.1.2: icmp_seq=4 ttl=64 time=XX ms

--- 10.100.1.2 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss
```

**从设备B ping设备A：**

```bash
ping -c 4 10.100.1.1
```

### 6.3 检查隧道流量

```bash
# 查看WireGuard统计信息
sudo wg show edgelink0 dump

# 预期看到非零的传输/接收字节数
```

## 7. 验证API端点

### 7.1 检查API Gateway健康状态

```bash
curl http://localhost:8080/health
```

预期响应：
```json
{"status":"healthy"}
```

### 7.2 查看设备配置

```bash
curl http://localhost:8080/api/v1/device/<DEVICE_ID>/config
```

预期响应：
```json
{
  "device_id": "...",
  "virtual_ip": "10.100.1.1",
  "virtual_network_id": "...",
  "platform": "linux",
  "peers": [
    {
      "public_key": "...",
      "allowed_ips": ["10.100.1.2/32"],
      "endpoint": "...",
      "persistent_keepalive": 25
    }
  ]
}
```

## 8. 验证指标上报

检查日志确认指标正在上报：

```bash
# API Gateway日志
docker logs edgelink-api-gateway -f | grep metrics

# 守护进程日志（在设备上）
# 应该看到定期的指标上报日志
```

## 9. 故障排查

### 设备无法注册

1. 检查控制平面API是否可达：
   ```bash
   curl http://<CONTROL_PLANE_IP>:8080/health
   ```

2. 检查PSK是否正确和有效

3. 查看API Gateway日志：
   ```bash
   docker logs edgelink-api-gateway
   ```

### 无法建立隧道

1. 检查WireGuard内核模块：
   ```bash
   lsmod | grep wireguard
   ```

2. 检查防火墙规则：
   ```bash
   sudo iptables -L -n -v
   ```

3. 检查NAT类型（如果设备在NAT后）

4. 查看守护进程日志

### Ping失败

1. 确认两个设备都在线：
   ```bash
   curl http://localhost:8080/api/v1/device/<DEVICE_ID>/config
   ```

2. 检查路由表：
   ```bash
   ip route show
   ```

3. 使用tcpdump捕获流量：
   ```bash
   sudo tcpdump -i edgelink0
   ```

## 10. 清理环境

```bash
# 停止守护进程（在设备上）
# Ctrl+C 停止守护进程

# 删除WireGuard接口
sudo ip link delete edgelink0

# 停止Docker服务（在控制平面上）
cd /home/await/project/edge-link
docker-compose down -v
```

## 成功标准

✅ **完整测试通过**条件：

1. 两个设备成功注册并获得虚拟IP
2. 守护进程启动无错误
3. WireGuard接口创建成功并处于UP状态
4. 双向ping测试成功（0%丢包）
5. WireGuard统计显示有数据传输
6. 指标定期上报到控制平面
7. API端点返回正确的设备配置

如果所有条件满足，说明**用户故事1（设备注册和网络连接）功能完全正常且可独立测试**。
