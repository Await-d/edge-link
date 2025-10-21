# Alert Service Integration Architecture Diagrams

## 1. 总体架构图

```mermaid
graph TB
    subgraph "Edge-Link Core"
        Device[Device]
        Metrics[Metrics Collector]
    end

    subgraph "Alert Service"
        Generator[Alert Generator]
        Checker[Threshold Checker]
        Scheduler[Notification Scheduler]

        subgraph "Integration Layer"
            Manager[Integration Manager]
            Factory[Integration Factory]

            subgraph "Platform Adapters"
                PD[PagerDuty<br/>Priority: 1]
                OG[Opsgenie<br/>Priority: 2]
                SL[Slack<br/>Priority: 3]
                DC[Discord<br/>Priority: 4]
                TM[Teams<br/>Priority: 5]
            end
        end
    end

    subgraph "External Services"
        PDAPI[PagerDuty<br/>Events API v2]
        OGAPI[Opsgenie<br/>Alerts API v2]
        SLAPI[Slack<br/>Webhook]
        DCAPI[Discord<br/>Webhook]
        TMAPI[Teams<br/>Webhook]
    end

    Device -->|Telemetry| Metrics
    Metrics -->|Check Thresholds| Checker
    Checker -->|Alert Triggered| Generator
    Generator -->|New Alert| Scheduler
    Scheduler -->|Notify| Manager

    Manager -->|Send| PD
    Manager -->|Send| OG
    Manager -->|Send| SL
    Manager -->|Send| DC
    Manager -->|Send| TM

    PD -->|HTTPS POST| PDAPI
    OG -->|HTTPS POST| OGAPI
    SL -->|HTTPS POST| SLAPI
    DC -->|HTTPS POST| DCAPI
    TM -->|HTTPS POST| TMAPI

    Factory -.->|Create & Register| Manager
    Factory -.->|Initialize| PD
    Factory -.->|Initialize| OG
    Factory -.->|Initialize| SL
    Factory -.->|Initialize| DC
    Factory -.->|Initialize| TM

    style Manager fill:#4CAF50,stroke:#2E7D32,stroke-width:3px
    style Factory fill:#2196F3,stroke:#1565C0,stroke-width:2px
    style PD fill:#FF6B6B,stroke:#D32F2F
    style OG fill:#4ECDC4,stroke:#00897B
    style SL fill:#FFE66D,stroke:#F9A825
    style DC fill:#A8DADC,stroke:#0277BD
    style TM fill:#457B9D,stroke:#01579B
```

## 2. 告警发送流程

```mermaid
sequenceDiagram
    participant AS as Alert Service
    participant IM as Integration Manager
    participant PD as PagerDuty Adapter
    participant SL as Slack Adapter
    participant DC as Discord Adapter
    participant PDAPI as PagerDuty API
    participant SLAPI as Slack API
    participant DCAPI as Discord API

    AS->>IM: SendAlert(ctx, alert)

    Note over IM: 获取启用的集成<br/>并按优先级排序

    par 并发发送到所有平台
        IM->>PD: SendAlert(ctx, alert)
        Note over PD: Priority: 1
        PD->>PD: buildEvent(alert)
        PD->>PD: marshal JSON

        loop 重试（最多3次）
            PD->>PDAPI: POST /v2/enqueue
            alt 成功 (200-299)
                PDAPI-->>PD: Success
                PD-->>IM: nil
            else 服务器错误 (5xx)
                PDAPI-->>PD: 500 Error
                Note over PD: 等待2s后重试
            else 客户端错误 (4xx)
                PDAPI-->>PD: 400 Error
                PD-->>IM: Error (不可重试)
            end
        end
    and
        IM->>SL: SendAlert(ctx, alert)
        Note over SL: Priority: 3
        SL->>SL: buildMessage(alert)
        SL->>SL: marshal JSON

        loop 重试（最多2次）
            SL->>SLAPI: POST webhook_url
            alt 成功 (200)
                SLAPI-->>SL: ok
                SL-->>IM: nil
            else 失败
                SLAPI-->>SL: Error
                Note over SL: 等待1s后重试
            end
        end
    and
        IM->>DC: SendAlert(ctx, alert)
        Note over DC: Priority: 4
        DC->>DC: buildMessage(alert)
        DC->>DC: marshal JSON

        DC->>DCAPI: POST webhook_url
        alt 成功 (204)
            DCAPI-->>DC: No Content
            DC-->>IM: nil
        else 失败
            DCAPI-->>DC: Error
            DC-->>IM: Error
        end
    end

    Note over IM: 收集所有结果

    alt 至少一个平台成功
        IM-->>AS: nil (成功)
    else 所有平台失败
        IM-->>AS: Error (失败)
    end
```

