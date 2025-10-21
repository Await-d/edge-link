# SMTP邮件通知配置指南

## 概述

EdgeLink Alert Service支持多种邮件提供商,包括:
- SMTP (通用,支持Gmail, Outlook等)
- SendGrid API
- Mailgun API
- Amazon SES

## 配置方式

### 1. SMTP通用配置 (推荐用于开发环境)

#### Gmail示例
```bash
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password  # 使用应用专用密码,不是账户密码
SMTP_USE_TLS=false
SMTP_USE_STARTTLS=true
SMTP_SKIP_VERIFY=false
```

**获取Gmail应用专用密码:**
1. 访问 https://myaccount.google.com/security
2. 启用两步验证
3. 生成"应用专用密码"
4. 使用生成的16位密码作为SMTP_PASSWORD

#### Office 365 / Outlook示例
```bash
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp-mail.outlook.com
SMTP_PORT=587
SMTP_USERNAME=your-email@outlook.com
SMTP_PASSWORD=your-password
SMTP_USE_STARTTLS=true
```

#### 自建SMTP服务器
```bash
EMAIL_PROVIDER=smtp
SMTP_HOST=mail.yourcompany.com
SMTP_PORT=25  # 或 465(SSL), 587(TLS)
SMTP_USERNAME=alerts@yourcompany.com
SMTP_PASSWORD=your-password
SMTP_USE_TLS=true  # 如果使用465端口
SMTP_SKIP_VERIFY=false  # 生产环境必须为false
```

### 2. SendGrid配置 (推荐用于生产环境)

```bash
EMAIL_PROVIDER=sendgrid
SENDGRID_API_KEY=SG.xxxxxxxxxxxxxxxxxxxxx
SENDGRID_SANDBOX_MODE=false  # 测试时设为true
```

**获取SendGrid API Key:**
1. 注册 https://sendgrid.com/
2. Settings -> API Keys -> Create API Key
3. 选择"Full Access"或"Restricted Access"
4. 复制生成的API密钥

**优点:**
- 高送达率和可靠性
- 详细的发送统计和分析
- 每月免费100封邮件

### 3. Mailgun配置

```bash
EMAIL_PROVIDER=mailgun
MAILGUN_DOMAIN=mg.yourdomain.com
MAILGUN_API_KEY=key-xxxxxxxxxxxxxxxxxxxxxxxx
MAILGUN_BASE_URL=https://api.mailgun.net  # EU区域使用 https://api.eu.mailgun.net
```

**获取Mailgun配置:**
1. 注册 https://mailgun.com/
2. 添加并验证域名
3. Settings -> API Keys -> Private API key
4. 复制域名和API密钥

**优点:**
- 强大的API功能
- 灵活的域名配置
- 每月免费5000封邮件

### 4. Amazon SES配置

```bash
EMAIL_PROVIDER=ses
AWS_SES_REGION=us-east-1
AWS_ACCESS_KEY_ID=AKIAXXXXXXXXXXXXXXXX
AWS_SECRET_ACCESS_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
AWS_SES_CONFIG_SET=my-config-set  # 可选
```

**配置Amazon SES:**
1. AWS Console -> SES
2. 验证发件人邮箱或域名
3. 申请移出沙箱环境(生产环境必需)
4. IAM -> 创建用户,授予SES发送权限
5. 生成访问密钥

**优点:**
- 极低成本($0.10/1000封)
- 高度可扩展
- 与AWS生态集成

## 通用配置项

以下配置适用于所有邮件提供商:

```bash
# 发件人信息
EMAIL_FROM_ADDRESS=noreply@edgelink.com
EMAIL_FROM_NAME=EdgeLink Alert System
EMAIL_REPLY_TO=support@edgelink.com

# 队列和重试配置
EMAIL_QUEUE_SIZE=1000          # 邮件队列大小
EMAIL_MAX_RETRIES=3            # 失败后最大重试次数
EMAIL_RETRY_DELAY=5s           # 重试延迟时间

# 速率限制
EMAIL_RATE_LIMIT=100           # 每分钟最大发送数
EMAIL_RATE_PERIOD=1m           # 速率限制周期

# 模板配置
EMAIL_TEMPLATE_DIR=./templates/email  # 模板文件目录
EMAIL_DEFAULT_LANG=zh-CN              # 默认语言
```

## Docker Compose配置

在`docker-compose.yml`中添加环境变量:

```yaml
alert-service:
  environment:
    # 选择一种提供商配置
    - EMAIL_PROVIDER=smtp

    # SMTP配置
    - SMTP_HOST=smtp.gmail.com
    - SMTP_PORT=587
    - SMTP_USERNAME=${GMAIL_USERNAME}
    - SMTP_PASSWORD=${GMAIL_APP_PASSWORD}
    - SMTP_USE_STARTTLS=true

    # 通用配置
    - EMAIL_FROM_ADDRESS=noreply@edgelink.com
    - EMAIL_FROM_NAME=EdgeLink Alert
    - EMAIL_QUEUE_SIZE=1000
    - EMAIL_MAX_RETRIES=3
```

## 环境变量文件

创建`.env`文件管理敏感信息:

```bash
# .env
GMAIL_USERNAME=your-email@gmail.com
GMAIL_APP_PASSWORD=your-16-char-app-password

# 或 SendGrid
SENDGRID_API_KEY=SG.xxxxxxxxxxxx

# 或 Mailgun
MAILGUN_API_KEY=key-xxxxxxxxxxxx
MAILGUN_DOMAIN=mg.yourdomain.com
```

在`docker-compose.yml`中引用:
```yaml
env_file:
  - .env
```

## 邮件模板自定义

默认使用内置模板,如需自定义:

1. 创建模板目录: `mkdir -p templates/email`
2. 创建HTML模板文件: `templates/email/alert.html`
3. 配置模板路径: `EMAIL_TEMPLATE_DIR=./templates/email`

模板可用变量:
- `{{.Title}}` - 告警标题
- `{{.Message}}` - 告警消息
- `{{.Severity}}` - 严重程度
- `{{.SeverityColor}}` - 严重程度颜色
- `{{.AlertType}}` - 告警类型
- `{{.CreatedAt}}` - 创建时间
- `{{.DeviceID}}` - 设备ID(可选)

## 测试邮件发送

发送测试邮件验证配置:

```bash
# 使用curl测试API(假设有测试端点)
curl -X POST http://localhost:8080/api/v1/email/test \
  -H "Content-Type: application/json" \
  -d '{"to": ["test@example.com"]}'
```

## 故障排查

### 1. Gmail "Less secure app access" 错误
**解决方案:** 使用应用专用密码,不要使用账户密码

### 2. 连接超时
**检查:**
- 防火墙是否允许出站SMTP端口(25/465/587)
- SMTP服务器地址是否正确
- 网络连接是否正常

### 3. 认证失败
**检查:**
- 用户名和密码是否正确
- 是否需要使用应用专用密码
- SMTP服务器是否要求特定的认证方式

### 4. TLS/SSL错误
**解决方案:**
- 确认端口和TLS配置匹配(587用StartTLS, 465用TLS)
- 开发环境可临时设置`SMTP_SKIP_VERIFY=true`
- 生产环境确保证书有效

### 5. 速率限制
**解决方案:**
- 调整`EMAIL_RATE_LIMIT`和`EMAIL_RATE_PERIOD`
- 使用专业邮件服务(SendGrid/Mailgun)提高限额
- 实施邮件聚合策略减少发送频率

## 监控和日志

邮件发送统计可通过API获取:

```bash
# 获取邮件发送统计
curl http://localhost:8080/api/v1/email/stats
```

返回示例:
```json
{
  "total_sent": 150,
  "total_failed": 3,
  "total_retried": 5,
  "queue_length": 2,
  "last_sent_time": "2025-10-20T10:30:00Z",
  "last_error_time": "2025-10-20T09:15:00Z",
  "last_error": "connection timeout"
}
```

## 生产环境最佳实践

1. **使用专业邮件服务** - SendGrid/Mailgun/SES而非SMTP
2. **配置速率限制** - 避免触发提供商限制
3. **启用重试机制** - 确保关键告警不丢失
4. **监控发送状态** - 定期检查邮件历史记录
5. **定期清理历史** - 避免数据库膨胀
6. **使用环境变量** - 不要硬编码敏感信息
7. **验证发件人域名** - 提高送达率和信誉度
8. **实施邮件去重** - 避免重复告警骚扰用户

## 安全注意事项

1. **不要在代码中硬编码密码**
2. **使用环境变量或密钥管理服务**
3. **定期轮换API密钥**
4. **限制SMTP用户权限(仅发送)**
5. **启用TLS/SSL加密传输**
6. **生产环境禁用`SMTP_SKIP_VERIFY`**
7. **审计邮件发送记录**
8. **限制收件人数量防止滥用**

## 参考资源

- Gmail SMTP: https://support.google.com/mail/answer/7126229
- SendGrid Docs: https://docs.sendgrid.com/
- Mailgun Docs: https://documentation.mailgun.com/
- Amazon SES: https://docs.aws.amazon.com/ses/
