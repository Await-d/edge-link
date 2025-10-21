# EdgeLink Monitoring and Auto-Rollback

本文档介绍EdgeLink的监控和自动回滚系统配置。

## 概述

EdgeLink使用Prometheus + Alertmanager + Kubernetes实现自动化监控和部署回滚：

- **Prometheus**: 收集应用和系统指标
- **PrometheusRule**: 定义告警规则（高错误率、高延迟、Pod崩溃等）
- **Alertmanager**: 告警路由和聚合
- **Auto-Rollback Webhook**: 接收告警并触发Kubernetes部署回滚

## 架构

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Prometheus │────→│ Alertmanager │────→│ Rollback Webhook│
│   (Metrics) │     │  (Routing)   │     │  (Automation)   │
└─────────────┘     └──────────────┘     └─────────────────┘
       ↑                                           │
       │                                           ↓
┌─────────────┐                           ┌─────────────────┐
│   Services  │                           │   Kubernetes    │
│ (Exporters) │                           │ (kubectl rollout│
└─────────────┘                           │     undo)       │
                                          └─────────────────┘
```

## 告警规则

### 关键告警（触发自动回滚）

| 告警名称 | 条件 | 阈值 | 持续时间 | 操作 |
|---------|------|------|---------|------|
| `EdgeLinkHighErrorRate` | HTTP 5xx错误率 | >5% | 2分钟 | 自动回滚 |
| `EdgeLinkHighLatency` | P95延迟 | >500ms | 3分钟 | 自动回滚 |
| `EdgeLinkPodCrashLooping` | Pod重启率 | >0/15分钟 | 1分钟 | 自动回滚 |

### 警告告警（需人工介入）

| 告警名称 | 条件 | 阈值 | 持续时间 | 操作 |
|---------|------|------|---------|------|
| `EdgeLinkDeploymentReplicaMismatch` | 副本数不匹配 | 任何不匹配 | 5分钟 | 调查 |
| `EdgeLinkHighMemoryUsage` | 内存使用率 | >90% | 5分钟 | 调查 |
| `EdgeLinkCPUThrottling` | CPU限流 | >0.1s/s | 3分钟 | 调查 |
| `EdgeLinkDeviceConnectionFailures` | 设备连接失败率 | >10% | 5分钟 | 调查 |
| `EdgeLinkNATTraversalFailures` | NAT穿透失败率 | >30% | 5分钟 | 调查 |
| `EdgeLinkDatabasePoolExhaustion` | 数据库连接池 | >90% | 3分钟 | 调查 |

## 部署配置

### Helm Values配置

```yaml
# values.yaml
monitoring:
  # Prometheus规则
  prometheusRules:
    enabled: true
  
  # 自动回滚配置
  autoRollback:
    enabled: true
    dryRun: false          # 生产环境设置为false启用实际回滚
    maxRetries: 3          # 最大重试次数
    timeout: 300           # 回滚超时（秒）
    webhookUrl: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
  
  # 告警阈值
  thresholds:
    errorRate: 0.05        # 5%错误率
    p95Latency: 0.5        # 500ms P95延迟
    memoryUsage: 0.9       # 90%内存使用
    cpuThrottling: 0.1     # CPU限流
  
  # 告警集成
  alerting:
    slack:
      enabled: true
      webhookUrl: "https://hooks.slack.com/..."
      channel: "#edgelink-alerts"
    
    email:
      enabled: true
      to: "ops@edgelink.com"
      from: "alertmanager@edgelink.com"
      smarthost: "smtp.gmail.com:587"
    
    pagerduty:
      enabled: true
      serviceKey: "YOUR_PAGERDUTY_SERVICE_KEY"
```

### 安装部署

```bash
# 部署EdgeLink Control Plane（包含监控和自动回滚）
helm upgrade --install edge-link \
  ./infrastructure/helm/edge-link-control-plane \
  --namespace edgelink \
  --create-namespace \
  --values values-production.yaml

# 验证PrometheusRule已创建
kubectl get prometheusrule -n edgelink

# 验证Alertmanager配置
kubectl get configmap -n edgelink | grep alertmanager

# 验证Rollback Webhook运行
kubectl get deployment -n edgelink | grep rollback-webhook
kubectl logs -n edgelink deployment/edge-link-rollback-webhook
```

## 手动回滚脚本

当自动回滚失败或需要手动干预时，使用以下脚本：

```bash
# 基本用法
./scripts/auto-rollback.sh <deployment-name> [reason]

# 示例：回滚API Gateway
./scripts/auto-rollback.sh edgelink-api-gateway "High error rate detected"

# Dry run模式（不执行实际操作）
DRY_RUN=true ./scripts/auto-rollback.sh edgelink-device-service

