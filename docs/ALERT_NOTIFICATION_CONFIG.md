# Edge-Link 告警通知配置指南

## 1. 功能测试总结

### ✅ 已完成测试

#### 1.1 设备离线检测
- **测试设备**: 3个
  - Device 1: 离线1分钟 → 触发 HIGH 告警
  - Device 2: 离线10分钟 → 触发 HIGH 告警
  - Device 3: 离线1小时 → 触发 CRITICAL 告警

#### 1.2 告警生成
- **告警数量**: 6条（每次检查循环生成2条）
- **告警类型**: device_offline
- **严重程度**: HIGH (离线5-30分钟), CRITICAL (离线>30分钟)
- **告警内容**:
  - 中文标题和消息
  - 完整的元数据（设备名称、最后上线时间、离线时长、虚拟网络ID）

#### 1.3 Webhook通知
- **测试工具**: Python HTTP服务器 (test-webhook-receiver.py)
- **测试结果**: ✅ 成功接收和解析告警数据
- **数据完整性**: 所有字段正确传输

---

## 2. SMTP邮件通知配置

### 2.1 配置位置

邮件通知配置在 `backend/cmd/alert-service/internal/notifier/email_notifier.go` 中。

### 2.2 环境变量配置

在 `docker-compose.yml` 或系统环境中设置以下变量：

```yaml
services:
  alert-service:
    environment:
      - SMTP_HOST=smtp.gmail.com
      - SMTP_PORT=587
      - SMTP_USER=your-email@gmail.com
      - SMTP_PASSWORD=your-app-password
      - SMTP_FROM=EdgeLink Alert <noreply@edgelink.com>
```

### 2.3 常用SMTP服务器配置

#### Gmail
```
SMTP_HOST: smtp.gmail.com
SMTP_PORT: 587
SMTP_USER: your-email@gmail.com
SMTP_PASSWORD: <应用专用密码>
```
**注意**: 需要启用"两步验证"并生成"应用专用密码"

#### Outlook/Office 365
```
SMTP_HOST: smtp.office365.com
SMTP_PORT: 587
SMTP_USER: your-email@outlook.com
SMTP_PASSWORD: <账户密码>
```

#### SendGrid
```
SMTP_HOST: smtp.sendgrid.net
SMTP_PORT: 587
SMTP_USER: apikey
SMTP_PASSWORD: <SendGrid API Key>
```

#### Mailgun
```
SMTP_HOST: smtp.mailgun.org
SMTP_PORT: 587
SMTP_USER: postmaster@your-domain.mailgun.org
SMTP_PASSWORD: <Mailgun SMTP密码>
```

### 2.4 代码修改步骤

修改 `email_notifier.go` 以从环境变量读取配置：

```go
func NewEmailNotifier(logger *zap.Logger) *EmailNotifier {
    return &EmailNotifier{
        smtpHost:     os.Getenv("SMTP_HOST"),
        smtpPort:     os.Getenv("SMTP_PORT"),
        smtpUser:     os.Getenv("SMTP_USER"),
        smtpPassword: os.Getenv("SMTP_PASSWORD"),
        fromAddress:  os.Getenv("SMTP_FROM"),
        logger:       logger,
    }
}
```

### 2.5 测试邮件发送

```bash
# 重新构建Alert Service
docker compose build alert-service

# 重启服务
docker compose restart alert-service

# 查看日志
docker logs edgelink-alert-service -f

# 等待下一次告警检查（每分钟一次）
# 或手动触发告警测试
```

---

## 3. Webhook通知配置

### 3.1 配置Webhook URL

通过NotificationScheduler配置Webhook端点。

### 3.2 Webhook Payload格式

```json
{
  "alert_id": "uuid",
  "title": "告警标题",
  "message": "告警消息",
  "severity": "critical|high|medium|low",
  "alert_type": "device_offline|high_latency|...",
  "device_id": "设备UUID",
  "metadata": {
    "device_name": "设备名称",
    "last_seen_at": "2025-10-20T10:42:26Z",
    "offline_duration": "1h14m0s",
    "virtual_network_id": "网络UUID"
  },
  "created_at": "2025-10-20T11:56:26Z",
  "timestamp": "2025-10-20T12:10:00Z"
}
```

