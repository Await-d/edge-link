# Alert Service - Third-Party Integrations

## 概述

这是Alert Service的第三方告警平台集成模块，提供了统一的接口来集成多个告警平台，包括PagerDuty、Opsgenie、Slack、Discord和Microsoft Teams。

## 特性

- ✅ **统一接口**：所有平台通过相同的Integration接口访问
- ✅ **多平台并发**：同时发送告警到所有启用平台
- ✅ **失败重试**：指数退避重试机制，最大化送达率
- ✅ **优先级路由**：基于优先级排序，支持备用通道
- ✅ **健康检查**：运行时平台可用性检测
- ✅ **指标收集**：成功率、响应时间等关键指标
- ✅ **容错设计**：部分失败不影响整体，单平台故障隔离
- ✅ **配置驱动**：YAML配置，支持环境变量覆盖

## 支持的平台

| 平台 | 状态 | 特性 |
|------|------|------|
| **PagerDuty** | ✅ | Events API v2, 去重键, 自动解决 |
| **Opsgenie** | ✅ | Alerts API v2, 团队路由, P1-P5优先级 |
| **Slack** | ✅ | Webhook, Block Kit, 交互按钮 |
| **Discord** | ✅ | Webhook, Embed消息, 丰富格式 |
| **Teams** | ✅ | Webhook, Adaptive Cards, 操作按钮 |

## 快速开始

### 1. 配置文件

复制示例配置并根据你的环境修改：

```bash
cp config/integrations.example.yaml config/integrations.yaml
```

配置示例：

```yaml
integrations:
  pagerduty:
    enabled: true
    priority: 1
    integration_key: "${PAGERDUTY_INTEGRATION_KEY}"

  slack:
    enabled: true
    priority: 2
    webhook_url: "${SLACK_WEBHOOK_URL}"
    channel: "#alerts"
```

### 2. 环境变量

设置必要的环境变量：

```bash
export PAGERDUTY_INTEGRATION_KEY="your-integration-key"
export OPSGENIE_API_KEY="your-api-key"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/YOUR/WEBHOOK"
export TEAMS_WEBHOOK_URL="https://outlook.office.com/webhook/YOUR/WEBHOOK"
```

### 3. 代码集成

```go
package main

import (
    "context"
    "github.com/edgelink/backend/cmd/alert-service/internal/config"
    "github.com/edgelink/backend/cmd/alert-service/internal/integrations"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // 加载配置
    cfg := loadConfig("config/integrations.yaml")

    // 创建集成管理器
    factory := integrations.NewFactory(logger)
    manager, _ := factory.CreateManager(cfg.Integrations)

    // 发送告警
    ctx := context.Background()
    alert := createAlert()

    if err := manager.SendAlert(ctx, alert); err != nil {
        logger.Error("Failed to send alert", zap.Error(err))
    }
}
```

## 平台配置详解

### PagerDuty

1. 登录PagerDuty -> Service -> Integrations
2. 创建"Events API v2"集成
3. 复制Integration Key

```yaml
pagerduty:
  enabled: true
  priority: 1
  integration_key: "R034XXXXXXXXXXX"
  default_service: "edge-link"
  severity_map:
    critical: "critical"
    high: "error"
    medium: "warning"
    low: "info"
```

### Opsgenie

1. 登录Opsgenie -> Settings -> API Key Management
2. 创建"Create and Update"权限的API Key

```yaml
opsgenie:
  enabled: true
  priority: 2
  api_key: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  default_teams:
    - "network-ops"
  priority_map:
    critical: "P1"
    high: "P2"
    medium: "P3"
    low: "P4"
```

### Slack

1. 访问 https://api.slack.com/messaging/webhooks
2. 创建Incoming Webhook
3. 选择目标频道并复制Webhook URL

```yaml
slack:
  enabled: true
  priority: 3
  webhook_url: "https://hooks.slack.com/services/T00/B00/XXXX"
  channel: "#alerts"  # 可选，覆盖默认频道
  username: "EdgeLink Alert Bot"
  icon_emoji: ":warning:"
```

### Discord

1. 服务器设置 -> 整合 -> Webhook
2. 创建Webhook并复制URL

```yaml
discord:
  enabled: true
  priority: 4
  webhook_url: "https://discord.com/api/webhooks/123/XXXX"
  username: "EdgeLink Alert Bot"
```

### Microsoft Teams

1. Teams频道 -> 连接器 -> Incoming Webhook
2. 配置Webhook并复制URL

```yaml
teams:
  enabled: true
  priority: 5
  webhook_url: "https://outlook.office.com/webhook/xxx"
```

## API接口

### Integration接口

所有平台适配器实现此接口：

```go
type Integration interface {
    // Name 返回集成名称
    Name() string

    // SendAlert 发送告警
    SendAlert(ctx context.Context, alert *domain.Alert) error

    // ResolveAlert 解决告警
    ResolveAlert(ctx context.Context, alertID string) error

    // UpdateAlert 更新告警状态
    UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error

    // ValidateConfig 验证配置
    ValidateConfig() error

    // HealthCheck 健康检查
    HealthCheck(ctx context.Context) error
}
```

### Manager接口

```go
// 发送告警到所有启用平台
func (m *Manager) SendAlert(ctx context.Context, alert *domain.Alert) error

// 解决告警
func (m *Manager) ResolveAlert(ctx context.Context, alertID string) error

// 更新告警状态
func (m *Manager) UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error

// 健康检查所有平台
func (m *Manager) HealthCheck(ctx context.Context) map[string]error

// 获取指标
func (m *Manager) GetMetrics() map[string]*IntegrationMetrics
```

