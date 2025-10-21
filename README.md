# Edge-Link - 端到端直连系统

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org)
[![React Version](https://img.shields.io/badge/react-19.0-blue.svg)](https://reactjs.org)
[![Backend CI](https://github.com/yourusername/edge-link/workflows/Backend%20CI/CD/badge.svg)](https://github.com/yourusername/edge-link/actions/workflows/backend.yml)
[![Frontend CI](https://github.com/yourusername/edge-link/workflows/Frontend%20CI/CD/badge.svg)](https://github.com/yourusername/edge-link/actions/workflows/frontend.yml)
[![Coverage](https://codecov.io/gh/yourusername/edge-link/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/edge-link)

Edge-Link 是一个企业级的 WireGuard 点对点 (P2P) VPN 管理平台，提供智能 NAT 穿透、集中式控制平面和全面的设备管理能力。

## ✨ 核心特性

- **P2P 直连隧道**: 使用 WireGuard 实现设备间加密隧道连接
- **智能 NAT 穿透**: STUN 探测 + ICE-lite + TURN 回退机制
- **集中式管理**: RESTful API 和 WebSocket 实时更新
- **跨平台支持**: Linux/Windows/macOS 桌面客户端，Android/iOS 移动应用
- **健康监控**: 自动化告警系统，实时设备状态监控
- **安全认证**: Ed25519 签名 + 预共享密钥双重认证

## 📋 系统架构

```
┌─────────────────┐      ┌──────────────────────────────┐      ┌─────────────────┐
│  客户端层       │◄────►│  控制平面                      │◄────►│  管理界面       │
│  (Desktop/Mobile)│      │  - API Gateway                │      │  (React SPA)    │
│  - WireGuard    │      │  - Device Service             │      │  - Ant Design   │
│  - TUN Interface│      │  - Topology Service           │      │  - ECharts      │
│  - NAT Puncher  │      │  - NAT Coordinator            │      │  - WebSocket    │
└─────────────────┘      │  - Alert Service              │      └─────────────────┘
                         │  - Background Worker          │
                         └──────────────────────────────┘
                                     │
                         ┌───────────┴────────────┐
                         │  数据层                 │
                         │  - PostgreSQL          │
                         │  - Redis (Pub/Sub)     │
                         │  - MinIO (S3)          │
                         └────────────────────────┘
```

## 🛠️ 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin (HTTP), gRPC, Fx (依赖注入)
- **数据库**: PostgreSQL 14+, Redis 7+
- **ORM**: GORM
- **消息队列**: Redis Pub/Sub
- **网络**: WireGuard (kernel/wireguard-go), STUN/TURN (coturn)

### 前端
- **语言**: TypeScript 5+
- **框架**: React 19, Vite 5
- **UI**: Ant Design 5
- **图表**: ECharts 5
- **状态管理**: Zustand, TanStack Query

### 移动端
- **iOS**: Swift 5.9+, WireGuardKit
- **Android**: Kotlin 1.9+, wireguard-android

### 基础设施
- **容器化**: Docker, Kubernetes
- **编排**: Helm Charts
- **监控**: Prometheus, Grafana, Jaeger
- **日志**: Loki / ELK Stack

## 📦 快速开始

### 前置要求

- **Docker** 20.10+ & **Docker Compose** 2.0+
- **Go** 1.21+ (本地开发)
- **Node.js** 20+ & npm (前端开发)
- **kubectl** 1.28+ & **helm** 3.13+ (Kubernetes 部署)
- **WireGuard** 内核模块或 wireguard-go (客户端)

### 方式一：Docker Compose 一键启动（推荐）

这是最快的启动方式，适合开发和测试：

```bash
# 1. 克隆仓库
git clone https://github.com/yourusername/edge-link.git
cd edge-link

# 2. 启动所有服务
docker-compose up -d

# 3. 查看服务状态
docker-compose ps

# 服务访问地址：
# - 前端管理界面: http://localhost:13000
# - API Gateway: http://localhost:18080
# - PostgreSQL: localhost:15432
# - Redis: localhost:16379
```

等待所有服务启动后，访问 http://localhost:13000 查看管理界面。

### 方式二：本地开发环境

适合需要修改代码的开发者：

1. **启动基础设施（PostgreSQL, Redis）**
```bash
docker-compose up -d postgres redis
```

2. **运行数据库迁移**
```bash
cd backend
go run internal/migrations/migrate.go up
```

3. **启动控制平面服务**

开启多个终端窗口分别运行：

```bash
# Terminal 1: API Gateway (端口 8080)
cd backend/cmd/api-gateway
go run main.go

# Terminal 2: Device Service (gRPC 端口 50051)
cd backend/cmd/device-service
go run main.go

# Terminal 3: Topology Service (gRPC 端口 50052)
cd backend/cmd/topology-service
go run main.go

# Terminal 4: NAT Coordinator (gRPC 端口 50053)
cd backend/cmd/nat-coordinator
go run main.go

# Terminal 5: Alert Service (可选)
cd backend/cmd/alert-service
go run main.go

# Terminal 6: Background Worker (可选)
cd backend/cmd/background-worker
go run main.go
```

4. **启动前端开发服务器**
```bash
cd frontend
npm install
npm run dev
# 访问 http://localhost:5173
```

5. **种子数据（可选）**
```bash
# 创建测试组织、虚拟网络和PSK
./scripts/seed-data.sh
```

### 方式三：Kubernetes 生产部署

适合生产环境或多节点集群：

```bash
# 1. 创建命名空间
kubectl create namespace edgelink-system

# 2. 部署控制平面（包含所有微服务）
helm install edgelink infrastructure/helm/edge-link-control-plane \
  --namespace edgelink-system \
  --set postgresql.auth.password=your-secure-password \
  --set redis.auth.password=your-redis-password \
  --set ingress.hosts[0].host=api.edgelink.example.com \
  --set frontend.env.apiUrl=https://api.edgelink.example.com \
  --set alertService.smtp.host=smtp.gmail.com \
  --set alertService.smtp.user=noreply@edgelink.com \
  --set-string alertService.smtp.password=your-smtp-password

# 3. 等待所有 Pod 就绪
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=edge-link -n edgelink-system --timeout=300s

# 4. 检查部署状态
kubectl get pods -n edgelink-system
kubectl get svc -n edgelink-system
kubectl get ingress -n edgelink-system

# 5. (可选) 部署 Sidecar DaemonSet（节点级 VPN）
helm install edgelink-sidecar infrastructure/helm/edgelink-sidecar \
  --namespace edgelink-system \
  --set edgelink.serverUrl=https://api.edgelink.example.com \
  --set edgelink.preSharedKey=your-psk-from-control-plane
```

#### 自定义配置

创建 `values-production.yaml` 文件：

```yaml
# values-production.yaml
global:
  imageRegistry: your-registry.io

apiGateway:
  replicaCount: 3
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi

postgresql:
  auth:
    password: your-secure-db-password
  primary:
    persistence:
      size: 100Gi

redis:
  auth:
    password: your-secure-redis-password

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: edgelink.example.com
      paths:
        - path: /
          pathType: Prefix
          backend: frontend
        - path: /api
          pathType: Prefix
          backend: api-gateway
  tls:
    - secretName: edgelink-tls
      hosts:
        - edgelink.example.com
```

使用自定义配置部署：

```bash
helm install edgelink infrastructure/helm/edge-link-control-plane \
  --namespace edgelink-system \
  --values values-production.yaml
```

## 🔑 客户端安装与配置

### 桌面客户端（Linux/Windows/macOS）

#### 下载预编译二进制

从 [Releases](https://github.com/yourusername/edge-link/releases) 页面下载对应平台的客户端。

#### Linux 安装

```bash
# 下载并解压
wget https://github.com/yourusername/edge-link/releases/latest/download/edgelink-linux-amd64.tar.gz
tar xzf edgelink-linux-amd64.tar.gz
sudo mv edgelink-cli edgelink-daemon /usr/local/bin/

# 安装 WireGuard 内核模块
sudo apt update
sudo apt install wireguard

# 或者使用 userspace 实现
sudo apt install wireguard-go
```

#### Windows 安装

1. 下载 `EdgeLinkSetup.msi`
2. 双击运行安装程序
3. 按照向导完成安装
4. 安装程序会自动安装 Wintun 驱动

#### macOS 安装

```bash
# 方式一：使用 .app 包
# 下载 EdgeLink.app，拖到 Applications 文件夹

# 方式二：使用 CLI
wget https://github.com/yourusername/edge-link/releases/latest/download/edgelink-darwin-arm64.tar.gz
tar xzf edgelink-darwin-arm64.tar.gz
sudo mv edgelink-cli /usr/local/bin/
```

### 设备注册

```bash
# 首次注册设备
edgelink-cli register \
  --server https://api.edgelink.example.com \
  --psk your-pre-shared-key \
  --name my-laptop

# 输出示例：
# ✓ Device registered successfully
# Device ID: 550e8400-e29b-41d4-a716-446655440000
# Virtual IP: 10.99.0.5/24
# Virtual Network: default-network
```

### 连接到网络

```bash
# 方式一：使用 CLI 手动连接
sudo edgelink-cli connect

# 方式二：使用守护进程（后台自动连接）
sudo systemctl enable edgelink-daemon
sudo systemctl start edgelink-daemon

# 查看连接状态
edgelink-cli status

# 输出示例：
# Status: Connected
# Virtual IP: 10.99.0.5
# Active Peers: 3
#   - peer1 (10.99.0.2): Direct, 45ms
#   - peer2 (10.99.0.8): Direct, 23ms
#   - peer3 (10.99.0.15): Relay, 120ms
```

### 轻量级客户端（IoT/Edge 设备）

适用于资源受限设备：

```bash
# 下载轻量级客户端（<10MB）
wget https://github.com/yourusername/edge-link/releases/latest/download/edgelink-lite-linux-arm64
chmod +x edgelink-lite-linux-arm64

# 注册并连接
sudo ./edgelink-lite-linux-arm64 \
  --server https://api.edgelink.example.com \
  --key your-psk \
  --name iot-device-01 \
  --register

sudo ./edgelink-lite-linux-arm64 --connect
```

### Docker Sidecar

在容器中使用 EdgeLink：

```bash
# 拉取镜像
docker pull ghcr.io/yourusername/edgelink-sidecar:latest

# 运行 Sidecar
docker run -d \
  --name edgelink-sidecar \
  --cap-add NET_ADMIN \
  --device /dev/net/tun \
  -e EDGELINK_SERVER=https://api.edgelink.example.com \
  -e EDGELINK_PSK=your-psk \
  -e EDGELINK_DEVICE_NAME=my-container \
  ghcr.io/yourusername/edgelink-sidecar:latest
```

### Kubernetes DaemonSet

在 K8s 集群中为每个节点部署 VPN：

```bash
helm install edgelink-sidecar infrastructure/helm/edgelink-sidecar \
  --namespace edgelink-system \
  --set edgelink.serverUrl=https://api.edgelink.example.com \
  --set edgelink.preSharedKey=your-psk

# 验证部署
kubectl get pods -n edgelink-system -l app.kubernetes.io/name=edgelink-sidecar
```

详见 [Sidecar DaemonSet 文档](infrastructure/helm/edgelink-sidecar/README.md)。

## 📊 监控与可观测性

### Prometheus + Grafana

部署监控栈：

```bash
# 1. 部署 Prometheus
kubectl apply -f monitoring/prometheus/prometheus-deployment.yaml

# 2. 部署 Grafana
kubectl apply -f monitoring/grafana/grafana-deployment.yaml

# 3. 导入仪表板
# - Control Plane Overview: monitoring/grafana/dashboards/control-plane-overview.json
# - Device Health: monitoring/grafana/dashboards/device-health.json
# - Tunnel Metrics: monitoring/grafana/dashboards/tunnel-metrics.json
```

访问 Grafana (默认 admin/admin)：
```bash
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

### Loki 日志聚合

```bash
# 部署 Loki
kubectl apply -f monitoring/loki/loki-deployment.yaml

# 部署 Promtail (日志收集器)
kubectl apply -f monitoring/loki/promtail-daemonset.yaml
```

在 Grafana 中添加 Loki 数据源查看日志。

### 关键指标

- `edgelink_devices_total`: 已注册设备总数
- `edgelink_devices_online_total`: 在线设备数
- `edgelink_tunnels_active_total`: 活跃隧道数
- `edgelink_tunnel_latency_milliseconds`: 隧道延迟
- `edgelink_tunnel_packets_dropped_total`: 丢包数
- `http_request_duration_seconds`: HTTP 请求延迟
- `grpc_server_handling_seconds`: gRPC 处理延迟

### 预定义告警

Prometheus 告警规则包括：

- **服务健康**: API Gateway/Device Service/Topology Service 宕机
- **性能**: HTTP/gRPC 高延迟（p95 > 1s）
- **设备健康**: 离线率 > 20%，注册失败率高
- **隧道健康**: 建立失败，高延迟，丢包 > 5%
- **资源**: CPU > 80%，内存 > 85%，磁盘 < 15%
- **数据库**: PostgreSQL 宕机，连接池使用率 > 80%
- **安全**: 认证失败率高，疑似攻击

## 📖 文档

- [架构设计](specs/001-edge-link-core/plan.md)
- [API 文档](specs/001-edge-link-core/contracts/control-plane-api-v1.yaml)
- [WebSocket 事件](specs/001-edge-link-core/contracts/websocket-events.md)
- [部署指南](specs/001-edge-link-core/quickstart.md)
- [数据模型](specs/001-edge-link-core/data-model.md)

## 🧪 测试

```bash
# 后端单元测试
cd backend
go test ./...

# 后端集成测试（需要 Docker）
go test -tags=integration ./...

# 前端测试
cd frontend
npm run test
```

## 🔒 安全性

- **加密**: 所有隧道流量通过 WireGuard 加密（ChaCha20-Poly1305）
- **认证**: Ed25519 公钥签名 + HMAC-SHA256 预共享密钥
- **最小权限**: 基于角色的访问控制 (RBAC)
- **审计日志**: 所有管理操作记录不可变审计日志
- **密钥轮换**: 自动化密钥过期和轮换机制

## 🤝 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详情。

## 📝 许可证

本项目采用 Apache 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 📞 联系方式

- **问题反馈**: [GitHub Issues](https://github.com/yourusername/edge-link/issues)
- **邮件**: support@edgelink.example.com

---

**构建状态**: ![Build](https://github.com/yourusername/edge-link/workflows/CI/badge.svg)
**测试覆盖率**: ![Coverage](https://codecov.io/gh/yourusername/edge-link/branch/master/graph/badge.svg)
