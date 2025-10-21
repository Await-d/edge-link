# 通知规则引擎使用指南

## 快速开始

### 1. 基本概念

通知规则引擎由以下核心概念组成:

- **规则 (Rule)**: 定义何时、如何发送告警通知的配置
- **条件 (Conditions)**: 规则触发的匹配条件
- **动作 (Actions)**: 规则触发后执行的通知操作
- **优先级 (Priority)**: 规则执行的顺序 (数字越小优先级越高)

### 2. 创建第一个规则

创建 `alert-rules.yaml` 文件:

```yaml
version: "1.0"

rules:
  - id: "my-first-rule"
    name: "Critical Alerts"
    priority: 10
    enabled: true
    conditions:
      severity:
        - critical
    actions:
      - type: email
        enabled: true
        config:
          recipients:
            - admin@example.com
```

### 3. 启动服务

```bash
# 设置规则文件路径
export ALERT_RULES_FILE=./alert-rules.yaml
export ENABLE_RULE_ENGINE=true

# 启动告警服务
./alert-service
```

## 规则配置详解

### 规则结构

```yaml
rules:
  - id: "unique-rule-id"           # 必需: 唯一标识符
    name: "Rule Display Name"      # 必需: 显示名称
    description: "Rule description" # 可选: 规则描述
    enabled: true                   # 可选: 是否启用 (默认true)
    priority: 10                    # 可选: 优先级 (默认100)
    conditions: {...}               # 必需: 匹配条件
    actions: [...]                  # 必需: 通知动作
    rate_limit: {...}               # 可选: 速率限制
    escalation: {...}               # 可选: 告警升级
    silence: {...}                  # 可选: 静默规则
```

### 条件配置 (Conditions)

#### 基本条件

**按严重程度匹配**:
```yaml
conditions:
  severity:
    - critical
    - high
```

**按告警类型匹配**:
```yaml
conditions:
  alert_types:
    - device_offline
    - high_latency
```

**按设备ID匹配**:
```yaml
conditions:
  device_ids:
    - "uuid-device-1"
    - "uuid-device-2"
```

**按设备标签匹配**:
```yaml
conditions:
  device_tags:
    - production
    - critical-infrastructure
```

#### 时间范围条件

**工作时间**:
```yaml
conditions:
  time_range:
    start: "09:00"
    end: "18:00"
    timezone: "Asia/Shanghai"
    weekdays:
      - Monday
      - Tuesday
      - Wednesday
      - Thursday
      - Friday
```

**跨天时间范围**:
```yaml
conditions:
  time_range:
    start: "22:00"
    end: "06:00"    # 22:00 到次日 06:00
    timezone: "Asia/Shanghai"
```

#### 高级条件组合

**AND逻辑 (所有条件必须满足)**:
```yaml
conditions:
  all_of:
    - severity: [critical]
    - device_tags: [production]
```

**OR逻辑 (任一条件满足)**:
```yaml
conditions:
  any_of:
    - severity: [critical]
    - alert_types: [device_offline]
```

**NOT逻辑 (条件不满足)**:
```yaml
conditions:
  none_of:
    - message_match: ".*test.*"
    - device_tags: [development]
```

**复杂组合**:
```yaml
conditions:
  all_of:
    - severity: [critical, high]
    - any_of:
        - time_range:
            start: "09:00"
            end: "18:00"
        - device_tags: [always-alert]
  none_of:
    - message_match: ".*scheduled maintenance.*"
```

#### 消息内容匹配

使用正则表达式匹配告警消息:
```yaml
conditions:
  message_match: ".*authentication failed.*"
```

#### 元数据匹配

匹配告警元数据中的键值对:
```yaml
conditions:
  metadata:
    region: "us-east-1"
    environment: "production"
```

### 动作配置 (Actions)

#### Email通知

```yaml
actions:
  - type: email
    enabled: true
    config:
      recipients:
        - admin@example.com
        - ops-team@example.com
```

#### Webhook通知

```yaml
actions:
  - type: webhook
    enabled: true
    config:
      url: "https://your-webhook-endpoint.com/alerts"
```

#### Slack通知

```yaml
actions:
  - type: slack
    enabled: true
    config:
      webhook_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
      channel: "#alerts"
      username: "EdgeLink Alerts"
```

#### PagerDuty通知

```yaml
actions:
  - type: pagerduty
    enabled: true
    config:
      service_key: "YOUR_PAGERDUTY_SERVICE_KEY"
```

#### 钉钉通知

```yaml
actions:
  - type: dingtalk
    enabled: true
    config:
      webhook_url: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"
      at_mobiles:
        - "13800138000"
```

#### 企业微信通知

```yaml
actions:
  - type: wechat
    enabled: true
    config:
      webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"
```

#### Telegram通知

