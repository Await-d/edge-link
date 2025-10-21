# 邮件系统新增依赖

本次实现添加了以下Go依赖包,需要运行`go mod tidy`来安装:

## 核心依赖

### gopkg.in/gomail.v2
- **用途**: SMTP邮件发送库
- **版本**: v2.0.0-20160411212932-81ebce5c23df
- **功能**: 支持SMTP认证、TLS/StartTLS、HTML/纯文本邮件
- **安装**:
```bash
go get gopkg.in/gomail.v2
```

## 可选依赖(未来扩展)

以下依赖用于实现第三方邮件服务集成,当前为占位实现,需要时再安装:

### SendGrid
```bash
go get github.com/sendgrid/sendgrid-go
```

### Mailgun
```bash
go get github.com/mailgun/mailgun-go/v4
```

### Amazon SES
```bash
go get github.com/aws/aws-sdk-go/service/ses
```

## 安装步骤

在backend目录执行:

```bash
cd /home/await/project/edge-link/backend
go mod tidy
```

这将自动下载并安装所有必需的依赖包。

## 依赖说明

### gomail.v2 特性
- 支持多种SMTP认证方式(PLAIN, LOGIN, CRAM-MD5)
- 自动处理TLS和StartTLS连接
- 支持HTML和纯文本多部分邮件
- 支持附件和内嵌图片
- 连接池和批量发送
- 轻量级,无额外依赖

### 为什么选择gomail.v2
1. 成熟稳定,广泛使用
2. API简洁易用
3. 支持所有主流SMTP服务器
4. 性能良好,适合生产环境
5. 活跃维护,问题少

## 构建Docker镜像

更新Dockerfile以包含新依赖:

```dockerfile
# 在Dockerfile中确保go mod download
RUN go mod download
RUN go build -o alert-service ./cmd/alert-service
```

重新构建镜像:
```bash
docker-compose build alert-service
```