# 自定义配置
NAMESPACE=edgelink-staging \
  MAX_ROLLBACK_RETRIES=5 \
  ROLLBACK_TIMEOUT=600 \
  WEBHOOK_URL=https://hooks.slack.com/... \
  ./scripts/auto-rollback.sh edgelink-topology-service "Manual rollback test"

# 查看可用部署
kubectl get deployments -n edgelink
```

## 本地开发测试

### 启动Prometheus + Alertmanager

```bash
# 使用Docker Compose启动完整监控栈
cd infrastructure/docker
docker-compose up -d prometheus alertmanager

# 查看Prometheus
open http://localhost:9090

# 查看Alertmanager
open http://localhost:9093
```

### 配置文件位置

- **Prometheus配置**: `infrastructure/docker/prometheus.yml`
- **告警规则**: `infrastructure/docker/prometheus-rules.yml`
- **Helm模板**: `infrastructure/helm/edge-link-control-plane/templates/prometheus-rules.yaml`

### 测试告警

```bash
# 手动触发错误率告警（模拟高5xx错误）
for i in {1..100}; do
  curl http://localhost:8080/api/v1/test/error || true
done

# 手动触发延迟告警（模拟慢请求）
for i in {1..50}; do
  curl http://localhost:8080/api/v1/test/slow?delay=1000 &
done
wait

# 查看活跃告警
curl http://localhost:9093/api/v1/alerts | jq
```

## 故障排查

### Rollback Webhook无法接收告警

```bash
# 检查Service和Endpoint
kubectl get svc -n edgelink | grep rollback-webhook
kubectl get endpoints -n edgelink | grep rollback-webhook

# 检查Pod日志
kubectl logs -n edgelink -l app.kubernetes.io/component=rollback-webhook --tail=100

# 测试webhook连接
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -v http://edge-link-rollback-webhook.edgelink.svc.cluster.local:8080/webhook \
  -d '{"deployment":"test","reason":"test"}'
```

### 自动回滚未触发

```bash
# 检查PrometheusRule是否加载
kubectl get prometheusrule -n edgelink -o yaml

# 检查Alertmanager路由配置
kubectl get configmap edge-link-alertmanager-config -n edgelink -o yaml

# 检查活跃告警
kubectl port-forward -n edgelink svc/prometheus 9090:9090
curl http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[] | select(.name | contains("EdgeLink"))'

# 检查Alertmanager接收的告警
kubectl port-forward -n edgelink svc/alertmanager 9093:9093
curl http://localhost:9093/api/v1/alerts | jq
```

### Rollback失败

```bash
# 检查RBAC权限
kubectl auth can-i create deployments/rollback \
  --as=system:serviceaccount:edgelink:edge-link-rollback -n edgelink

# 检查部署历史
kubectl rollout history deployment/<deployment-name> -n edgelink

# 手动回滚并查看详细日志
kubectl rollout undo deployment/<deployment-name> -n edgelink --v=9

# 查看Pod事件
kubectl get events -n edgelink --field-selector involvedObject.name=<deployment-name> \
  --sort-by='.lastTimestamp'
```

## 最佳实践

### 1. 分阶段启用自动回滚

```yaml
# 阶段1：Dry run模式（仅日志，不实际回滚）
monitoring:
  autoRollback:
    enabled: true
    dryRun: true

# 阶段2：非生产环境启用
# values-staging.yaml
monitoring:
  autoRollback:
    enabled: true
    dryRun: false

# 阶段3：生产环境启用（设置更严格阈值）
# values-production.yaml
monitoring:
  thresholds:
    errorRate: 0.03      # 3%错误率（更严格）
    p95Latency: 0.3      # 300ms（更严格）
  autoRollback:
    enabled: true
    dryRun: false
```

### 2. 配置告警抑制

避免告警风暴：

```yaml
# Alertmanager inhibit rules已在模板中配置
# 当critical告警触发时，会抑制相同服务的warning告警
```

### 3. 设置回滚窗口

避免频繁回滚：

```yaml
# 在Helm chart中配置最小部署间隔
apiGateway:
  deployment:
    minReadySeconds: 30
    progressDeadlineSeconds: 600
  
  # 滚动更新策略
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
```

### 4. 监控回滚操作本身

```bash
# 创建Dashboard监控回滚指标
# - 回滚频率
# - 回滚成功率
# - 回滚触发原因
# - 平均回滚时长

# Prometheus查询示例
rate(edgelink_rollback_total[1h])
edgelink_rollback_duration_seconds{quantile="0.95"}
```

## 相关文档

- [灾难恢复手册](../../docs/disaster-recovery.md)
- [Prometheus Operator文档](https://prometheus-operator.dev/)
- [Kubernetes Rollback文档](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-back-a-deployment)
- [Alertmanager配置](https://prometheus.io/docs/alerting/latest/configuration/)

## 支持

如有问题，请联系：
- Slack: #edgelink-ops
- Email: ops@edgelink.com
- On-call: PagerDuty (EdgeLink Platform)
