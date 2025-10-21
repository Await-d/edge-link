# 通知规则引擎设计文档

## 1. 概述

通知规则引擎是告警服务的核心组件，提供灵活的规则匹配和通知路由能力。系统支持基于多维度条件的规则匹配、多种通知渠道、速率限制、告警升级和静默规则等高级特性。

## 2. 架构设计

### 2.1 核心组件

```
┌─────────────────────────────────────────────────────────────┐
│                     Rule Engine                              │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Parser  │  │ Matcher  │  │ Executor │  │  Engine  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│       │             │              │              │          │
│       ▼             ▼              ▼              ▼          │
│  规则解析      条件匹配       动作执行       引擎核心       │
│  ────────      ────────       ────────       ────────       │
│  YAML解析      逻辑组合       多渠道发送     规则管理       │
│  验证规则      时间范围       重试机制       速率限制       │
│  默认值        正则匹配       错误处理       告警升级       │
│                元数据匹配                     热重载         │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
          ┌─────────────────────────────────┐
          │   NotificationScheduler          │
          │   (集成规则引擎)                │
          └─────────────────────────────────┘
```

### 2.2 数据流

```
Alert Generated
    │
    ▼
┌───────────────┐
│ Rule Engine   │
│ Process()     │
└───────┬───────┘
        │
        ▼
┌──────────────────┐
│ Match Rules      │◄─── Rule Set (YAML)
│ (Priority Order) │
└─────────┬────────┘
          │
          ▼
    ┌─────────┐
    │ Matched │
    │ Rules   │
    └────┬────┘
         │
         ├─────────────────┬──────────────────┬──────────────
         ▼                 ▼                  ▼
    ┌────────┐      ┌──────────┐      ┌────────────┐
    │ Check  │      │  Check   │      │  Execute   │
    │ Silence│      │Rate Limit│      │  Actions   │
    └───┬────┘      └────┬─────┘      └──────┬─────┘
        │                │                    │
        └────────────────┴────────────────────┘
                         │
                         ▼
                  ┌──────────────┐
                  │   Notifiers  │
                  │ (Email, Slack│
                  │  PagerDuty, │
                  │  DingTalk...) │
                  └──────────────┘
```

## 3. 核心功能

### 3.1 规则解析 (Parser)

**职责**:
- 从YAML文件或字节流解析规则集
- 验证规则完整性和正确性
- 初始化默认值

**验证器链**:
1. `BasicValidator`: 基本字段验证
2. `ConditionsValidator`: 条件逻辑验证
3. `ActionsValidator`: 动作配置验证
4. `TimeRangeValidator`: 时间范围格式验证

**支持的数据源**:
- 文件系统 (YAML)
- 字节数组 (用于动态配置)
- 未来可扩展: 数据库、配置中心

### 3.2 条件匹配 (Matcher)

**匹配维度**:
- `severity`: 告警严重程度 (critical/high/medium/low)
- `alert_types`: 告警类型 (device_offline/high_latency/etc)
- `device_ids`: 特定设备ID列表
- `device_tags`: 设备标签 (production/staging/critical-infrastructure)
- `time_range`: 时间范围 (工作时间/非工作时间)
- `message_match`: 消息内容正则匹配
- `metadata`: 告警元数据键值对匹配

**逻辑组合**:
- `all_of`: AND逻辑 (所有条件必须满足)
- `any_of`: OR逻辑 (任一条件满足即可)
- `none_of`: NOT逻辑 (所有条件都不满足)

**示例**:
```yaml
conditions:
  all_of:
    - severity: [critical]
    - any_of:
        - device_tags: [production]
        - device_ids: ["device-123"]
  none_of:
    - message_match: ".*test.*"
```

### 3.3 动作执行 (Executor)

**支持的通知渠道**:

| 渠道类型 | 配置参数 | 特性 |
|---------|---------|------|
| Email | recipients | SMTP邮件发送 |
| Webhook | url | 通用HTTP POST |
| Slack | webhook_url, channel | Slack消息推送 |
| PagerDuty | service_key | 事件创建和升级 |
| DingTalk | webhook_url, at_mobiles | 钉钉群机器人 |
| WeChat | webhook_url | 企业微信群机器人 |
| Telegram | bot_token, chat_id | Telegram Bot API |
| Custom | url, method, body | 自定义HTTP请求 |

**重试策略**:
```yaml
retry_policy:
  max_retries: 3           # 最大重试次数
  retry_delay: 5s          # 初始重试延迟
  backoff_rate: 2.0        # 指数退避倍率
```

### 3.4 规则引擎 (Engine)

**核心特性**:

1. **速率限制**
   ```yaml
   rate_limit:
     max_notifications: 10   # 最大通知数
     window: 1h              # 时间窗口
     scope: per_rule         # 范围: global/per_device/per_rule
   ```

2. **告警升级**
   ```yaml
   escalation:
     enabled: true
     wait_duration: 15m      # 未确认等待时长
     escalate_to:            # 升级后的通知动作
       - type: pagerduty
     repeat_interval: 30m    # 重复通知间隔
     max_repeat: 3           # 最大重复次数
   ```

3. **静默规则**
   ```yaml
   silence:
     enabled: true
     time_ranges:
       - start: "02:00"
         end: "04:00"
         timezone: "Asia/Shanghai"
         weekdays: [Sunday]
   ```

4. **规则热重载**
   - 定期检查文件变更
   - 无需重启服务
   - 原子性更新规则集

## 4. 数据结构

### 4.1 核心类型

```go
// Rule 通知规则
type Rule struct {
    ID          string
    Name        string
    Priority    int              // 数字越小优先级越高
    Conditions  Conditions       // 匹配条件
    Actions     []Action         // 通知动作
    RateLimit   *RateLimit       // 速率限制
    Escalation  *Escalation      // 告警升级
    Silence     *SilenceRule     // 静默规则
}

// Conditions 匹配条件
type Conditions struct {
    Severity     []Severity
    AlertTypes   []AlertType
    DeviceIDs    []string
    DeviceTags   []string
    TimeRange    *TimeRange
    MessageMatch string
    Metadata     map[string]string
    AllOf        []Conditions    // AND逻辑
    AnyOf        []Conditions    // OR逻辑
    NoneOf       []Conditions    // NOT逻辑
}

// Action 通知动作
type Action struct {
    Type        ActionType
    Config      map[string]interface{}
    RetryPolicy *RetryPolicy
}
```

### 4.2 匹配上下文

```go
type MatchContext struct {
    Alert     *domain.Alert
    Device    *domain.Device
    Timestamp time.Time
    Metadata  map[string]interface{}
}
```

### 4.3 执行上下文

```go
type ExecutionContext struct {
    Alert         *domain.Alert
    Device        *domain.Device
    Rule          *Rule
    Timestamp     time.Time
    PreviousTries int
}
```

## 5. 性能考虑

### 5.1 优化策略

1. **正则表达式缓存**
   - 编译后的正则表达式缓存在Matcher中
   - 避免重复编译开销

2. **并发处理**
   - 规则匹配和执行相互独立
   - 多个动作可并行执行

3. **速率限制实现**
   - 基于滑动窗口的token bucket算法
   - O(n)时间复杂度，n为窗口内token数

4. **规则优先级排序**
   - 匹配后按优先级排序
   - 支持短路执行（可选）

### 5.2 可扩展性

1. **规则数据源**
   - 当前: 文件系统
   - 未来: PostgreSQL、Redis、Etcd

2. **通知渠道**
   - 插件化架构
   - 通过`ActionType`注册新渠道

3. **分布式支持**
   - 速率限制可迁移到Redis
   - 升级状态可持久化到数据库

