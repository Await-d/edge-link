# EdgeLink 故障排查指南

本指南提供 EdgeLink 常见问题的诊断和解决方法。

## 目录

- [通用诊断工具](#通用诊断工具)
- [控制平面问题](#控制平面问题)
- [客户端连接问题](#客户端连接问题)
- [网络问题](#网络问题)
- [性能问题](#性能问题)
- [数据库问题](#数据库问题)
- [日志分析](#日志分析)

## 通用诊断工具

### 健康检查命令

```bash
# 检查所有服务状态
kubectl get pods -n edgelink-system

# 检查服务日志
kubectl logs -f deployment/edgelink-api-gateway -n edgelink-system --tail=100

# 检查服务健康端点
curl http://api.edgelink.com/health

# 检查数据库连接
kubectl exec -it deployment/edgelink-api-gateway -n edgelink-system -- \
  psql -h postgres -U edgelink -d edgelink -c "SELECT 1"

# 检查 Redis 连接
kubectl exec -it deployment/edgelink-api-gateway -n edgelink-system -- \
  redis-cli -h redis ping
```

### 查看指标

```bash
# Prometheus 查询
kubectl port-forward -n monitoring svc/prometheus-operated 9090:9090
# 访问 http://localhost:9090

# Grafana 仪表板
kubectl port-forward -n monitoring svc/grafana 3000:3000
# 访问 http://localhost:3000
```

## 控制平面问题

### 问题: API Gateway 无响应

**症状**:
- HTTP 请求超时
- 502/503 错误
- Pod 状态为 CrashLoopBackOff

**诊断步骤**:

```bash
# 1. 检查 Pod 状态
kubectl get pods -n edgelink-system | grep api-gateway

# 2. 查看 Pod 事件
kubectl describe pod <pod-name> -n edgelink-system

# 3. 查看日志
kubectl logs <pod-name> -n edgelink-system --previous

# 4. 检查资源使用
kubectl top pod <pod-name> -n edgelink-system
```

**常见原因和解决方法**:

1. **数据库连接失败**
   ```bash
   # 检查数据库密码
   kubectl get secret edgelink-db-secret -n edgelink-system -o jsonpath='{.data.password}' | base64 -d

   # 测试数据库连接
   kubectl run psql-test --rm -it --image=postgres:15 -- \
     psql -h postgres -U edgelink -d edgelink
   ```

2. **内存不足 (OOMKilled)**
   ```bash
   # 检查是否被 OOM 杀死
   kubectl describe pod <pod-name> -n edgelink-system | grep -i oom

   # 增加内存限制
   kubectl set resources deployment/edgelink-api-gateway \
     --limits=memory=2Gi \
     --namespace edgelink-system
   ```

3. **配置错误**
   ```bash
   # 检查环境变量
   kubectl get deployment edgelink-api-gateway -n edgelink-system -o yaml | grep -A 20 env:
   ```

### 问题: gRPC 服务通信失败

**症状**:
- API Gateway 日志显示 "connection refused"
- Device/Topology Service 不可达

**诊断**:

```bash
# 测试 gRPC 连接
kubectl exec -it deployment/edgelink-api-gateway -n edgelink-system -- \
  grpcurl -plaintext device-service:50051 list

# 检查 Service DNS
kubectl exec -it deployment/edgelink-api-gateway -n edgelink-system -- \
  nslookup device-service

# 检查网络策略
kubectl get networkpolicies -n edgelink-system
```

**解决方法**:
- 确保 Service 端口正确暴露
- 检查网络策略是否允许流量
- 验证 Pod 标签选择器

### 问题: WebSocket 连接断开

**症状**:
- 前端实时更新不工作
- WebSocket 连接频繁断开

**诊断**:

```bash
# 检查 WebSocket 连接
wscat -c wss://api.edgelink.com/ws

# 查看 Redis Pub/Sub
kubectl exec -it deployment/redis -n edgelink-system -- \
  redis-cli
> PSUBSCRIBE edgelink:*
```

**常见原因**:
1. **Ingress 超时配置太短**
   ```yaml
   # 增加 Ingress 超时
   nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
   nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
   ```

2. **Redis 连接丢失**
   - 检查 Redis 健康状态
   - 验证 Redis 密码配置

## 客户端连接问题

### 问题: 设备注册失败

**症状**:
- `edgelink-cli register` 返回认证错误
- 403/401 HTTP 状态码

**诊断**:

```bash
# 详细模式运行
edgelink-cli register \
  --server https://api.edgelink.com \
  --psk your-psk \
  --debug

# 检查服务器日志
kubectl logs -f deployment/edgelink-api-gateway -n edgelink-system | grep register

# 验证 PSK
kubectl exec -it deployment/edgelink-api-gateway -n edgelink-system -- \
  psql -h postgres -U edgelink -d edgelink \
  -c "SELECT * FROM pre_shared_keys WHERE key_hash = encode(digest('your-psk', 'sha256'), 'hex');"
```

**解决方法**:
1. 验证 PSK 正确性
2. 检查组织是否存在
3. 确认 PSK 未过期

### 问题: WireGuard 隧道无法建立

**症状**:
- `edgelink-cli status` 显示 "Disconnected"
- 无法 ping 其他设备

**诊断**:

```bash
# 检查 WireGuard 接口
sudo wg show

# 检查路由表
ip route show

# 测试 UDP 连通性
nc -u -v <peer-ip> 51820

# 检查防火墙
sudo iptables -L -n | grep 51820
```

**常见问题**:

1. **WireGuard 未安装/未加载**
   ```bash
   # Linux
   sudo modprobe wireguard
   lsmod | grep wireguard

   # 或安装 wireguard-go
   sudo apt install wireguard-go
   ```

2. **UDP 51820 端口被阻止**
   ```bash
   # 开放端口
   sudo ufw allow 51820/udp
   sudo iptables -A INPUT -p udp --dport 51820 -j ACCEPT
   ```

3. **NAT 类型不兼容**
   - 检查 NAT Coordinator 日志
   - 验证 STUN/TURN 配置
   - 考虑使用 TURN 中继

### 问题: 设备显示离线

**症状**:
- 管理界面显示设备离线
- 但设备本地显示已连接

**诊断**:

```bash
# 检查心跳发送
edgelink-cli metrics --verbose

# 检查服务器是否收到心跳
kubectl logs -f deployment/edgelink-api-gateway -n edgelink-system | grep metrics

# 检查设备最后在线时间
kubectl exec -it deployment/edgelink-api-gateway -n edgelink-system -- \
  psql -h postgres -U edgelink -d edgelink \
  -c "SELECT id, name, last_seen_at FROM devices WHERE name = 'your-device';"
```

**解决方法**:
- 检查客户端网络连接
- 验证 API 认证配置
- 检查客户端守护进程是否运行

## 网络问题

### 问题: NAT 穿透失败

**症状**:
- 两个设备无法建立直连
- 所有流量都通过 TURN 中继

**诊断**:

```bash
# 检查 NAT 类型
edgelink-cli nat-detect

# 查看 NAT Coordinator 日志
kubectl logs -f deployment/edgelink-nat-coordinator -n edgelink-system

# 测试 STUN 服务器
stun stun.edgelink.com 3478
```

**NAT 类型兼容性**:
| Client A \ Client B | Full Cone | Restricted | Port Restricted | Symmetric |
|---------------------|-----------|------------|-----------------|-----------|
| Full Cone           | ✅ Direct | ✅ Direct  | ✅ Direct       | ❌ Relay  |
| Restricted          | ✅ Direct | ✅ Direct  | ⚠️ Maybe       | ❌ Relay  |
| Port Restricted     | ✅ Direct | ⚠️ Maybe  | ❌ Relay        | ❌ Relay  |
| Symmetric           | ❌ Relay  | ❌ Relay   | ❌ Relay        | ❌ Relay  |

**解决方法**:
1. 配置 TURN 服务器作为中继
2. 使用 UPnP/NAT-PMP 映射端口
3. 手动配置端口转发

### 问题: 高延迟/丢包

**症状**:
- Ping 延迟 > 500ms
- 丢包率 > 5%

**诊断**:

```bash
# 测试设备间延迟
ping -c 100 10.99.0.5

# 检查 WireGuard 统计
sudo wg show all dump

# 查看隧道指标
# 在 Grafana 查看 Tunnel Metrics 仪表板
```

**常见原因**:
1. **路由次优** - 流量经过 TURN 中继
2. **网络拥塞** - 检查带宽使用情况
3. **MTU 问题** - 尝试降低 MTU (1420 → 1280)

## 性能问题

### 问题: API 响应慢

**症状**:
- HTTP 请求延迟 > 1s
- 前端加载缓慢

**诊断**:

```bash
# 检查 Prometheus 指标
http_request_duration_seconds{quantile="0.95"}

# 查看慢查询
kubectl exec -it deployment/postgres -n edgelink-system -- \
  psql -U edgelink -d edgelink \
  -c "SELECT query, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"

# 检查数据库连接池
kubectl logs deployment/edgelink-api-gateway -n edgelink-system | grep "connection pool"
```

**优化方法**:
1. 增加数据库连接池大小
2. 添加数据库索引
3. 启用 Redis 缓存
4. 横向扩展服务实例

### 问题: 内存使用过高

**症状**:
- Pod 被 OOMKilled
- 内存使用持续增长

**诊断**:

```bash
# 查看内存使用
kubectl top pods -n edgelink-system

# 生成堆转储 (Go)
kubectl exec -it <pod-name> -n edgelink-system -- \
  curl http://localhost:6060/debug/pprof/heap > heap.prof

# 分析堆转储
go tool pprof heap.prof
```

**解决方法**:
1. 增加内存限制
2. 检查内存泄漏
3. 优化数据结构
4. 添加对象池

## 数据库问题

### 问题: 数据库连接耗尽

**症状**:
- "too many clients" 错误
- 服务无法获取数据库连接

**诊断**:

```bash
# 查看活动连接
kubectl exec -it deployment/postgres -n edgelink-system -- \
  psql -U edgelink -d edgelink \
  -c "SELECT count(*) FROM pg_stat_activity;"

# 查看连接详情
kubectl exec -it deployment/postgres -n edgelink-system -- \
  psql -U edgelink -d edgelink \
  -c "SELECT usename, application_name, state, count(*) FROM pg_stat_activity GROUP BY usename, application_name, state;"
```

**解决方法**:
```sql
-- 增加最大连接数
ALTER SYSTEM SET max_connections = 200;
SELECT pg_reload_conf();

-- 杀死空闲连接
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'idle'
AND state_change < NOW() - INTERVAL '5 minutes';
```

### 问题: 数据库性能下降

**症状**:
- 查询执行时间增加
- CPU 使用率高

**诊断**:

```bash
# 检查慢查询
SELECT query, calls, mean_exec_time, max_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 100
ORDER BY mean_exec_time DESC
LIMIT 20;

# 检查缺失索引
SELECT schemaname, tablename, attname, n_distinct, correlation
FROM pg_stats
WHERE schemaname = 'public'
AND n_distinct > 100
ORDER BY correlation;

# 检查表膨胀
SELECT tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

**优化方法**:
```sql
-- 添加索引
CREATE INDEX idx_devices_last_seen ON devices(last_seen_at);
CREATE INDEX idx_sessions_device_id ON sessions(device_id, created_at);

-- VACUUM 表
VACUUM ANALYZE devices;
VACUUM ANALYZE sessions;

-- 更新统计信息
ANALYZE;
```

## 日志分析

### 使用 kubectl logs

```bash
# 实时日志
kubectl logs -f deployment/edgelink-api-gateway -n edgelink-system

# 最近 100 行
kubectl logs deployment/edgelink-api-gateway -n edgelink-system --tail=100

# 过滤错误
kubectl logs deployment/edgelink-api-gateway -n edgelink-system | grep -i error

# 多个 Pod 日志
kubectl logs -l app.kubernetes.io/name=edgelink-api-gateway -n edgelink-system --all-containers=true
```

### 使用 Loki/Grafana

```bash
# LogQL 查询示例
{namespace="edgelink-system", app="api-gateway"} |= "error" | json

# 查询特定设备
{namespace="edgelink-system"} | json | device_id="550e8400-e29b-41d4-a716-446655440000"

# 查询慢请求
{namespace="edgelink-system"} | json | duration > 1s
```

### 常见错误模式

**1. 认证失败**
```
ERROR auth: signature verification failed device_id=xxx
```
→ 检查设备密钥是否正确

**2. 数据库锁超时**
```
ERROR database: pq: deadlock detected
```
→ 检查事务逻辑，减少锁争用

**3. Redis 连接失败**
```
ERROR redis: dial tcp: i/o timeout
```
→ 检查 Redis 健康状态和网络

**4. gRPC 超时**
```
ERROR grpc: context deadline exceeded
```
→ 增加超时时间或优化服务性能

## 获取帮助

如果问题仍未解决：

1. **收集诊断信息**:
   ```bash
   # 运行诊断脚本
   ./scripts/collect-diagnostics.sh
   ```

2. **提交 Issue**:
   - GitHub Issues: https://github.com/yourusername/edge-link/issues
   - 包含诊断信息和日志
   - 描述复现步骤

3. **联系支持**:
   - Email: support@edgelink.com
   - Slack: edgelink.slack.com

---

最后更新: 2024-01-01