### 3.3 集成第三方服务

#### Slack
```bash
Webhook URL: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

#### Discord
```bash
Webhook URL: https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN
```

#### Microsoft Teams
```bash
Webhook URL: https://outlook.office.com/webhook/YOUR_WEBHOOK_URL
```

#### PagerDuty
```bash
Webhook URL: https://events.pagerduty.com/v2/enqueue
需要额外配置: Integration Key
```

---

## 4. 告警通知流程

```
┌─────────────────┐
│  Alert Service  │ (每分钟检查一次)
└────────┬────────┘
         │
         ├──> ThresholdChecker: 检查离线设备
         ├──> ThresholdChecker: 检查高延迟会话
         │
         ├──> AlertGenerator: 生成告警记录
         │
         └──> NotificationScheduler: 调度通知
              ├──> EmailNotifier: 发送邮件
              └──> WebhookNotifier: 发送HTTP请求
```

---

## 5. 测试工具

### 5.1 Webhook接收器

使用提供的 `test-webhook-receiver.py`：

```bash
# 启动接收器
python3 test-webhook-receiver.py 7777

# 发送测试通知
curl -X POST http://localhost:7777/webhook \
  -H "Content-Type: application/json" \
  -d @alert-payload.json
```

### 5.2 邮件测试工具

**MailHog** (本地SMTP测试服务器):
```bash
# Docker运行MailHog
docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog

# 配置
SMTP_HOST: localhost
SMTP_PORT: 1025
SMTP_USER: (留空)
SMTP_PASSWORD: (留空)

# 查看邮件: http://localhost:8025
```

**Mailtrap** (在线测试):
1. 注册 https://mailtrap.io
2. 获取SMTP凭据
3. 配置Alert Service使用Mailtrap设置

---

## 6. 监控和调试

### 6.1 查看告警记录

```sql
SELECT id, device_id, severity, type, title,
       created_at, status
FROM alerts
ORDER BY created_at DESC
LIMIT 10;
```

### 6.2 查看Alert Service日志

```bash
docker logs edgelink-alert-service --tail 100 -f
```

### 6.3 常见问题

#### 问题1: 没有生成告警
- 检查设备的`last_seen_at`字段是否更新
- 检查设备的`online`字段是否为true
- 查看Alert Service日志中的"issues_found"计数

#### 问题2: 邮件发送失败
- 验证SMTP凭据是否正确
- 检查SMTP服务器端口是否开放
- 查看日志中的错误信息

#### 问题3: Webhook不触发
- 验证Webhook URL是否可访问
- 检查NotificationScheduler配置
- 查看日志中的HTTP错误代码

---

## 7. 生产环境建议

1. **SMTP配置**:
   - 使用专业的邮件发送服务（SendGrid, Mailgun, Amazon SES）
   - 设置发送速率限制
   - 配置邮件模板

2. **Webhook配置**:
   - 使用HTTPS端点
   - 实现重试机制
   - 添加签名验证

3. **告警去重**:
   - 实现告警聚合（相同设备在短时间内只发送一次）
   - 添加静默期配置
   - 支持告警确认和关闭

4. **通知渠道**:
   - 根据严重程度选择不同通知渠道
   - CRITICAL → 邮件 + Webhook + 短信
   - HIGH → 邮件 + Webhook
   - MEDIUM/LOW → 仅Webhook

---

## 8. 测试结果

### 告警生成测试 ✅
- 3个测试设备成功创建
- 6条告警成功生成
- 严重程度分级正确

### Webhook通知测试 ✅
- Webhook接收器成功接收通知
- 所有字段正确解析
- 元数据完整传输

### 下一步
- 配置实际的SMTP服务器进行邮件测试
- 集成第三方告警平台（PagerDuty, Opsgenie等）
- 实现告警管理UI界面