## 重试机制

默认重试配置：

```go
RetryConfig{
    MaxRetries:    3,           // 最多重试3次
    InitialDelay:  2s,          // 初始延迟2秒
    MaxDelay:      30s,         // 最大延迟30秒
    BackoffFactor: 2.0,         // 指数退避因子
}
```

重试时间线：
```
Attempt 0: 立即 (0s)
Attempt 1: 2s后
Attempt 2: 4s后
Attempt 3: 8s后
```

## 错误处理

### 错误分类

```go
type IntegrationError struct {
    Integration string  // 集成名称
    Operation   string  // 操作类型
    AlertID     string  // 告警ID
    Err         error   // 原始错误
    Retryable   bool    // 是否可重试
}
```

### 可重试错误
- 5xx服务器错误
- 429 速率限制
- 网络超时
- 连接失败

### 不可重试错误
- 4xx客户端错误（认证、格式错误等）
- 配置错误
- JSON序列化失败

## 测试

### 单元测试

```bash
cd internal/integrations
go test -v ./...
```

### 集成测试

需要设置真实的API凭证：

```bash
# 测试Slack集成
go run examples/examples.go test slack

# 测试PagerDuty集成
go run examples/examples.go test pagerduty

# 测试所有集成
go run examples/examples.go example
```

### 健康检查测试

```bash
# 检查所有平台健康状态
curl http://localhost:8080/api/v1/integrations/health
```

## 监控指标

每个集成收集以下指标：

```go
type IntegrationMetrics struct {
    TotalSent       int64         // 总发送数
    SuccessCount    int64         // 成功数
    FailureCount    int64         // 失败数
    LastSentTime    time.Time     // 最后发送时间
    AvgResponseTime time.Duration // 平均响应时间
}
```

访问指标：

```bash
# 获取所有集成指标
curl http://localhost:8080/api/v1/integrations/metrics
```

## 扩展新平台

### 步骤1: 创建适配器

```go
// internal/integrations/newplatform/newplatform.go
package newplatform

type Integration struct {
    config *Config
    logger *zap.Logger
}

func (i *Integration) SendAlert(ctx context.Context, alert *domain.Alert) error {
    // 实现发送逻辑
    return nil
}

// 实现其他接口方法...
```

### 步骤2: 定义配置

```go
type Config struct {
    Enabled     bool
    Priority    int
    APIKey      string
    RetryConfig integrations.RetryConfig
}

func (c *Config) Enabled() bool { return c.Enabled }
func (c *Config) Priority() int { return c.Priority }
func (c *Config) RetryConfig() integrations.RetryConfig { return c.RetryConfig }
```

### 步骤3: 在Factory中注册

```go
// internal/integrations/factory.go
case "newplatform":
    cfg := config.NewPlatform.ToNewPlatformConfig()
    integration := newplatform.NewIntegration(cfg, logger)
    return integration, nil
```

### 步骤4: 更新配置文件

```yaml
integrations:
  newplatform:
    enabled: true
    priority: 6
    api_key: "${NEWPLATFORM_API_KEY}"
```

## 架构设计

详细架构文档请参考：[ARCHITECTURE.md](ARCHITECTURE.md)

## 故障排查

### 问题：告警未发送到平台

**检查清单**：
1. 配置文件中`enabled: true`
2. 环境变量已正确设置
3. API凭证有效
4. 网络连接正常
5. 查看日志中的错误信息

### 问题：重试次数过多

**解决方案**：
1. 检查平台API状态
2. 验证速率限制
3. 调整重试配置：
```yaml
retry_config:
  max_retries: 2      # 减少重试次数
  initial_delay: 5s   # 增加初始延迟
```

### 问题：告警格式不正确

**解决方案**：
1. 检查severity映射配置
2. 验证metadata格式
3. 查看平台API文档

## 安全建议

1. **不要在配置文件中硬编码API Key**
   ```yaml
   # ❌ 错误
   api_key: "R034EXAMPLE123456"

   # ✅ 正确
   api_key: "${PAGERDUTY_INTEGRATION_KEY}"
   ```

2. **使用HTTPS通信**
   - 所有适配器默认使用HTTPS
   - 验证SSL证书

3. **日志脱敏**
   - API Key不记录在日志中
   - 仅记录操作结果和错误信息

4. **访问控制**
   - 限制配置文件权限（600）
   - 使用secret管理工具（如Vault）

## 性能考虑

### 并发发送
- 多平台并行通知，不阻塞主流程
- 单平台失败不影响其他平台

### 连接池
- HTTP客户端复用连接
- 默认超时10-15秒

### 资源消耗
- 每个集成一个goroutine
- 内存占用约1-2MB/集成

## 相关文档

- [ARCHITECTURE.md](ARCHITECTURE.md) - 详细架构设计
- [examples/examples.go](examples/examples.go) - 使用示例
- [config/integrations.example.yaml](../config/integrations.example.yaml) - 配置示例

## 贡献指南

欢迎贡献新的平台集成！请遵循以下步骤：

1. Fork仓库
2. 创建特性分支
3. 实现Integration接口
4. 添加测试
5. 更新文档
6. 提交Pull Request

## 许可证

Edge-Link项目的一部分，遵循项目统一许可证。

## 联系方式

- 问题反馈：GitHub Issues
- 技术讨论：项目Wiki
