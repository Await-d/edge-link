package notifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/edgelink/backend/internal/config"
	"github.com/edgelink/backend/internal/domain"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

// EmailProvider 邮件提供商接口
type EmailProvider interface {
	Send(ctx context.Context, msg *EmailMessage) error
	Name() string
}

// EmailMessage 邮件消息
type EmailMessage struct {
	To          []string
	Subject     string
	HTMLBody    string
	TextBody    string
	Attachments []string
	Headers     map[string]string
}

// EmailTask 邮件发送任务
type EmailTask struct {
	Message   *EmailMessage
	Alert     *domain.Alert
	Retries   int
	CreatedAt time.Time
}

// EmailNotifier 邮件通知器
type EmailNotifier struct {
	config     *config.EmailConfig
	provider   EmailProvider
	templates  *template.Template
	logger     *zap.Logger

	// 邮件队列和工作池
	queue      chan *EmailTask
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc

	// 速率限制
	rateLimiter *RateLimiter

	// 统计信息
	stats EmailStats
	mu    sync.RWMutex
}

// EmailStats 邮件发送统计
type EmailStats struct {
	TotalSent     int64
	TotalFailed   int64
	TotalRetried  int64
	QueueLength   int
	LastSentTime  time.Time
	LastErrorTime time.Time
	LastError     string
}

// RateLimiter 速率限制器
type RateLimiter struct {
	tokens   int
	maxToken int
	refill   time.Duration
	mu       sync.Mutex
	ticker   *time.Ticker
	done     chan struct{}
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(maxTokens int, refillPeriod time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:   maxTokens,
		maxToken: maxTokens,
		refill:   refillPeriod,
		done:     make(chan struct{}),
	}

	// 启动令牌补充协程
	rl.ticker = time.NewTicker(refillPeriod)
	go rl.refillTokens()

	return rl
}

// refillTokens 定期补充令牌
func (rl *RateLimiter) refillTokens() {
	for {
		select {
		case <-rl.ticker.C:
			rl.mu.Lock()
			rl.tokens = rl.maxToken
			rl.mu.Unlock()
		case <-rl.done:
			rl.ticker.Stop()
			return
		}
	}
}

// Allow 检查是否允许发送(消耗一个令牌)
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// Stop 停止速率限制器
func (rl *RateLimiter) Stop() {
	close(rl.done)
}

// NewEmailNotifier 创建邮件通知器
func NewEmailNotifier(cfg *config.Config, logger *zap.Logger) (*EmailNotifier, error) {
	emailCfg := &cfg.Email

	// 根据配置选择邮件提供商
	var provider EmailProvider
	var err error

	switch emailCfg.Provider {
	case "smtp", "":
		provider, err = NewSMTPProvider(emailCfg, logger)
	case "sendgrid":
		provider, err = NewSendGridProvider(emailCfg, logger)
	case "mailgun":
		provider, err = NewMailgunProvider(emailCfg, logger)
	case "ses":
		provider, err = NewSESProvider(emailCfg, logger)
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", emailCfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create email provider: %w", err)
	}

	// 加载邮件模板
	templates, err := loadEmailTemplates(emailCfg.TemplateDir)
	if err != nil {
		logger.Warn("Failed to load email templates, using inline templates",
			zap.Error(err),
		)
		templates = nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	notifier := &EmailNotifier{
		config:      emailCfg,
		provider:    provider,
		templates:   templates,
		logger:      logger,
		queue:       make(chan *EmailTask, emailCfg.QueueSize),
		ctx:         ctx,
		cancel:      cancel,
		rateLimiter: NewRateLimiter(emailCfg.RateLimit, emailCfg.RatePeriod),
		stats:       EmailStats{},
	}

	// 启动工作协程处理邮件队列
	numWorkers := 3 // 并发发送邮件的worker数量
	for i := 0; i < numWorkers; i++ {
		notifier.wg.Add(1)
		go notifier.worker(i)
	}

	logger.Info("Email notifier initialized",
		zap.String("provider", provider.Name()),
		zap.Int("queue_size", emailCfg.QueueSize),
		zap.Int("workers", numWorkers),
	)

	return notifier, nil
}

// loadEmailTemplates 加载邮件模板
func loadEmailTemplates(templateDir string) (*template.Template, error) {
	if templateDir == "" {
		return nil, fmt.Errorf("template directory not specified")
	}

	// 检查目录是否存在
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory does not exist: %s", templateDir)
	}

	// 加载所有.html模板文件
	pattern := filepath.Join(templateDir, "*.html")
	tmpl, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return tmpl, nil
}