## 3. 重试机制流程

```mermaid
graph TD
    Start[开始发送] --> Attempt0[尝试 0: 立即发送]

    Attempt0 --> Check0{成功?}
    Check0 -->|是| Success[返回成功]
    Check0 -->|否| IsRetryable0{可重试?}

    IsRetryable0 -->|否| Failed[返回失败]
    IsRetryable0 -->|是| Wait1[等待 2s]

    Wait1 --> Attempt1[尝试 1: 重试]
    Attempt1 --> Check1{成功?}
    Check1 -->|是| Success
    Check1 -->|否| IsRetryable1{可重试?}

    IsRetryable1 -->|否| Failed
    IsRetryable1 -->|是| Wait2[等待 4s<br/>指数退避]

    Wait2 --> Attempt2[尝试 2: 重试]
    Attempt2 --> Check2{成功?}
    Check2 -->|是| Success
    Check2 -->|否| IsRetryable2{可重试?}

    IsRetryable2 -->|否| Failed
    IsRetryable2 -->|是| Wait3[等待 8s<br/>指数退避]

    Wait3 --> Attempt3[尝试 3: 最后一次]
    Attempt3 --> Check3{成功?}
    Check3 -->|是| Success
    Check3 -->|否| Failed

    style Success fill:#4CAF50,stroke:#2E7D32
    style Failed fill:#F44336,stroke:#C62828
    style Wait1 fill:#FFC107,stroke:#F57C00
    style Wait2 fill:#FFC107,stroke:#F57C00
    style Wait3 fill:#FFC107,stroke:#F57C00
```

## 4. 错误分类决策树

```mermaid
graph TD
    Error[收到错误] --> CheckType{错误类型?}

    CheckType -->|5xx| Retryable1[可重试]
    CheckType -->|429 Rate Limit| Retryable2[可重试]
    CheckType -->|网络超时| Retryable3[可重试]
    CheckType -->|连接失败| Retryable4[可重试]

    CheckType -->|4xx 客户端错误| NotRetryable1[不可重试]
    CheckType -->|配置错误| NotRetryable2[不可重试]
    CheckType -->|认证失败| NotRetryable3[不可重试]
    CheckType -->|JSON序列化失败| NotRetryable4[不可重试]

    Retryable1 --> Retry[执行重试逻辑]
    Retryable2 --> Retry
    Retryable3 --> Retry
    Retryable4 --> Retry

    NotRetryable1 --> Fail[立即返回失败]
    NotRetryable2 --> Fail
    NotRetryable3 --> Fail
    NotRetryable4 --> Fail

    Retry --> BackoffWait[指数退避等待]
    BackoffWait --> NextAttempt[下一次尝试]

    style Retryable1 fill:#4CAF50,stroke:#2E7D32
    style Retryable2 fill:#4CAF50,stroke:#2E7D32
    style Retryable3 fill:#4CAF50,stroke:#2E7D32
    style Retryable4 fill:#4CAF50,stroke:#2E7D32
    style NotRetryable1 fill:#F44336,stroke:#C62828
    style NotRetryable2 fill:#F44336,stroke:#C62828
    style NotRetryable3 fill:#F44336,stroke:#C62828
    style NotRetryable4 fill:#F44336,stroke:#C62828
```

## 5. 集成生命周期

