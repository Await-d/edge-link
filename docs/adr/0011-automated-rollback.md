# ADR-0011: 基于Prometheus实现自动回滚

**状态**: Accepted

**日期**: 2025-10-20

**决策者**: DevOps Team, SRE Team

**相关ADR**: 
- [ADR-0007: 使用Kubernetes进行容器编排](0007-kubernetes-deployment.md)
- [ADR-0010: 使用Prometheus + Grafana进行监控](0010-prometheus-monitoring.md)

---

## 上下文和问题陈述

EdgeLink在生产环境中经历了几次部署失败事件，导致服务中断：

1. **2025-09-15事件**：新版本引入内存泄漏，15分钟内内存使用率从40%飙升至95%，导致OOM kill
2. **2025-09-28事件**：数据库连接池配置错误，高负载下5xx错误率达到12%，持续20分钟才被发现
3. **2025-10-10事件**：依赖库升级导致P95延迟从150ms增至800ms，影响用户体验

这些事件的共同问题：
- **发现延迟**：依赖人工监控Dashboard，平均发现时间10-15分钟
- **决策延迟**：需要团队讨论确认是否回滚，平均5-10分钟
- **执行延迟**：手动执行kubectl命令，2-5分钟
- **总MTTR**：平均25分钟，远超15分钟的SLA要求

我们需要一个自动化系统，能够：
- 在部署后3分钟内检测到异常
- 无需人工干预自动触发回滚
- 将MTTR降低到5分钟以内
- 避免误回滚（false positive）

## 决策驱动因素

- **SLA要求**：99.9%可用性，MTTR < 15分钟
- **人力成本**：减少7x24值班压力
- **用户体验**：最小化故障影响时间窗口
- **合规要求**：企业客户要求自动化故障恢复能力
- **团队规模**：小团队无法持续人工监控
- **风险控制**：快速失败优于长时间降级

## 考虑的方案

### 方案1: Prometheus + Alertmanager + Kubernetes自动回滚

基于现有Prometheus监控栈，通过PrometheusRule定义告警，Alertmanager路由到webhook，webhook执行kubectl rollout undo。

**优点**:
- **已有基础设施**：复用现有Prometheus + Alertmanager
- **低复杂度**：仅需添加PrometheusRule和webhook receiver
- **Kubernetes原生**：使用kubectl rollout undo，稳定可靠
- **灵活配置**：告警规则、阈值可调
- **审计友好**：所有操作记录在Kubernetes Events
- **干预能力**：可随时暂停/恢复自动回滚

**缺点**:
- **需要额外webhook**：需要开发小型webhook服务
- **告警精确度依赖指标**：需要精心设计PromQL查询
- **无法预测式回滚**：仅能基于已发生的问题回滚

**成本估算**:
- 开发成本：低（2-3天开发webhook和Helm模板）
- 运维成本：低（复用现有监控栈）
- 许可证成本：$0

### 方案2: Flagger金丝雀部署 + 自动回滚

使用Flagger进行渐进式部署，自动评估指标并回滚。

**优点**:
- **成熟方案**：Weaveworks官方支持，社区活跃
- **渐进式部署**：流量逐步切换，降低爆炸半径
- **多种策略**：支持Canary、Blue/Green、A/B Testing
- **丰富集成**：支持Istio、Linkerd、App Mesh
- **可视化**：Flagger UI展示部署进度

**缺点**:
- **引入Service Mesh**：需要Istio/Linkerd，架构复杂度大幅提升
- **学习曲线**：团队需要学习Flagger + Service Mesh
- **资源开销**：Sidecar增加20-30% CPU/内存消耗
- **调试困难**：Service Mesh故障排查复杂
- **过度设计**：EdgeLink当前规模不需要Service Mesh全部能力

**成本估算**:
- 开发成本：高（2-3周迁移到Service Mesh）
- 运维成本：高（Service Mesh运维复杂）
- 许可证成本：$0（开源）

### 方案3: ArgoCD Sync Waves + Health Checks

使用ArgoCD的Progressive Delivery能力进行自动回滚。

**优点**:
- **GitOps原生**：与GitOps workflow集成良好
- **声明式配置**：回滚策略在Git中版本化
- **强大的健康检查**：支持自定义Health Assessment
- **可视化**：ArgoCD UI清晰展示同步状态

**缺点**:
- **需要ArgoCD**：引入新的部署工具
- **与Helm冲突**：ArgoCD与Helm Release管理有overlap
- **回滚延迟**：基于同步周期，响应较慢
- **复杂度**：GitOps全流程需要团队转变

