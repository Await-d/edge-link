# EdgeLink 部署指南

本指南提供 EdgeLink 控制平面和客户端的完整部署说明，涵盖开发、测试和生产环境。

## 目录

- [系统要求](#系统要求)
- [架构概览](#架构概览)
- [开发环境部署](#开发环境部署)
- [生产环境部署](#生产环境部署)
- [客户端部署](#客户端部署)
- [监控部署](#监控部署)
- [安全配置](#安全配置)
- [故障恢复](#故障恢复)

## 系统要求

### 控制平面服务器

**最低配置** (开发/测试)：
- CPU: 4核
- 内存: 8GB RAM
- 存储: 50GB SSD
- 网络: 100Mbps

**推荐配置** (生产环境，1000设备)：
- CPU: 8核+
- 内存: 16GB+ RAM
- 存储: 200GB+ SSD (数据库) + 100GB (日志)
- 网络: 1Gbps+

**高可用配置** (5000+设备)：
- API Gateway: 3+ 实例 (2核/4GB 每个)
- Device Service: 3+ 实例 (2核/4GB 每个)
- Topology Service: 2+ 实例 (2核/4GB 每个)
- NAT Coordinator: 2+ 实例 (2核/4GB 每个)
- PostgreSQL: 主从复制 (8核/32GB 主)
- Redis: 主从+哨兵 (4核/8GB 主)

### 客户端设备

- Linux: 内核 ≥ 5.6 (内核 WireGuard) 或任意 + wireguard-go
- Windows: Windows 10/11, Windows Server 2019+
- macOS: macOS 12 (Monterey)+
- iOS: iOS 15+
- Android: Android 8.0+

## 架构概览

```
┌──────────────────────────────────────────────────────────────┐
│                      Internet / WAN                           │
└─────────────────────────┬────────────────────────────────────┘
                          │
                    ┌─────▼──────┐
                    │  Load      │
                    │  Balancer  │  (NGINX/HAProxy/Cloud LB)
                    └─────┬──────┘
                          │
          ┌───────────────┼───────────────┐
          │               │               │
    ┌─────▼──────┐  ┌────▼─────┐  ┌─────▼──────┐
    │ API Gateway│  │ API GW   │  │  API GW    │
    │  Instance1 │  │ Instance2│  │ Instance3  │
    └─────┬──────┘  └────┬─────┘  └─────┬──────┘
          │              │              │
          └──────────────┼──────────────┘
                         │
          ┌──────────────┴──────────────┐
          │                             │
    ┌─────▼─────────┐         ┌─────────▼──────┐
    │  gRPC Services│         │   Redis Cluster│
    │  - Device     │         │   (Pub/Sub)    │
    │  - Topology   │         └────────────────┘
    │  - NAT        │
    └───────┬───────┘
            │
    ┌───────▼────────┐
    │  PostgreSQL    │
    │  Primary       │◄──────┐
    │  + Replicas    │       │ Replication
    └────────────────┘       │
            │                │
    ┌───────▼────────┐  ┌────▼────────┐
    │  Backup        │  │  Standby    │
    │  (WAL Archive) │  │  Replica    │
    └────────────────┘  └─────────────┘
```

## 开发环境部署

### 使用 Docker Compose (最简单)

```bash
# 1. 克隆仓库
git clone https://github.com/yourusername/edge-link.git
cd edge-link

# 2. 配置环境变量
cp .env.example .env
vim .env  # 修改数据库密码等

# 3. 启动所有服务
docker-compose up -d

# 4. 查看日志
docker-compose logs -f

# 5. 验证服务
curl http://localhost:18080/health
```

### 手动启动 (开发调试)

```bash
# 1. 启动基础设施
docker-compose up -d postgres redis

# 2. 运行数据库迁移
cd backend
go run internal/migrations/migrate.go up

# 3. 启动后端服务 (多个终端)
# Terminal 1
cd backend/cmd/api-gateway
go run main.go

# Terminal 2
cd backend/cmd/device-service
go run main.go

# Terminal 3
cd backend/cmd/topology-service
go run main.go

# Terminal 4
cd backend/cmd/nat-coordinator
go run main.go

# 4. 启动前端
cd frontend
npm install
npm run dev
```

## 生产环境部署

### Kubernetes 部署 (推荐)

#### 前置条件

- Kubernetes 集群 ≥ 1.26
- Helm ≥ 3.13
- kubectl 已配置
- Ingress Controller (NGINX 推荐)
- cert-manager (可选，用于 TLS)

#### 步骤 1: 准备命名空间

```bash
# 创建命名空间
kubectl create namespace edgelink-prod

# 创建 Docker registry secret (如使用私有镜像)
kubectl create secret docker-registry edgelink-registry \
  --docker-server=ghcr.io \
  --docker-username=your-username \
  --docker-password=your-token \
  --namespace edgelink-prod
```

#### 步骤 2: 配置 values.yaml

创建 `values-production.yaml`:

```yaml
# values-production.yaml
global:
  imageRegistry: ghcr.io/yourusername
  imagePullSecrets:
    - edgelink-registry

# API Gateway 配置
apiGateway:
  replicaCount: 3
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 500m
      memory: 512Mi
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70

# Device Service 配置
deviceService:
  replicaCount: 3
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi

# Topology Service 配置
topologyService:
  replicaCount: 2
  resources:
    limits:
      cpu: 500m
      memory: 512Mi

# NAT Coordinator 配置
natCoordinator:
  replicaCount: 2
  env:
    STUN_SERVER_ADDRESS: "stun.edgelink.com:3478"

# PostgreSQL 配置 (推荐使用外部托管数据库)
postgresql:
  enabled: false  # 使用外部数据库

# 外部数据库连接
externalDatabase:
  host: postgres.example.com
  port: 5432
  database: edgelink_prod
  username: edgelink
  existingSecret: edgelink-db-secret
  existingSecretPasswordKey: password

# Redis 配置 (推荐使用外部 Redis)
redis:
  enabled: false

externalRedis:
  host: redis.example.com
  port: 6379
  existingSecret: edgelink-redis-secret
  existingSecretPasswordKey: password

# Ingress 配置
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
  hosts:
    - host: api.edgelink.com
      paths:
        - path: /
          pathType: Prefix
          backend: frontend
        - path: /api
          pathType: Prefix
          backend: api-gateway
        - path: /ws
          pathType: Prefix
          backend: api-gateway
  tls:
    - secretName: edgelink-tls
      hosts:
        - api.edgelink.com

# 前端配置
frontend:
  replicaCount: 2
  env:
    apiUrl: "https://api.edgelink.com"
    wsUrl: "wss://api.edgelink.com/ws"

# Alert Service 配置
alertService:
  enabled: true
  smtp:
    host: smtp.sendgrid.net
    port: "587"
    user: apikey
    existingSecret: edgelink-smtp-secret

# 监控配置
serviceMonitor:
  enabled: true
  interval: 30s
```

#### 步骤 3: 创建 Secrets

```bash
# 数据库密码
kubectl create secret generic edgelink-db-secret \
  --from-literal=password='your-secure-db-password' \
  --namespace edgelink-prod

# Redis 密码
kubectl create secret generic edgelink-redis-secret \
  --from-literal=password='your-secure-redis-password' \
  --namespace edgelink-prod

# SMTP 密码
kubectl create secret generic edgelink-smtp-secret \
  --from-literal=smtp-password='your-smtp-api-key' \
  --namespace edgelink-prod
```

#### 步骤 4: 部署控制平面

```bash
# 安装或升级
helm upgrade --install edgelink \
  infrastructure/helm/edge-link-control-plane \
  --namespace edgelink-prod \
  --values values-production.yaml \
  --wait \
  --timeout 10m

# 验证部署
kubectl get pods -n edgelink-prod
kubectl get svc -n edgelink-prod
kubectl get ingress -n edgelink-prod
```

#### 步骤 5: 验证服务

```bash
# 检查 Pod 状态
kubectl get pods -n edgelink-prod -w

# 查看日志
kubectl logs -f deployment/edgelink-api-gateway -n edgelink-prod

# 测试健康检查
curl https://api.edgelink.com/health

# 测试 API
curl https://api.edgelink.com/api/v1/health
```

### 数据库迁移

生产环境首次部署需要运行迁移：

```bash
# 使用 Job 运行迁移
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: edgelink-migrate
  namespace: edgelink-prod
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: ghcr.io/yourusername/edgelink-api-gateway:latest
        command: ["/app/migrate", "up"]
        env:
        - name: DB_HOST
          value: postgres.example.com
        - name: DB_PORT
          value: "5432"
        - name: DB_NAME
          value: edgelink_prod
        - name: DB_USER
          value: edgelink
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: edgelink-db-secret
              key: password
      restartPolicy: Never
  backoffLimit: 3
EOF

# 查看迁移日志
kubectl logs -f job/edgelink-migrate -n edgelink-prod
```

## 客户端部署

### Linux 客户端

```bash
# 下载二进制
wget https://github.com/yourusername/edge-link/releases/latest/download/edgelink-linux-amd64.tar.gz
tar xzf edgelink-linux-amd64.tar.gz
sudo mv edgelink-cli /usr/local/bin/
sudo mv edgelink-daemon /usr/local/bin/

# 安装 WireGuard
sudo apt update && sudo apt install -y wireguard

# 注册设备
sudo edgelink-cli register \
  --server https://api.edgelink.com \
  --psk your-pre-shared-key \
  --name $(hostname)

# 安装 systemd service
sudo tee /etc/systemd/system/edgelink.service > /dev/null <<EOF
[Unit]
Description=EdgeLink VPN Client
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/edgelink-daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable edgelink
sudo systemctl start edgelink
```

### Kubernetes DaemonSet 部署

在 K8s 集群中为每个节点部署 VPN:

```bash
# 创建 PSK secret
kubectl create secret generic edgelink-psk \
  --from-literal=psk='your-pre-shared-key' \
  --namespace edgelink-system

# 部署 DaemonSet
helm install edgelink-sidecar infrastructure/helm/edgelink-sidecar \
  --namespace edgelink-system \
  --create-namespace \
  --set edgelink.serverUrl=https://api.edgelink.com \
  --set-string edgelink.preSharedKey='your-psk'

# 验证
kubectl get pods -n edgelink-system -l app.kubernetes.io/name=edgelink-sidecar
```

## 监控部署

### Prometheus + Grafana

```bash
# 部署 Prometheus
kubectl create namespace monitoring
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# 应用 EdgeLink ServiceMonitor
kubectl apply -f monitoring/prometheus/servicemonitor.yaml

# 导入 Grafana 仪表板
kubectl create configmap edgelink-dashboards \
  --from-file=monitoring/grafana/dashboards/ \
  --namespace monitoring
```

### Loki 日志聚合

```bash
# 部署 Loki
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack \
  --namespace monitoring \
  --set promtail.enabled=true

# 配置 Promtail 收集 EdgeLink 日志
kubectl apply -f monitoring/loki/promtail-config.yaml
```

## 安全配置

### TLS 证书

使用 cert-manager 自动管理证书：

```bash
# 安装 cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# 创建 Let's Encrypt ClusterIssuer
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@edgelink.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

### 网络策略

```bash
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: edgelink-network-policy
  namespace: edgelink-prod
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: edge-link
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: edgelink-prod
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 50051-50053
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 6379  # Redis
    - protocol: TCP
      port: 443   # HTTPS
    - protocol: UDP
      port: 53    # DNS
EOF
```

## 故障恢复

### 备份策略

#### 数据库备份

```bash
# 每日全量备份
kubectl create cronjob edgelink-db-backup \
  --image=postgres:15 \
  --schedule="0 2 * * *" \
  --restart=OnFailure \
  -- sh -c "pg_dump -h postgres.example.com -U edgelink edgelink_prod | gzip > /backup/edgelink-$(date +%Y%m%d).sql.gz"
```

#### 恢复数据库

```bash
# 从备份恢复
gunzip < edgelink-20240101.sql.gz | psql -h postgres.example.com -U edgelink edgelink_prod
```

### 滚动更新

```bash
# 零停机更新
helm upgrade edgelink infrastructure/helm/edge-link-control-plane \
  --namespace edgelink-prod \
  --values values-production.yaml \
  --set apiGateway.image.tag=v1.2.0 \
  --wait

# 回滚到上一版本
helm rollback edgelink --namespace edgelink-prod
```

### 紧急扩容

```bash
# 快速扩容 API Gateway
kubectl scale deployment edgelink-api-gateway \
  --replicas=10 \
  --namespace edgelink-prod

# 快速扩容 Device Service
kubectl scale deployment edgelink-device-service \
  --replicas=5 \
  --namespace edgelink-prod
```

## 环境变量参考

### API Gateway

| 变量 | 说明 | 默认值 |
|-----|-----|--------|
| `SERVER_PORT` | HTTP 端口 | `8080` |
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_NAME` | 数据库名称 | `edgelink` |
| `DB_USER` | 数据库用户 | `edgelink` |
| `DB_PASSWORD` | 数据库密码 | (required) |
| `REDIS_HOST` | Redis 主机 | `localhost` |
| `REDIS_PORT` | Redis 端口 | `6379` |
| `LOG_LEVEL` | 日志级别 | `info` |

### Device Service

| 变量 | 说明 | 默认值 |
|-----|-----|--------|
| `GRPC_PORT` | gRPC 端口 | `50051` |
| `DB_HOST` | 数据库主机 | `localhost` |

### Alert Service

| 变量 | 说明 | 默认值 |
|-----|-----|--------|
| `SMTP_HOST` | SMTP 服务器 | (required) |
| `SMTP_PORT` | SMTP 端口 | `587` |
| `SMTP_USER` | SMTP 用户名 | (required) |
| `SMTP_PASSWORD` | SMTP 密码 | (required) |

## 下一步

- [故障排查指南](troubleshooting.md)
- [API 文档](api/README.md)
- [监控指南](monitoring.md)
- [安全最佳实践](security.md)

---

最后更新: 2024-01-01