```mermaid
stateDiagram-v2
    [*] --> Uninitialized: 启动服务

    Uninitialized --> Validating: 加载配置
    Validating --> Initialized: 验证通过
    Validating --> Failed: 验证失败

    Initialized --> Registered: 注册到Manager
    Registered --> HealthChecking: 定期健康检查

    HealthChecking --> Healthy: 检查通过
    HealthChecking --> Unhealthy: 检查失败

    Healthy --> Sending: 发送告警
    Unhealthy --> Sending: 尝试发送

    Sending --> Success: 发送成功
    Sending --> Retrying: 发送失败（可重试）
    Sending --> Error: 发送失败（不可重试）

    Success --> Healthy: 更新指标
    Retrying --> Success: 重试成功
    Retrying --> Error: 重试失败

    Error --> Unhealthy: 标记不健康

    Healthy --> HealthChecking: 定期检查
    Unhealthy --> HealthChecking: 定期检查

    Registered --> Unregistered: 注销集成
    Unregistered --> [*]
    Failed --> [*]
```

## 6. 数据流图

```mermaid
graph LR
    subgraph "Input"
        Alert[Alert Entity]
    end

    subgraph "Processing"
        Manager[Integration Manager]

        subgraph "Transformation"
            PDTrans[PagerDuty<br/>Transform]
            OGTrans[Opsgenie<br/>Transform]
            SLTrans[Slack<br/>Transform]
        end

        subgraph "Serialization"
            PDJSON[PagerDuty<br/>JSON]
            OGJSON[Opsgenie<br/>JSON]
            SLJSON[Slack<br/>JSON]
        end

        subgraph "HTTP Client"
            PDReq[PagerDuty<br/>Request]
            OGReq[Opsgenie<br/>Request]
            SLReq[Slack<br/>Request]
        end
    end

    subgraph "Output"
        PDAPI[PagerDuty<br/>API]
        OGAPI[Opsgenie<br/>API]
        SLAPI[Slack<br/>API]
    end

    Alert --> Manager

    Manager -->|Alert Data| PDTrans
    Manager -->|Alert Data| OGTrans
    Manager -->|Alert Data| SLTrans

    PDTrans -->|Event Struct| PDJSON
    OGTrans -->|Alert Struct| OGJSON
    SLTrans -->|Message Struct| SLJSON

    PDJSON -->|[]byte| PDReq
    OGJSON -->|[]byte| OGReq
    SLJSON -->|[]byte| SLReq

    PDReq -->|POST| PDAPI
    OGReq -->|POST| OGAPI
    SLReq -->|POST| SLAPI

    style Manager fill:#4CAF50
    style PDTrans fill:#FFE66D
    style OGTrans fill:#FFE66D
    style SLTrans fill:#FFE66D
```

## 7. 配置加载流程

```mermaid
graph TD
    Start[启动服务] --> LoadConfig[加载YAML配置]
    LoadConfig --> EnvOverride[环境变量覆盖]
    EnvOverride --> Validate[验证配置]

    Validate --> ValidateEnabled{至少一个<br/>集成启用?}
    ValidateEnabled -->|否| ConfigError[配置错误]
    ValidateEnabled -->|是| ValidateKeys[验证API密钥]

    ValidateKeys --> KeyCheck{密钥完整?}
    KeyCheck -->|否| ConfigError
    KeyCheck -->|是| CreateFactory[创建Factory]

    CreateFactory --> CreateManager[创建Manager]
    CreateManager --> RegisterIntegrations[注册集成]

    RegisterIntegrations --> CheckPD{PagerDuty<br/>启用?}
    CheckPD -->|是| RegisterPD[注册PD<br/>Priority: 1]
    CheckPD -->|否| CheckOG
    RegisterPD --> CheckOG

    CheckOG{Opsgenie<br/>启用?}
    CheckOG -->|是| RegisterOG[注册OG<br/>Priority: 2]
    CheckOG -->|否| CheckSlack
    RegisterOG --> CheckSlack

    CheckSlack{Slack<br/>启用?}
    CheckSlack -->|是| RegisterSlack[注册Slack<br/>Priority: 3]
    CheckSlack -->|否| Complete
    RegisterSlack --> Complete

    Complete[初始化完成] --> HealthCheck[执行健康检查]
    HealthCheck --> Ready[服务就绪]

    ConfigError --> Failed[启动失败]

    style Ready fill:#4CAF50,stroke:#2E7D32,stroke-width:3px
    style Failed fill:#F44336,stroke:#C62828,stroke-width:3px
    style CreateManager fill:#2196F3,stroke:#1565C0
```