**成本估算**:
- 开发成本：中（1-2周）
- 运维成本：中
- 许可证成本：$0

## 决策结果

**选择的方案**: Prometheus + Alertmanager + Kubernetes自动回滚

**核心理由**:

1. **最小化改动**：复用现有Prometheus监控栈，无需引入Service Mesh或ArgoCD
2. **快速实施**：2-3天即可上线，立即见效
3. **简单可控**：逻辑清晰，故障排查简单
4. **精确控制**：可针对不同服务配置不同阈值和策略
5. **成本最优**：无额外资源开销
6. **团队熟悉**：团队已掌握Prometheus和Kubernetes

**为什么不选Flagger**:
虽然Flagger功能强大，但EdgeLink当前阶段不需要Service Mesh的全部能力。引入Service Mesh会增加40%+的系统复杂度，而我们的核心问题（快速回滚）可以用更简单的方案解决。

未来如果需要更复杂的流量管理（如A/B测试、蓝绿部署），可以考虑逐步迁移到Flagger。

## 决策后果

### 积极影响

- **MTTR大幅降低**：
  - 从平均25分钟降至<5分钟（80%降幅）
  - 检测延迟：15分钟 → 2分钟
  - 决策延迟：10分钟 → 0分钟（自动）
  - 执行延迟：5分钟 → 2分钟

- **服务可用性提升**：
  - 故障影响时间窗口减少80%
  - 预计可用性从99.5%提升至99.9%

- **人力成本降低**：
  - 减少70%的紧急oncall响应
  - 工程师可专注于根因分析而非救火

- **风险可控**：
  - Dry run模式验证告警规则
  - 可随时禁用自动回滚
  - 详细的审计日志

### 消极影响

- **误回滚风险**：
  - 如果告警规则配置不当，可能误回滚正常部署
  - 缓解措施：Dry run测试 + 宽松阈值 + 观察期

- **复杂性增加**：
  - 增加PrometheusRule、Webhook、RBAC配置维护负担
  - 缓解措施：完善文档和故障排查指南

- **依赖Prometheus**：
  - 如果Prometheus故障，自动回滚失效
  - 缓解措施：Prometheus自身高可用部署

### 风险与缓解措施

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 误回滚导致频繁部署失败 | High | Medium | 1. 初期设置宽松阈值 2. Dry run模式验证 3. 每次回滚发送Slack通知 |
| 告警风暴导致过度回滚 | Medium | Low | 1. Alertmanager抑制规则 2. 回滚冷却期（最小间隔15分钟） |
| Prometheus故障导致无法回滚 | High | Low | 1. Prometheus HA部署 2. 保留手动回滚能力 3. 监控Prometheus自身健康 |
| 回滚后问题仍存在 | Medium | Low | 1. 保留多个历史版本 2. 可回滚到任意版本 3. 通知oncall介入 |

### 技术债务

**webhook实现简化**:
当前webhook使用简单的shell脚本实现。在流量增长后，可能需要重构为Go服务以获得更好的性能和错误处理。

**计划**: 在webhook QPS > 10时重构为Go实现。

## 实现说明

### 行动项

- [x] 设计PrometheusRule定义（3类关键告警 + 7类警告告警）
- [x] 实现rollback webhook Deployment
- [x] 配置RBAC权限（ServiceAccount + Role）
- [x] 创建回滚脚本（auto-rollback.sh）
- [x] 编写Helm chart模板
- [x] 更新values.yaml配置选项
- [x] 集成Alertmanager路由配置
- [ ] Staging环境Dry run测试（3天）
- [ ] 生产环境启用（分阶段）
- [ ] 编写运维手册和故障排查指南
- [ ] 团队培训

### 时间线

- **决策日期**: 2025-10-20
- **开发完成**: 2025-10-20
- **Staging测试**: 2025-10-21 ~ 2025-10-23
- **生产环境启用**: 2025-10-24（Dry run）
- **全面启用**: 2025-10-27

### 成功指标

- MTTR < 5分钟（目标：3分钟）
- 自动回滚成功率 > 95%
- 误回滚率 < 2%
- 服务可用性 > 99.9%
- Oncall响应次数降低 > 60%

## 验证与监控

### 告警规则验证