// SendAlert 发送告警邮件(异步入队)
func (en *EmailNotifier) SendAlert(ctx context.Context, alert *domain.Alert, recipients []string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// 渲染邮件内容
	subject := fmt.Sprintf("[EdgeLink Alert] %s", alert.Title)
	htmlBody, err := en.renderEmailTemplate(alert)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// 创建邮件消息
	message := &EmailMessage{
		To:       recipients,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: en.extractTextFromHTML(htmlBody),
	}

	// 创建邮件任务并入队
	task := &EmailTask{
		Message:   message,
		Alert:     alert,
		Retries:   0,
		CreatedAt: time.Now(),
	}

	select {
	case en.queue <- task:
		en.logger.Debug("Email task queued",
			zap.String("alert_id", alert.ID.String()),
			zap.Strings("recipients", recipients),
		)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("email queue is full")
	}
}

// worker 邮件发送工作协程
func (en *EmailNotifier) worker(id int) {
	defer en.wg.Done()

	en.logger.Info("Email worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-en.ctx.Done():
			en.logger.Info("Email worker stopping", zap.Int("worker_id", id))
			return

		case task := <-en.queue:
			// 更新队列长度统计
			en.mu.Lock()
			en.stats.QueueLength = len(en.queue)
			en.mu.Unlock()

			// 速率限制检查
			if !en.rateLimiter.Allow() {
				en.logger.Warn("Rate limit reached, re-queuing email",
					zap.Int("worker_id", id),
				)
				// 重新入队(带延迟)
				time.Sleep(time.Second)
				select {
				case en.queue <- task:
				default:
					en.logger.Error("Failed to re-queue email after rate limit")
				}
				continue
			}

			// 发送邮件
			if err := en.sendEmail(task); err != nil {
				en.handleSendError(task, err)
			} else {
				en.handleSendSuccess(task)
			}
		}
	}
}

// sendEmail 实际发送邮件
func (en *EmailNotifier) sendEmail(task *EmailTask) error {
	ctx, cancel := context.WithTimeout(en.ctx, 30*time.Second)
	defer cancel()

	err := en.provider.Send(ctx, task.Message)
	if err != nil {
		return err
	}

	en.logger.Info("Email sent successfully",
		zap.String("alert_id", task.Alert.ID.String()),
		zap.Strings("recipients", task.Message.To),
		zap.Int("retries", task.Retries),
	)

	return nil
}

// handleSendSuccess 处理发送成功
func (en *EmailNotifier) handleSendSuccess(task *EmailTask) {
	en.mu.Lock()
	en.stats.TotalSent++
	en.stats.LastSentTime = time.Now()
	en.mu.Unlock()
}

// handleSendError 处理发送错误
func (en *EmailNotifier) handleSendError(task *EmailTask, err error) {
	en.logger.Error("Failed to send email",
		zap.String("alert_id", task.Alert.ID.String()),
		zap.Error(err),
		zap.Int("retries", task.Retries),
	)

	en.mu.Lock()
	en.stats.TotalFailed++
	en.stats.LastErrorTime = time.Now()
	en.stats.LastError = err.Error()
	en.mu.Unlock()

	// 重试逻辑
	if task.Retries < en.config.MaxRetries {
		task.Retries++

		en.mu.Lock()
		en.stats.TotalRetried++
		en.mu.Unlock()

		en.logger.Info("Retrying email send",
			zap.String("alert_id", task.Alert.ID.String()),
			zap.Int("retry_attempt", task.Retries),
			zap.Duration("delay", en.config.RetryDelay),
		)

		// 延迟后重新入队
		time.Sleep(en.config.RetryDelay)
		select {
		case en.queue <- task:
		default:
			en.logger.Error("Failed to re-queue email for retry",
				zap.String("alert_id", task.Alert.ID.String()),
			)
		}
	}
}

// renderEmailTemplate 渲染邮件模板
func (en *EmailNotifier) renderEmailTemplate(alert *domain.Alert) (string, error) {
	data := map[string]interface{}{
		"Title":         alert.Title,
		"Message":       alert.Message,
		"Severity":      alert.Severity,
		"SeverityColor": en.getSeverityColor(alert.Severity),
		"AlertType":     alert.Type,
		"CreatedAt":     alert.CreatedAt.Format("2006-01-02 15:04:05"),
		"DeviceID":      "",
	}

	if alert.DeviceID != nil {
		data["DeviceID"] = alert.DeviceID.String()
	}

	// 如果有外部模板文件,优先使用
	if en.templates != nil {
		return en.renderTemplateFile("alert.html", data)
	}

	// 否则使用内置模板
	return en.renderInlineTemplate(data)
}