## 6. 安全考虑

### 6.1 配置安全

- 敏感信息 (API密钥、Token) 应从环境变量或密钥管理系统读取
- 不要在规则文件中硬编码密钥

### 6.2 输入验证

- 严格验证规则配置
- 防止正则表达式DOS攻击
- 限制规则数量和复杂度

### 6.3 访问控制

- API接口需要认证
- 规则文件权限控制
- 审计规则变更日志

## 7. 监控和可观测性

### 7.1 指标 (Metrics)

建议收集的指标:
- `rules_loaded_total`: 加载的规则总数
- `rules_matched_total`: 规则匹配次数
- `actions_executed_total`: 动作执行次数
- `actions_failed_total`: 动作失败次数
- `rate_limit_exceeded_total`: 速率限制触发次数
- `escalations_triggered_total`: 告警升级触发次数

### 7.2 日志

关键操作日志:
- 规则加载和重载
- 规则匹配结果
- 动作执行状态
- 重试和失败详情

### 7.3 追踪

- 每个告警分配trace_id
- 追踪从匹配到执行的完整链路

## 8. 部署建议

### 8.1 配置文件位置

开发环境:
```
/home/await/project/edge-link/backend/alert-rules.yaml
```

生产环境:
```
/etc/edgelink/alert-rules.yaml
```

### 8.2 环境变量

```bash
# 规则文件路径
ALERT_RULES_FILE=/etc/edgelink/alert-rules.yaml

# 启用规则引擎
ENABLE_RULE_ENGINE=true

# 启用热重载
ENABLE_HOT_RELOAD=true

# 重载间隔
RULE_RELOAD_INTERVAL=5m
```

### 8.3 Docker部署

```yaml
version: '3.8'
services:
  alert-service:
    image: edgelink/alert-service:latest
    volumes:
      - ./alert-rules.yaml:/etc/edgelink/alert-rules.yaml:ro
    environment:
      - ALERT_RULES_FILE=/etc/edgelink/alert-rules.yaml
      - ENABLE_RULE_ENGINE=true
```

## 9. 测试策略

### 9.1 单元测试

- Parser: 测试YAML解析和验证
- Matcher: 测试各种条件组合
- Executor: Mock HTTP客户端测试

### 9.2 集成测试

- 端到端规则执行流程
- 真实通知渠道测试 (使用测试webhook)
- 速率限制和升级逻辑

### 9.3 性能测试

- 规则匹配性能 (1000+ 规则)
- 并发告警处理
- 内存使用监控

## 10. 故障处理

### 10.1 常见问题

| 问题 | 原因 | 解决方案 |
|-----|------|---------|
| 规则加载失败 | YAML格式错误 | 检查语法，查看日志 |
| 通知未发送 | 规则不匹配 | 使用测试API验证规则 |
| 速率限制过严 | 配置过小 | 调整rate_limit参数 |
| 告警升级未触发 | 时间未达到 | 检查wait_duration设置 |

### 10.2 降级策略

如果规则引擎故障，系统会自动回退到传统调度方式（基于severity的简单路由）。

## 11. 未来扩展

### 11.1 计划功能

1. **规则组**: 规则的逻辑分组和继承
2. **条件模板**: 可复用的条件片段
3. **动作模板**: 预定义的通知配置
4. **规则版本控制**: Git集成，回滚支持
5. **规则测试框架**: 单元测试规则配置
6. **可视化编辑器**: Web UI规则配置
7. **机器学习集成**: 自动调整规则参数

### 11.2 技术演进

- 从文件配置迁移到数据库
- 支持动态规则（用户自定义）
- 分布式规则引擎集群
- 实时规则性能分析

## 12. 参考资料

- Prometheus Alertmanager: 告警路由参考
- Grafana Alerting: 通知渠道集成
- PagerDuty Event API: 升级逻辑
- RFC 5545 (iCalendar): 时间范围表示