```bash
# Dry run模式测试
helm upgrade edge-link ./infrastructure/helm/edge-link-control-plane \
  --set monitoring.autoRollback.dryRun=true \
  --set monitoring.thresholds.errorRate=0.02  # 临时降低阈值触发告警

# 观察日志
kubectl logs -n edgelink -l app.kubernetes.io/component=rollback-webhook --tail=100
```

### 监控回滚操作

```promql
# 回滚频率
rate(edgelink_rollback_total[1h])

# 回滚成功率
edgelink_rollback_success_total / edgelink_rollback_total

# 回滚触发原因分布
edgelink_rollback_total by (reason)

# 平均回滚时长
histogram_quantile(0.95, edgelink_rollback_duration_seconds_bucket)
```

### Grafana Dashboard

创建专门的"自动回滚"Dashboard，展示：
- 回滚历史时间线
- 回滚原因分布
- 回滚成功率趋势
- MTTR趋势

## 参考资料

- [Prometheus Alerting官方文档](https://prometheus.io/docs/alerting/latest/)
- [Kubernetes Rollout文档](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-back-a-deployment)
- [Google SRE Book - Automated Rollback](https://sre.google/sre-book/automation-at-google/)
- [Netflix Spinnaker - Automated Canary Analysis](https://netflixtechblog.com/automated-canary-analysis-at-netflix-with-kayenta-3260bc7acc69)
- [EdgeLink监控文档](../../infrastructure/monitoring/README.md)
- [EdgeLink灾难恢复手册](../disaster-recovery.md)

## 审核历史

| 日期 | 变更 | 作者 |
|------|------|------|
| 2025-10-20 | 初始创建并实施 | DevOps Team |
| 2025-10-20 | 状态更新为Accepted | SRE Lead |

---

## 附录

### 关键告警规则

```yaml
# 高错误率（自动回滚）
- alert: EdgeLinkHighErrorRate
  expr: |
    (sum(rate(http_requests_total{status=~"5.."}[5m])) by (service)
     / sum(rate(http_requests_total[5m])) by (service)) > 0.05
  for: 2m
  labels:
    severity: critical
    auto_rollback: "true"
  annotations:
    summary: "High error rate in {{ $labels.service }}"

# 高延迟（自动回滚）
- alert: EdgeLinkHighLatency
  expr: |
    histogram_quantile(0.95,
      sum(rate(http_request_duration_seconds_bucket[5m])) by (service, le)
    ) > 0.5
  for: 3m
  labels:
    severity: critical
    auto_rollback: "true"

# Pod崩溃（自动回滚）
- alert: EdgeLinkPodCrashLooping
  expr: |
    rate(kube_pod_container_status_restarts_total{pod=~"edgelink-.*"}[15m]) > 0
  for: 1m
  labels:
    severity: critical
    auto_rollback: "true"
```

### 自动回滚流程图

```
┌─────────────────┐
│  Deployment     │
│  (New Version)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Prometheus     │
│  (Monitor)      │
└────────┬────────┘
         │ Error Rate > 5%
         │ for 2 minutes
         ▼
┌─────────────────┐
│ PrometheusRule  │
│  (Evaluate)     │
└────────┬────────┘
         │ Alert Firing
         ▼
┌─────────────────┐
│ Alertmanager    │
│  (Route)        │
└────────┬────────┘
         │ auto_rollback="true"
         ▼
┌─────────────────┐
│ Rollback Webhook│
│  (Trigger)      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Kubernetes    │
│ kubectl rollout │
│      undo       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Deployment     │
│ (Previous Ver)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Verify Health  │
│  (2 min)        │
└────────┬────────┘
         │
         ▼
     Success ✅
```

### 分阶段启用计划

**阶段1: Dry Run（3天）**
```yaml
monitoring:
  autoRollback:
    enabled: true
    dryRun: true  # 仅日志，不实际回滚
```
观察哪些告警会被触发，调整阈值。

**阶段2: 单服务启用（3天）**
```yaml
# 仅在非关键服务（如background-worker）启用
monitoring:
  autoRollback:
    enabled: true
    dryRun: false
```

**阶段3: 全面启用**
```yaml
# 所有服务启用，但设置宽松阈值
monitoring:
  thresholds:
    errorRate: 0.08  # 8%（初期宽松）
    p95Latency: 1.0  # 1秒
```

**阶段4: 收紧阈值**
```yaml
# 根据历史数据优化阈值
monitoring:
  thresholds:
    errorRate: 0.05  # 5%
    p95Latency: 0.5  # 500ms
```