## 8. 指标收集架构

```mermaid
graph TB
    subgraph "Integration Manager"
        Manager[Manager]
        MetricsMap[Metrics Map<br/>map[string]*Metrics]
    end

    subgraph "每个集成的指标"
        M1[PagerDuty Metrics]
        M2[Opsgenie Metrics]
        M3[Slack Metrics]
    end

    subgraph "指标内容"
        Fields[TotalSent: 100<br/>SuccessCount: 95<br/>FailureCount: 5<br/>AvgResponseTime: 200ms<br/>LastSentTime: 2025-10-20T10:30:00Z]
    end

    subgraph "外部监控"
        Prometheus[Prometheus]
        Grafana[Grafana]
        AlertManager[AlertManager]
    end

    Manager --> MetricsMap
    MetricsMap --> M1
    MetricsMap --> M2
    MetricsMap --> M3

    M1 --> Fields
    M2 --> Fields
    M3 --> Fields

    Manager -->|/metrics endpoint| Prometheus
    Prometheus --> Grafana
    Prometheus --> AlertManager

    style Manager fill:#4CAF50
    style MetricsMap fill:#2196F3
    style Prometheus fill:#E37222
    style Grafana fill:#F46800
```

## 9. 平台特性对比

```mermaid
graph TD
    subgraph "Platform Features"
        PD[PagerDuty]
        OG[Opsgenie]
        SL[Slack]
        DC[Discord]
        TM[Teams]
    end

    subgraph "Features"
        F1[去重键]
        F2[自动解决]
        F3[优先级]
        F4[团队路由]
        F5[富文本格式]
        F6[交互按钮]
    end

    PD -->|✅| F1
    PD -->|✅| F2
    PD -->|✅| F3
    PD -->|❌| F4
    PD -->|❌| F5
    PD -->|❌| F6

    OG -->|✅| F1
    OG -->|✅| F2
    OG -->|✅| F3
    OG -->|✅| F4
    OG -->|✅| F5
    OG -->|❌| F6

    SL -->|❌| F1
    SL -->|❌| F2
    SL -->|❌| F3
    SL -->|❌| F4
    SL -->|✅| F5
    SL -->|✅| F6

    DC -->|❌| F1
    DC -->|❌| F2
    DC -->|❌| F3
    DC -->|❌| F4
    DC -->|✅| F5
    DC -->|❌| F6

    TM -->|❌| F1
    TM -->|❌| F2
    TM -->|❌| F3
    TM -->|❌| F4
    TM -->|✅| F5
    TM -->|✅| F6

    style PD fill:#FF6B6B
    style OG fill:#4ECDC4
    style SL fill:#FFE66D
    style DC fill:#A8DADC
    style TM fill:#457B9D
```

## 10. 健康检查流程

```mermaid
sequenceDiagram
    participant HC as Health Check Service
    participant M as Manager
    participant PD as PagerDuty
    participant SL as Slack
    participant DC as Discord
    participant API as External APIs

    HC->>M: HealthCheck(ctx)

    par 并发检查所有集成
        M->>PD: HealthCheck(ctx)
        PD->>API: 发送测试事件
        alt 成功
            API-->>PD: 200 OK
            PD-->>M: nil
        else 失败
            API-->>PD: Error
            PD-->>M: Error
        end
    and
        M->>SL: HealthCheck(ctx)
        SL->>API: 发送测试消息
        alt 成功
            API-->>SL: 200 OK
            SL-->>M: nil
        else 失败
            API-->>SL: Error
            SL-->>M: Error
        end
    and
        M->>DC: HealthCheck(ctx)
        DC->>API: 发送测试消息
        alt 成功
            API-->>DC: 204 No Content
            DC-->>M: nil
        else 失败
            API-->>DC: Error
            DC-->>M: Error
        end
    end

    M-->>HC: map[string]error<br/>{<br/>  "pagerduty": nil,<br/>  "slack": nil,<br/>  "discord": error<br/>}

    Note over HC: 记录健康状态<br/>更新监控指标
```

这些图表展示了集成架构的各个方面，从整体架构到具体的流程细节，帮助理解系统的设计和工作原理。
