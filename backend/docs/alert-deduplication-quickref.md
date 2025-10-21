# 告警去重机制 - 快速参考

## 核心概念

### 去重键
```
设备ID + 告警类型 → SHA256 → 16字符Hash
```

### 时间窗口
- **去重窗口**: 30分钟（同一告警合并）
- **静默期**: 5分钟（解决后冷却）

### 升级机制
- **阈值**: 10次
- **升级路径**: low → medium → high → critical

---

## 配置环境变量

```bash
# 必需配置
ALERT_DEDUPE_WINDOW=30m
ALERT_SILENT_PERIOD=5m
ALERT_ESCALATION_THRESHOLD=10

# 可选配置
ALERT_CHECK_INTERVAL=1m
ALERT_DEVICE_OFFLINE_THRESHOLD=5m
ALERT_HIGH_LATENCY_THRESHOLD=200
```

---

## Redis键结构

```
alert:dedupe:{hash}   → 去重信息（TTL: 60分钟）
alert:lock:{hash}     → 分布式锁（TTL: 5秒）
alert:silent:{hash}   → 静默标记（TTL: 5分钟）
```

---

## 数据库字段

新增3个字段到`alerts`表：
- `occurrence_count` (INTEGER, 默认1) - 出现次数
- `first_seen_at` (TIMESTAMP) - 首次出现时间
- `last_seen_at` (TIMESTAMP) - 最后出现时间

---

## API方法

### DeduplicationManager

```go
// 检查是否应该创建新告警
ShouldCreateAlert(ctx, key, severity) (shouldCreate, existingID, shouldEscalate, err)

// 记录告警
RecordAlert(ctx, alertID, key, severity, isNew)

// 自动解决检查
IsInSilentPeriod(ctx, key) (bool, error)
SetSilentPeriod(ctx, key) error
```

### AlertGenerator

```go
// 生成告警（含去重逻辑）
GenerateAlert(ctx, issue) (*Alert, error)

// 自动解决设备告警
ResolveDeviceAlerts(ctx, deviceID, alertType) error
```

### AlertRepository

```go
// 更新出现次数
UpdateOccurrence(ctx, id, count) error

// 提升严重程度
EscalateSeverity(ctx, id, severity) error

// 按设备和类型解决
ResolveByDeviceAndType(ctx, deviceID, alertType) error
```

---

## 工作流程

### 1. 首次告警
```
检测问题 → 生成AlertKey → 获取锁 → 查Redis（无）
→ 创建DB记录 → 保存Redis → 发送通知
```

### 2. 重复告警
```
检测问题 → 生成AlertKey → 获取锁 → 查Redis（有）
→ 检查窗口（在内）→ 更新count → 检查升级 → 更新Redis
```

### 3. 告警升级
```
count >= 10 → 提升severity → 标记escalated → 发送升级通知
```

### 4. 自动解决
```
设备恢复 → 解决DB告警 → 删除Redis → 设置静默期
```

---

## 部署检查清单

- [ ] 运行数据库迁移 `000006_add_alert_deduplication_fields.up.sql`
- [ ] 配置环境变量（参考`.env.example`）
- [ ] 验证Redis连接
- [ ] 重启alert-service
- [ ] 检查日志中的"Alert check loop started"
- [ ] 验证Redis键：`redis-cli KEYS "alert:*"`

---

## 监控指标

建议监控：
- `alert_deduplication_hits_total` - 去重命中次数
- `alert_escalation_total` - 告警升级次数
- `alert_auto_resolved_total` - 自动解决次数
- `redis_lock_acquisition_duration_seconds` - 锁获取耗时

---

## 故障排查

### 重复告警仍然创建
- 检查`ALERT_DEDUPE_WINDOW`配置
- 验证Redis连接
- 检查日志中的"Failed to check deduplication"

### 告警未自动解决
- 确认调用了`ResolveDeviceAlerts()`
- 检查设备在线状态检测逻辑

### Redis内存占用过高
- 减小`ALERT_DEDUPE_WINDOW`
- 检查TTL是否正确设置
- 清理过期键：`redis-cli SCAN 0 MATCH alert:* COUNT 1000`

---

## 性能调优

### 低流量（<100设备）
```bash
ALERT_CHECK_INTERVAL=1m
ALERT_DEDUPE_WINDOW=30m
```

### 中等流量（100-1000设备）
```bash
ALERT_CHECK_INTERVAL=2m
ALERT_DEDUPE_WINDOW=15m
```

### 高流量（>1000设备）
```bash
ALERT_CHECK_INTERVAL=5m
ALERT_DEDUPE_WINDOW=10m
REDIS_POOL_SIZE=50
```

---

## 文件位置

```
backend/
├── cmd/alert-service/
│   ├── internal/deduplication/manager.go        # 去重管理器
│   ├── internal/generator/alert_generator.go    # 告警生成器
│   ├── main.go                                   # 服务入口
│   └── .env.example                              # 配置示例
├── internal/
│   ├── config/config.go                         # 配置定义
│   ├── domain/alert.go                          # 告警实体
│   └── repository/alert_repo.go                 # 告警仓储
├── migrations/
│   ├── 000006_add_alert_deduplication_fields.up.sql
│   └── 000006_add_alert_deduplication_fields.down.sql
└── docs/
    ├── alert-deduplication-design.md            # 详细设计文档
    └── alert-deduplication-architecture.md      # 架构图
```

---

## 快速测试

### 1. 创建测试告警
```bash
# 模拟设备离线（通过API或直接修改数据库）
curl -X POST http://localhost:8080/api/v1/test/trigger-offline \
  -d '{"device_id": "test-device-1"}'
```

### 2. 检查Redis
```bash
redis-cli
> KEYS alert:dedupe:*
> GET alert:dedupe:{hash}
> TTL alert:dedupe:{hash}
```

### 3. 查看数据库
```sql
SELECT id, occurrence_count, first_seen_at, last_seen_at, severity
FROM alerts
WHERE device_id = 'test-device-1'
ORDER BY created_at DESC
LIMIT 5;
```

### 4. 验证升级
```bash
# 等待告警出现10次以上
SELECT id, occurrence_count, severity
FROM alerts
WHERE occurrence_count >= 10;
```

---

## 常见问题

**Q: 去重窗口和检查间隔有什么区别？**
A: 检查间隔是多久检查一次系统健康，去重窗口是多久内的相同告警会被合并。

**Q: 静默期何时使用？**
A: 设备恢复在线后自动设置，避免设备频繁上下线造成告警风暴。

**Q: 如何手动清除去重信息？**
A: `redis-cli DEL alert:dedupe:{hash}` 或调用`RemoveDedupeInfo()`

**Q: 升级是否可逆？**
A: 当前实现不支持降级，只支持单向升级。如需降级需手动修改。

**Q: 多实例部署会冲突吗？**
A: 不会，使用Redis分布式锁协调多实例。

---

## 联系与支持

- 详细设计: `/backend/docs/alert-deduplication-design.md`
- 架构图: `/backend/docs/alert-deduplication-architecture.md`
- 配置示例: `/backend/cmd/alert-service/.env.example`
