# 灾难恢复和回滚手册

**最后更新**: 2025-10-20
**维护者**: EdgeLink DevOps Team
**紧急联系**: @devops-team (Slack), ops@edgelink.com

---

## 目录

1. [回滚触发条件](#1-回滚触发条件)
2. [应用回滚](#2-应用回滚)
3. [数据库回滚](#3-数据库回滚)
4. [完全灾难恢复](#4-完全灾难恢复)
5. [回滚验证](#5-回滚验证)
6. [常见失败场景处理](#6-常见失败场景处理)
7. [事后分析](#7-事后分析)

---

## 1. 回滚触发条件

### 自动回滚触发器 (生产环境)

EdgeLink已实现**完全自动化的回滚系统**（基于Prometheus + Alertmanager + Kubernetes），以下指标异常将自动触发回滚操作：

> **📖 详细配置参考**: [监控和自动回滚文档](../infrastructure/monitoring/README.md)
> 
> **🔧 配置文件位置**: `infrastructure/helm/edge-link-control-plane/templates/prometheus-rules.yaml`

#### 关键告警（自动触发回滚）

| 指标 | 阈值 | 基线 | 观察期 |
|------|------|------|--------|
| HTTP 5xx 错误率 | > 5% | < 0.1% | 5分钟 |
| P95响应延迟 | > 500ms | < 150ms | 5分钟 |
| Pod CrashLoopBackOff | > 0 | 0 | 立即 |
| 健康检查失败率 | > 10% | 0% | 2分钟 |
| 内存使用率 | > 90% | < 70% | 5分钟 |
| CPU使用率 | > 95% | < 60% | 5分钟 |
| 活跃连接数 | < 50% baseline | N/A | 3分钟 |

### 手动回滚触发条件

- 关键功能不可用（设备注册失败、连接建立失败）
- 数据一致性问题（设备配置错误、拓扑数据不匹配）
- 安全漏洞紧急修复回滚
- 客户端兼容性问题（旧版本客户端无法连接）

---

## 2. 应用回滚

### 2.1 Kubernetes 回滚 (推荐)

#### 场景: Helm Chart 部署出错

```bash
# 1. 查看发布历史
helm history edge-link-production -n edgelink-prod

# 输出示例:
# REVISION  UPDATED                  STATUS      CHART                    DESCRIPTION
# 1         Mon Oct 15 10:00:00 2025 superseded  edge-link-0.1.0          Install complete
# 2         Tue Oct 16 14:30:00 2025 superseded  edge-link-0.1.1          Upgrade complete
# 3         Wed Oct 17 16:45:00 2025 deployed    edge-link-0.2.0          Upgrade complete

# 2. 回滚到上一版本 (REVISION 2)
helm rollback edge-link-production 2 -n edgelink-prod

# 3. 等待回滚完成
kubectl rollout status deployment/edge-link-production-api-gateway -n edgelink-prod --timeout=5m

# 4. 验证所有服务
kubectl get pods -n edgelink-prod
kubectl get svc -n edgelink-prod

# 5. 运行健康检查
curl -f https://api.edgelink.production/health || echo "❌ Health check failed"
```

#### 场景: 单个微服务回滚

```bash
# 1. 查看部署历史
kubectl rollout history deployment/edge-link-production-api-gateway -n edgelink-prod

# 2. 回滚到上一版本
kubectl rollout undo deployment/edge-link-production-api-gateway -n edgelink-prod

# 3. 回滚到特定版本 (REVISION 5)
kubectl rollout undo deployment/edge-link-production-api-gateway --to-revision=5 -n edgelink-prod

# 4. 暂停部署（紧急止血）
kubectl rollout pause deployment/edge-link-production-api-gateway -n edgelink-prod

# 5. 恢复部署
kubectl rollout resume deployment/edge-link-production-api-gateway -n edgelink-prod

# 6. 验证回滚状态
kubectl rollout status deployment/edge-link-production-api-gateway -n edgelink-prod
```

### 2.2 Docker 镜像回滚

```bash
# 1. 确认当前镜像版本
kubectl describe deployment/edge-link-production-api-gateway -n edgelink-prod | grep Image

# 2. 更新到已知良好版本
kubectl set image deployment/edge-link-production-api-gateway \
  api-gateway=ghcr.io/edgelink/edgelink-api-gateway:v0.1.5 \
  -n edgelink-prod

# 3. 验证镜像更新
kubectl get pods -n edgelink-prod -o jsonpath='{.items[*].spec.containers[*].image}'
```

### 2.3 金丝雀部署回滚

```bash
# 使用Istio/Flagger进行金丝雀部署时的回滚

# 1. 检查金丝雀状态
kubectl get canary edge-link-api-gateway -n edgelink-prod

# 2. 手动回滚金丝雀（将流量设为0）
kubectl patch canary edge-link-api-gateway -n edgelink-prod \
  --type=merge \
  -p='{"spec": {"analysis": {"maxWeight": 0}}}'

# 3. 删除有问题的金丝雀版本
kubectl delete canary edge-link-api-gateway -n edgelink-prod
```

---

## 3. 数据库回滚

### 3.1 迁移回滚 (使用golang-migrate)

#### 场景: 数据库迁移失败

```bash
# 1. 检查当前迁移版本
docker exec -it edge-link-postgres psql -U edgelink -d edgelink_prod \
  -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"

# 2. 查看迁移历史
ls -la backend/migrations/

# 3. 回滚最后一次迁移
docker run --rm -v $(pwd)/backend/migrations:/migrations \
  --network edgelink-prod \
  migrate/migrate:latest \
  -path=/migrations \
  -database="postgres://edgelink:password@postgres:5432/edgelink_prod?sslmode=disable" \
  down 1

# 4. 验证迁移版本
docker exec -it edge-link-postgres psql -U edgelink -d edgelink_prod \
  -c "SELECT version FROM schema_migrations;"

# 5. 检查数据一致性
docker exec -it edge-link-postgres psql -U edgelink -d edgelink_prod \
  -c "SELECT COUNT(*) FROM devices; SELECT COUNT(*) FROM virtual_networks;"
```

#### 场景: 迁移不可逆（破坏性变更）

```bash
# ⚠️ 如果迁移删除了列或表，必须从备份恢复

# 1. 停止所有应用实例
kubectl scale deployment --all --replicas=0 -n edgelink-prod

# 2. 创建当前数据库快照（保留出错状态）
kubectl exec -it postgres-0 -n edgelink-prod -- \
  pg_dump -U edgelink edgelink_prod > /backup/failed_migration_$(date +%Y%m%d_%H%M%S).sql

# 3. 从最近的良好备份恢复
kubectl exec -it postgres-0 -n edgelink-prod -- \
  psql -U edgelink -d postgres -c "DROP DATABASE edgelink_prod;"
kubectl exec -it postgres-0 -n edgelink-prod -- \
  psql -U edgelink -d postgres -c "CREATE DATABASE edgelink_prod;"
kubectl exec -it postgres-0 -n edgelink-prod -- \
  pg_restore -U edgelink -d edgelink_prod /backup/latest_good_backup.dump

# 4. 验证数据恢复
kubectl exec -it postgres-0 -n edgelink-prod -- \
  psql -U edgelink -d edgelink_prod -c "SELECT COUNT(*) FROM devices;"

# 5. 重启应用（使用旧版本）
helm rollback edge-link-production <GOOD_REVISION> -n edgelink-prod
kubectl scale deployment --all --replicas=3 -n edgelink-prod
```

### 3.2 数据库备份验证

```bash
# 每日自动备份验证脚本
#!/bin/bash
BACKUP_FILE="/backup/edgelink_prod_$(date +%Y%m%d).dump"

# 1. 创建测试数据库
docker exec -it edge-link-postgres psql -U edgelink -d postgres \
  -c "DROP DATABASE IF EXISTS edgelink_test_restore; CREATE DATABASE edgelink_test_restore;"

# 2. 恢复到测试数据库
docker exec -it edge-link-postgres pg_restore \
  -U edgelink -d edgelink_test_restore "$BACKUP_FILE"

# 3. 验证关键表
docker exec -it edge-link-postgres psql -U edgelink -d edgelink_test_restore <<EOF
SELECT 'devices' AS table_name, COUNT(*) FROM devices
UNION ALL
SELECT 'virtual_networks', COUNT(*) FROM virtual_networks
UNION ALL
SELECT 'sessions', COUNT(*) FROM sessions;
EOF

# 4. 清理测试数据库
docker exec -it edge-link-postgres psql -U edgelink -d postgres \
  -c "DROP DATABASE edgelink_test_restore;"

echo "✅ Backup verified successfully"
```

---

## 4. 完全灾难恢复

### 4.1 Kubernetes 集群不可用

#### RTO (Recovery Time Objective): < 30分钟
#### RPO (Recovery Point Objective): < 15分钟

```bash
# 1. 切换到DR集群
kubectl config use-context dr-cluster-production

# 2. 恢复持久卷（使用Velero）
velero restore create production-dr-restore \
  --from-backup daily-backup-$(date -d "yesterday" +%Y%m%d) \
  --wait

# 3. 验证PVC恢复
kubectl get pvc -n edgelink-prod

# 4. 恢复应用
helm install edge-link-production \
  ./infrastructure/helm/edge-link-control-plane \
  -n edgelink-prod \
  --create-namespace \
  --values ./infrastructure/helm/values-production.yaml \
  --wait

# 5. 验证所有Pod运行
kubectl get pods -n edgelink-prod

# 6. 更新DNS指向DR集群
# 使用Cloudflare API或手动更新
curl -X PUT "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records/${RECORD_ID}" \
  -H "Authorization: Bearer ${CF_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{
    "type": "A",
    "name": "api.edgelink.com",
    "content": "'${DR_CLUSTER_IP}'",
    "ttl": 60,
    "proxied": false
  }'

# 7. 等待DNS传播（TTL=60s）
sleep 120

# 8. 运行完整smoke tests
./scripts/full-smoke-test.sh https://api.edgelink.com

# 9. 验证数据一致性
kubectl exec -it postgres-0 -n edgelink-prod -- \
  psql -U edgelink -d edgelink_prod -c "
    SELECT
      (SELECT COUNT(*) FROM devices) as devices_count,
      (SELECT COUNT(*) FROM virtual_networks) as vnet_count,
      (SELECT COUNT(*) FROM sessions WHERE ended_at IS NULL) as active_sessions;
  "

# 10. 通知客户端重连（如果需要）
# 发送WebSocket通知或等待客户端自动重连
```

### 4.2 数据中心完全故障

```bash
# 多区域部署时的区域切换

# 1. 检查区域健康状态
kubectl get nodes --context=us-west-prod
kubectl get nodes --context=us-east-prod

# 2. 更新全局负载均衡器（移除故障区域）
# 使用AWS Route 53 / GCP Cloud DNS
aws route53 change-resource-record-sets \
  --hosted-zone-id Z123456 \
  --change-batch file://remove-failing-region.json

# 3. 扩容健康区域容量
kubectl scale deployment --all --replicas=6 \
  --context=us-east-prod -n edgelink-prod

# 4. 验证容量充足
kubectl top nodes --context=us-east-prod
kubectl top pods -n edgelink-prod --context=us-east-prod

# 5. 监控流量迁移
watch "kubectl get hpa -n edgelink-prod --context=us-east-prod"
```

---

## 5. 回滚验证

### 5.1 健康检查清单

```bash
#!/bin/bash
# scripts/rollback-verification.sh

set -e

ENVIRONMENT=${1:-production}
API_URL="https://api.edgelink.${ENVIRONMENT}"

echo "🔍 验证回滚结果..."

# 1. 检查所有Pod运行正常
echo "1. 检查Pod状态..."
kubectl get pods -n edgelink-${ENVIRONMENT} | grep -v Running && exit 1 || echo "✅ All pods running"

# 2. 健康检查端点
echo "2. 健康检查端点..."
curl -sf ${API_URL}/health | jq '.status' | grep -q "healthy" || { echo "❌ Health check failed"; exit 1; }
echo "✅ Health check passed"

# 3. 设备注册功能
echo "3. 测试设备注册..."
DEVICE_RESPONSE=$(curl -sf -X POST ${API_URL}/api/v1/device/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-device-rollback-verify",
    "pre_shared_key": "test-psk-key",
    "public_key": "test-public-key"
  }')
echo $DEVICE_RESPONSE | jq '.device_id' || { echo "❌ Device registration failed"; exit 1; }
echo "✅ Device registration working"

# 4. 数据库连接
echo "4. 验证数据库连接..."
kubectl exec -it postgres-0 -n edgelink-${ENVIRONMENT} -- \
  psql -U edgelink -d edgelink_${ENVIRONMENT} -c "SELECT 1;" > /dev/null || { echo "❌ Database connection failed"; exit 1; }
echo "✅ Database connection OK"

# 5. Redis连接
echo "5. 验证Redis连接..."
kubectl exec -it redis-master-0 -n edgelink-${ENVIRONMENT} -- \
  redis-cli PING | grep -q PONG || { echo "❌ Redis connection failed"; exit 1; }
echo "✅ Redis connection OK"

# 6. 指标端点
echo "6. 检查Prometheus指标..."
curl -sf ${API_URL}/metrics | grep -q "http_requests_total" || { echo "❌ Metrics endpoint failed"; exit 1; }
echo "✅ Metrics endpoint working"

# 7. 响应时间
echo "7. 测试API响应时间..."
RESPONSE_TIME=$(curl -o /dev/null -s -w '%{time_total}\n' ${API_URL}/health)
if (( $(echo "$RESPONSE_TIME > 0.5" | bc -l) )); then
  echo "⚠️ Warning: Response time ${RESPONSE_TIME}s > 500ms"
else
  echo "✅ Response time ${RESPONSE_TIME}s < 500ms"
fi

# 8. 错误率
echo "8. 检查错误率..."
ERROR_RATE=$(kubectl exec -it prometheus-0 -n monitoring -- \
  promtool query instant http://localhost:9090 \
  'rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])' \
  | grep -oP '\d+\.\d+' | head -1)
if (( $(echo "$ERROR_RATE > 0.01" | bc -l) )); then
  echo "⚠️ Warning: Error rate ${ERROR_RATE} > 1%"
else
  echo "✅ Error rate ${ERROR_RATE} < 1%"
fi

echo ""
echo "✅ 回滚验证通过！"
echo "📊 建议观察期: 继续监控15分钟"
```

### 5.2 Smoke测试

```bash
#!/bin/bash
# scripts/smoke-test.sh

API_URL=${1:-https://api.edgelink.production}

echo "🚬 运行Smoke Tests..."

# Test 1: Health endpoint
curl -sf ${API_URL}/health || { echo "❌ Test 1 failed"; exit 1; }
echo "✅ Test 1: Health check passed"

# Test 2: Device registration
DEVICE_ID=$(curl -sf -X POST ${API_URL}/api/v1/device/register \
  -H "Content-Type: application/json" \
  -d '{"name":"smoke-test-device","pre_shared_key":"test-key","public_key":"test-pub-key"}' \
  | jq -r '.device_id')
[[ -n "$DEVICE_ID" ]] || { echo "❌ Test 2 failed"; exit 1; }
echo "✅ Test 2: Device registration passed (ID: $DEVICE_ID)"

# Test 3: Get device config
curl -sf ${API_URL}/api/v1/device/${DEVICE_ID}/config \
  -H "Authorization: Device ${DEVICE_ID}" > /dev/null || { echo "❌ Test 3 failed"; exit 1; }
echo "✅ Test 3: Get device config passed"

# Test 4: Submit metrics
curl -sf -X POST ${API_URL}/api/v1/device/${DEVICE_ID}/metrics \
  -H "Content-Type: application/json" \
  -H "Authorization: Device ${DEVICE_ID}" \
  -d '{"bandwidth_tx":1000,"bandwidth_rx":2000,"latency_ms":50}' > /dev/null || { echo "❌ Test 4 failed"; exit 1; }
echo "✅ Test 4: Submit metrics passed"

echo "✅ All smoke tests passed!"
```

---

## 6. 常见失败场景处理

### 6.1 配置不兼容导致回滚失败

**症状**: 回滚后服务启动失败，日志显示"unknown field"或"missing required field"

**解决方案**:
```bash
# 1. 从ConfigMap备份恢复旧配置
kubectl get configmap edge-link-config -n edgelink-prod -o yaml > /tmp/current_config.yaml

# 2. 恢复到已知良好配置
kubectl apply -f /backup/configmaps/edge-link-config-v0.1.5.yaml -n edgelink-prod

# 3. 重启所有Pod以加载新配置
kubectl rollout restart deployment --all -n edgelink-prod

# 4. 等待Pod就绪
kubectl rollout status deployment/edge-link-production-api-gateway -n edgelink-prod
```

### 6.2 数据库迁移无法回滚

**症状**: `migrate down` 失败，错误信息 "column does not exist"

**解决方案**:
```bash
# 1. 手动编写回滚SQL
cat > /tmp/manual_rollback.sql <<EOF
-- 恢复被删除的列
ALTER TABLE devices ADD COLUMN IF NOT EXISTS legacy_field VARCHAR(255);

-- 恢复数据（如果有备份）
UPDATE devices SET legacy_field = old_devices.legacy_field
FROM old_devices WHERE devices.id = old_devices.id;

-- 更新迁移版本
UPDATE schema_migrations SET version = '0001_previous_migration';
EOF

# 2. 执行手动回滚
kubectl exec -it postgres-0 -n edgelink-prod -- \
  psql -U edgelink -d edgelink_prod -f /tmp/manual_rollback.sql

# 3. 验证数据完整性
kubectl exec -it postgres-0 -n edgelink-prod -- \
  psql -U edgelink -d edgelink_prod -c "\d devices"
```

### 6.3 PVC数据损坏

**症状**: Pod CrashLoopBackOff，日志显示 "cannot mount volume"

**解决方案**:
```bash
# 1. 检查PVC状态
kubectl get pvc -n edgelink-prod
kubectl describe pvc postgres-data-postgres-0 -n edgelink-prod

# 2. 从快照恢复（如果使用CSI驱动）
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-data-postgres-0-restored
  namespace: edgelink-prod
spec:
  dataSource:
    name: postgres-snapshot-daily-$(date -d "yesterday" +%Y%m%d)
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
EOF

# 3. 更新StatefulSet使用新PVC
kubectl patch statefulset postgres -n edgelink-prod \
  --type=json \
  -p='[{"op": "replace", "path": "/spec/volumeClaimTemplates/0/metadata/name", "value": "postgres-data-postgres-0-restored"}]'

# 4. 重启StatefulSet
kubectl rollout restart statefulset postgres -n edgelink-prod
```

---

## 7. 事后分析

### 7.1 回滚记录模板

```markdown
# 回滚事件报告 - YYYY-MM-DD

## 基本信息
- **事件ID**: INC-2025-1020-001
- **触发时间**: 2025-10-20 14:35 UTC
- **回滚完成时间**: 2025-10-20 14:52 UTC
- **总停机时间**: 17分钟
- **影响范围**: 生产环境 - API Gateway
- **责任人**: @ops-team

## 触发原因
- [ ] 错误率超阈值 (5xx > 5%)
- [ ] 性能下降 (P95 > 500ms)
- [ ] Pod CrashLoop
- [ ] 数据一致性问题
- [ ] 其他: ___________

## 回滚步骤
1. 14:35 - 检测到5xx错误率飙升至15%
2. 14:37 - 确认回滚决策，通知#incidents频道
3. 14:38 - 执行 `helm rollback edge-link-production 12`
4. 14:45 - 等待Pod重启完成
5. 14:50 - 运行smoke tests，全部通过
6. 14:52 - 确认错误率恢复正常，解除警报

## RTO/RPO达成情况
- **RTO目标**: 30分钟
- **RTO实际**: 17分钟 ✅
- **RPO目标**: 15分钟
- **RPO实际**: 0分钟 (无数据丢失) ✅

## 根本原因分析
新版本中引入的数据库连接池配置错误导致连接泄漏，在高负载下耗尽数据库连接。

## 改进措施
- [ ] 在staging环境进行3小时负载测试
- [ ] 添加连接池监控指标到Grafana
- [ ] 更新发布检查清单，要求验证资源限制
- [ ] 编写回归测试覆盖连接池场景

## 经验教训
1. 数据库配置变更需要更严格的测试
2. 观察期应延长至15分钟（当前5分钟）
3. 回滚SOP有效，团队响应及时
```

### 7.2 季度演练计划

```markdown
# 2025 Q4 灾难恢复演练计划

## 10月演练: 应用回滚
- **日期**: 2025-10-25 10:00 AM (非业务高峰)
- **场景**: Helm chart升级失败，需回滚到上一版本
- **参与者**: DevOps团队 (3人)
- **预期RTO**: < 15分钟
- **验收标准**: smoke tests通过，错误率< 1%

## 11月演练: 数据库恢复
- **日期**: 2025-11-15 10:00 AM
- **场景**: 数据库迁移失败导致数据损坏，从备份恢复
- **参与者**: DevOps + DBA (4人)
- **预期RTO**: < 30分钟
- **预期RPO**: < 15分钟
- **验收标准**: 数据完整性验证通过，应用连接正常

## 12月演练: 完整DR切换
- **日期**: 2025-12-20 10:00 AM
- **场景**: 主数据中心故障，切换到DR集群
- **参与者**: 全体工程团队 (10人)
- **预期RTO**: < 30分钟
- **预期RPO**: < 15分钟
- **验收标准**:
  - DNS切换成功
  - 所有服务恢复
  - 客户端重连成功
  - 数据一致性验证通过

## 演练后行动
- [ ] 记录实际RTO/RPO
- [ ] 识别流程瓶颈
- [ ] 更新SOP文档
- [ ] 团队复盘会议
- [ ] 改进措施跟踪
```

---

## 附录

### A. 紧急联系人

| 角色 | 姓名 | Slack | 电话 | 备用联系 |
|------|------|-------|------|----------|
| DevOps Lead | Alice Zhang | @alice | +1-xxx-xxx-1234 | alice@edgelink.com |
| SRE On-call | Bob Chen | @bob | +1-xxx-xxx-5678 | bob@edgelink.com |
| DBA | Carol Li | @carol | +1-xxx-xxx-9012 | carol@edgelink.com |
| CTO | David Wang | @david | +1-xxx-xxx-3456 | david@edgelink.com |

### B. 关键系统凭证位置

- Kubernetes kubeconfig: 1Password Vault "Production K8s"
- Database密码: Vault secret `edgelink/prod/postgres`
- Cloudflare API Token: Vault secret `edgelink/cloudflare`
- GitHub Container Registry: 使用GITHUB_TOKEN (in Actions)

### C. 备份位置

- 数据库备份: S3 `s3://edgelink-backups/postgres/production/`
- Velero备份: S3 `s3://edgelink-backups/velero/production/`
- ConfigMap备份: Git repo `edgelink-infra/config-backups/`
- Helm values历史: Git repo `edgelink-infra/helm-values/`

---

**版本历史**:
- v1.0 (2025-10-20): 初始版本，包含K8s回滚、数据库恢复、DR切换
- v1.1 (待定): 计划添加客户端版本兼容性回滚流程

**审核人**: @ops-team
**下次审核日期**: 2025-11-20