```yaml
actions:
  - type: telegram
    enabled: true
    config:
      bot_token: "YOUR_BOT_TOKEN"
      chat_id: "YOUR_CHAT_ID"
```

#### 自定义HTTP请求

```yaml
actions:
  - type: custom
    enabled: true
    config:
      url: "https://your-custom-endpoint.com/alerts"
      method: "POST"
      body:
        service: "edgelink"
        environment: "production"
```

#### 重试策略

为动作配置重试策略:
```yaml
actions:
  - type: webhook
    config:
      url: "https://unreliable-endpoint.com"
    retry_policy:
      max_retries: 5           # 最大重试次数
      retry_delay: 10s         # 初始重试延迟
      backoff_rate: 2.0        # 指数退避倍率
```

### 速率限制 (Rate Limit)

限制通知频率，防止告警风暴:

**全局速率限制**:
```yaml
rate_limit:
  max_notifications: 10      # 最多10条通知
  window: 1h                 # 1小时内
  scope: "global"            # 全局范围
```

**按规则限制**:
```yaml
rate_limit:
  max_notifications: 5
  window: 30m
  scope: "per_rule"          # 每个规则独立计数
```

**按设备限制**:
```yaml
rate_limit:
  max_notifications: 3
  window: 10m
  scope: "per_device"        # 每个设备独立计数
```

### 告警升级 (Escalation)

未确认告警自动升级:

```yaml
escalation:
  enabled: true
  wait_duration: 15m         # 15分钟未确认后升级
  escalate_to:               # 升级后的通知动作
    - type: pagerduty
      config:
        service_key: "ESCALATION_SERVICE_KEY"
    - type: email
      config:
        recipients:
          - supervisor@example.com
  repeat_interval: 30m       # 每30分钟重复通知
  max_repeat: 3              # 最多重复3次
```

### 静默规则 (Silence)

在特定时间段静默通知:

**单个时间段**:
```yaml
silence:
  enabled: true
  comment: "Daily maintenance window"
  time_ranges:
    - start: "02:00"
      end: "04:00"
      timezone: "Asia/Shanghai"
```

**多个时间段**:
```yaml
silence:
  enabled: true
  time_ranges:
    - start: "02:00"
      end: "04:00"
      timezone: "Asia/Shanghai"
      weekdays: [Sunday]
    - start: "00:00"
      end: "23:59"
      timezone: "Asia/Shanghai"
      weekdays: [Saturday]
```

## 常见使用场景

### 场景1: 分级通知

不同严重程度使用不同通知渠道:

```yaml
rules:
  # 关键告警 - 立即通知所有人
  - id: "critical-alerts"
    name: "Critical Alerts"
    priority: 10
    conditions:
      severity: [critical]
    actions:
      - type: email
        config:
          recipients: [admin@example.com, ops@example.com]
      - type: pagerduty
        config:
          service_key: "CRITICAL_SERVICE_KEY"
      - type: slack
        config:
          webhook_url: "..."
          channel: "#critical-alerts"

  # 高优先级 - Slack通知
  - id: "high-alerts"
    name: "High Priority Alerts"
    priority: 20
    conditions:
      severity: [high]
    actions:
      - type: slack
        config:
          webhook_url: "..."
          channel: "#alerts"

  # 中低优先级 - 仅记录
  - id: "low-alerts"
    name: "Low Priority Alerts"
    priority: 30
    conditions:
      severity: [medium, low]
    actions:
      - type: webhook
        config:
          url: "https://log-aggregator.com/alerts"
```

### 场景2: 工作时间自适应

工作时间和非工作时间使用不同通知方式:

```yaml
rules:
  # 工作时间 - Slack通知
  - id: "office-hours"
    name: "Office Hours Notifications"
    priority: 10
    conditions:
      severity: [high, medium]
      time_range:
        start: "09:00"
        end: "18:00"
        timezone: "Asia/Shanghai"
        weekdays: [Monday, Tuesday, Wednesday, Thursday, Friday]
    actions:
      - type: slack
        config:
          webhook_url: "..."

  # 非工作时间 - 仅关键告警通过PagerDuty
  - id: "after-hours-critical"
    name: "After Hours Critical"
    priority: 10
    conditions:
      severity: [critical]
      any_of:
        - time_range:
            start: "18:00"
            end: "09:00"
        - time_range:
            start: "00:00"
            end: "23:59"
            weekdays: [Saturday, Sunday]
    actions:
      - type: pagerduty
        config:
          service_key: "ONCALL_SERVICE_KEY"
```

### 场景3: 团队路由

根据设备标签路由到不同团队:

```yaml
rules:
  # 生产环境 - 运维团队
  - id: "production-team"
    name: "Production Team Alerts"
    priority: 10
    conditions:
      device_tags: [production]
    actions:
      - type: slack
        config:
          webhook_url: "..."
          channel: "#ops-production"
      - type: email
        config:
          recipients: [ops-team@example.com]

  # 开发环境 - 开发团队
  - id: "development-team"
    name: "Development Team Alerts"
    priority: 20
    conditions:
      device_tags: [development]
    actions:
      - type: slack
        config:
          webhook_url: "..."
          channel: "#dev-alerts"

  # 测试环境 - 测试团队
  - id: "testing-team"
    name: "Testing Team Alerts"
    priority: 30
    conditions:
      device_tags: [testing, staging]
    actions:
      - type: email
        config:
          recipients: [qa-team@example.com]
```

### 场景4: 告警聚合和去重

使用速率限制防止告警风暴:

```yaml
rules:
  - id: "high-latency-aggregated"
    name: "High Latency (Aggregated)"
    priority: 20
    conditions:
      alert_types: [high_latency]
    actions:
      - type: slack
        config:
          webhook_url: "..."
    rate_limit:
      max_notifications: 3    # 每小时最多3条
      window: 1h
      scope: "per_device"     # 每个设备独立计数
```

### 场景5: 维护窗口静默

维护期间静默低优先级告警:

```yaml
rules:
  - id: "maintenance-silence"
    name: "Maintenance Window"
    priority: 50
    conditions:
      severity: [low, medium]
    actions:
      - type: webhook
        config:
          url: "https://maintenance-log.com"
    silence:
      enabled: true
      comment: "Weekly maintenance window"
      time_ranges:
        - start: "02:00"
          end: "04:00"
          timezone: "Asia/Shanghai"
          weekdays: [Sunday]
```

## API接口

### 查询所有规则

```bash
curl -X GET http://localhost:8080/api/v1/rules
```

响应:
```json
{
  "rules": [...],
  "total": 10
}
```

### 查询单个规则

```bash
curl -X GET http://localhost:8080/api/v1/rules/critical-alerts
```

### 重新加载规则

```bash
curl -X POST http://localhost:8080/api/v1/rules/reload \
  -H "Content-Type: application/json"
```

响应:
```json
{
  "success": true,
  "message": "Rules reloaded successfully",
  "count": 10
}
```

### 测试规则匹配

```bash
curl -X POST http://localhost:8080/api/v1/rules/test \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": "critical-alerts",
    "alert": {
      "severity": "critical",
      "type": "device_offline"
    }
  }'
```

## 最佳实践

### 1. 规则组织

- 按优先级排序规则 (10, 20, 30, ...)
- 使用有意义的规则ID和名称
- 添加描述说明规则用途
- 定期审查和清理不用的规则

### 2. 性能优化

- 避免过于复杂的正则表达式
- 合理设置速率限制
- 使用设备标签而非大量设备ID
- 限制规则总数 (建议 < 100)

### 3. 通知设计

- 关键告警使用多渠道通知
- 配置合理的重试策略
- 避免告警疲劳 (使用速率限制和聚合)
- 测试所有通知渠道的可用性

### 4. 维护建议

- 版本控制规则文件 (Git)
- 在测试环境验证规则变更
- 监控规则匹配和执行指标
- 定期审查告警升级和静默规则

### 5. 安全建议

- 不要在规则文件中硬编码密钥
- 使用环境变量存储敏感信息
- 限制规则文件访问权限
- 审计规则变更历史

## 故障排查

### 规则未匹配

1. 检查规则是否启用 (`enabled: true`)
2. 验证条件是否正确
3. 使用测试API验证规则逻辑
4. 查看日志中的匹配结果

### 通知未发送

1. 检查动作是否启用
2. 验证通知渠道配置 (URL、密钥等)
3. 检查速率限制是否触发
4. 查看错误日志

### 规则加载失败

1. 验证YAML语法
2. 检查必需字段
3. 查看详细错误信息
4. 使用YAML验证器

### 告警风暴

1. 调整速率限制参数
2. 启用告警聚合
3. 添加静默规则
4. 修复告警根本原因

## 示例配置

完整示例配置请参考:
```
/home/await/project/edge-link/backend/alert-rules.yaml
```

## 进阶话题

### 1. 动态规则加载

从数据库加载规则 (需要自定义实现):
```go
// 从数据库加载规则
rulesData := loadRulesFromDatabase()
engine.LoadRulesFromBytes(rulesData)
```

### 2. 自定义通知渠道

扩展Executor支持新的通知类型。

### 3. 规则测试框架

编写单元测试验证规则逻辑:
```go
// 测试规则匹配
matched, err := engine.TestRule("critical-alerts", alert, device)
assert.True(t, matched)
```

### 4. 规则性能分析

监控每个规则的执行时间和成功率。

## 参考资料

- 设计文档: `docs/notification-rule-engine-design.md`
- 示例配置: `alert-rules.yaml`
- API文档: Swagger UI