// renderTemplateFile 使用外部模板文件渲染
func (en *EmailNotifier) renderTemplateFile(name string, data interface{}) (string, error) {
	var buf []byte
	tmpl := en.templates.Lookup(name)
	if tmpl == nil {
		return "", fmt.Errorf("template not found: %s", name)
	}

	writer := &bytesWriter{buf: buf}
	if err := tmpl.Execute(writer, data); err != nil {
		return "", err
	}

	return string(writer.buf), nil
}

// bytesWriter 实现io.Writer接口
type bytesWriter struct {
	buf []byte
}

func (w *bytesWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

// renderInlineTemplate 使用内置模板渲染
func (en *EmailNotifier) renderInlineTemplate(data map[string]interface{}) (string, error) {
	tmplStr := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background-color: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .header {
            background-color: {{.SeverityColor}};
            color: white;
            padding: 20px;
        }
        .header h2 {
            margin: 0 0 10px 0;
            font-size: 24px;
        }
        .severity-badge {
            display: inline-block;
            padding: 5px 12px;
            background-color: rgba(255,255,255,0.2);
            border-radius: 4px;
            font-size: 14px;
            font-weight: 600;
        }
        .content {
            padding: 30px;
        }
        .alert-info {
            background-color: #f9f9f9;
            padding: 20px;
            margin: 15px 0;
            border-left: 4px solid {{.SeverityColor}};
            border-radius: 4px;
        }
        .alert-info h3 {
            margin: 0 0 10px 0;
            font-size: 16px;
            color: #555;
        }
        .alert-info p {
            margin: 5px 0;
            color: #666;
        }
        .alert-message {
            background-color: white;
            padding: 15px;
            border-radius: 4px;
            margin-top: 10px;
            font-size: 15px;
            line-height: 1.6;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 2px solid #e9e9e9;
            font-size: 13px;
            color: #999;
            text-align: center;
        }
        .footer a {
            color: #4CAF50;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>{{.Title}}</h2>
            <span class="severity-badge">严重程度: {{.Severity}}</span>
        </div>
        <div class="content">
            <div class="alert-info">
                <h3>告警消息</h3>
                <div class="alert-message">{{.Message}}</div>
            </div>
            <div class="alert-info">
                <h3>告警详情</h3>
                <p><strong>类型:</strong> {{.AlertType}}</p>
                <p><strong>时间:</strong> {{.CreatedAt}}</p>
                {{if .DeviceID}}<p><strong>设备ID:</strong> {{.DeviceID}}</p>{{end}}
            </div>
        </div>
        <div class="footer">
            <p>此邮件由EdgeLink告警系统自动发送,请勿直接回复</p>
            <p>如需了解更多信息或采取行动,请访问 <a href="#">EdgeLink管理控制台</a></p>
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("alert_inline").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	writer := &bytesWriter{}
	if err := tmpl.Execute(writer, data); err != nil {
		return "", err
	}

	return string(writer.buf), nil
}

// getSeverityColor 获取严重程度对应的颜色
func (en *EmailNotifier) getSeverityColor(severity domain.Severity) string {
	switch severity {
	case domain.SeverityCritical:
		return "#d32f2f" // 红色
	case domain.SeverityHigh:
		return "#f57c00" // 橙色
	case domain.SeverityMedium:
		return "#fbc02d" // 黄色
	case domain.SeverityLow:
		return "#1976d2" // 蓝色
	default:
		return "#757575" // 灰色
	}
}

// extractTextFromHTML 从HTML提取纯文本(简单实现)
func (en *EmailNotifier) extractTextFromHTML(html string) string {
	// TODO: 实现更完善的HTML到文本转换
	// 这里简单实现,生产环境建议使用专门的库
	return "请在支持HTML的邮件客户端中查看此邮件"
}

// GetStats 获取邮件发送统计
func (en *EmailNotifier) GetStats() EmailStats {
	en.mu.RLock()
	defer en.mu.RUnlock()

	stats := en.stats
	stats.QueueLength = len(en.queue)
	return stats
}

// Stop 停止邮件通知器
func (en *EmailNotifier) Stop() {
	en.logger.Info("Stopping email notifier")

	// 停止接收新任务
	en.cancel()

	// 等待队列中的任务处理完成(最多等待30秒)
	done := make(chan struct{})
	go func() {
		en.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		en.logger.Info("All email tasks completed")
	case <-time.After(30 * time.Second):
		en.logger.Warn("Email worker shutdown timeout, some tasks may be lost")
	}

	// 停止速率限制器
	en.rateLimiter.Stop()

	// 关闭队列
	close(en.queue)
}

// SMTPProvider SMTP邮件提供商
type SMTPProvider struct {
	config *config.EmailConfig
	dialer *gomail.Dialer
	logger *zap.Logger
}

// NewSMTPProvider 创建SMTP提供商
func NewSMTPProvider(cfg *config.EmailConfig, logger *zap.Logger) (*SMTPProvider, error) {
	if cfg.SMTP.Host == "" {
		return nil, fmt.Errorf("SMTP host not configured")
	}

	dialer := gomail.NewDialer(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
	)

	// TLS配置
	if cfg.SMTP.UseTLS || cfg.SMTP.UseStartTLS {
		dialer.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.SMTP.SkipVerify,
			ServerName:         cfg.SMTP.Host,
		}
		// gomail.v2会自动处理STARTTLS，无需额外配置
	}

	return &SMTPProvider{
		config: cfg,
		dialer: dialer,
		logger: logger,
	}, nil
}

// Send 发送邮件
func (p *SMTPProvider) Send(ctx context.Context, msg *EmailMessage) error {
	// 如果没有配置密码,跳过发送
	if p.config.SMTP.Password == "" {
		p.logger.Warn("SMTP password not configured, skipping email send")
		return nil
	}

	m := gomail.NewMessage()

	// 设置发件人
	fromAddr := p.config.FromAddress
	if p.config.FromName != "" {
		m.SetHeader("From", m.FormatAddress(p.config.FromAddress, p.config.FromName))
	} else {
		m.SetHeader("From", fromAddr)
	}

	// 设置收件人
	m.SetHeader("To", msg.To...)

	// 设置回复地址
	if p.config.ReplyTo != "" {
		m.SetHeader("Reply-To", p.config.ReplyTo)
	}

	// 设置主题
	m.SetHeader("Subject", msg.Subject)

	// 设置邮件内容
	if msg.HTMLBody != "" {
		m.SetBody("text/html", msg.HTMLBody)
		if msg.TextBody != "" {
			m.AddAlternative("text/plain", msg.TextBody)
		}
	} else {
		m.SetBody("text/plain", msg.TextBody)
	}

	// 添加自定义头部
	for k, v := range msg.Headers {
		m.SetHeader(k, v)
	}

	// 发送邮件
	if err := p.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}

// Name 提供商名称
func (p *SMTPProvider) Name() string {
	return "smtp"
}

// SendGridProvider SendGrid邮件提供商(占位实现)
type SendGridProvider struct {
	config *config.EmailConfig
	logger *zap.Logger
}

func NewSendGridProvider(cfg *config.EmailConfig, logger *zap.Logger) (*SendGridProvider, error) {
	if cfg.SendGrid.APIKey == "" {
		return nil, fmt.Errorf("SendGrid API key not configured")
	}
	return &SendGridProvider{config: cfg, logger: logger}, nil
}

func (p *SendGridProvider) Send(ctx context.Context, msg *EmailMessage) error {
	// TODO: 实现SendGrid API集成
	p.logger.Warn("SendGrid provider not fully implemented yet")
	return fmt.Errorf("SendGrid provider not implemented")
}

func (p *SendGridProvider) Name() string {
	return "sendgrid"
}

// MailgunProvider Mailgun邮件提供商(占位实现)
type MailgunProvider struct {
	config *config.EmailConfig
	logger *zap.Logger
}

func NewMailgunProvider(cfg *config.EmailConfig, logger *zap.Logger) (*MailgunProvider, error) {
	if cfg.Mailgun.APIKey == "" {
		return nil, fmt.Errorf("Mailgun API key not configured")
	}
	return &MailgunProvider{config: cfg, logger: logger}, nil
}

func (p *MailgunProvider) Send(ctx context.Context, msg *EmailMessage) error {
	// TODO: 实现Mailgun API集成
	p.logger.Warn("Mailgun provider not fully implemented yet")
	return fmt.Errorf("Mailgun provider not implemented")
}

func (p *MailgunProvider) Name() string {
	return "mailgun"
}

// SESProvider Amazon SES邮件提供商(占位实现)
type SESProvider struct {
	config *config.EmailConfig
	logger *zap.Logger
}

func NewSESProvider(cfg *config.EmailConfig, logger *zap.Logger) (*SESProvider, error) {
	if cfg.SES.AccessKeyID == "" || cfg.SES.SecretAccessKey == "" {
		return nil, fmt.Errorf("AWS credentials not configured")
	}
	return &SESProvider{config: cfg, logger: logger}, nil
}

func (p *SESProvider) Send(ctx context.Context, msg *EmailMessage) error {
	// TODO: 实现Amazon SES SDK集成
	p.logger.Warn("SES provider not fully implemented yet")
	return fmt.Errorf("SES provider not implemented")
}

func (p *SESProvider) Name() string {
	return "ses"
}
