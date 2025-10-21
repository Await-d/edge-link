package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Logging  LoggingConfig
	Metrics  MetricsConfig
	Email    EmailConfig
	Alert    AlertConfig
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string
	Format     string // "json" or "console"
	OutputPath string
}

// MetricsConfig 监控指标配置
type MetricsConfig struct {
	Enabled bool
	Port    int
}

// EmailConfig 邮件配置
type EmailConfig struct {
	// 邮件提供商类型: "smtp", "sendgrid", "mailgun", "ses"
	Provider string

	// SMTP通用配置
	SMTP SMTPConfig

	// 第三方服务配置
	SendGrid SendGridConfig
	Mailgun  MailgunConfig
	SES      SESConfig

	// 发送配置
	FromAddress  string
	FromName     string
	ReplyTo      string

	// 队列和重试配置
	QueueSize    int
	MaxRetries   int
	RetryDelay   time.Duration

	// 速率限制
	RateLimit    int           // 每分钟最大发送数
	RatePeriod   time.Duration // 速率限制周期

	// 模板配置
	TemplateDir  string
	DefaultLang  string
}

// SMTPConfig SMTP服务器配置
type SMTPConfig struct {
	Host         string
	Port         int
	Username     string
	Password     string
	UseTLS       bool
	UseStartTLS  bool
	SkipVerify   bool // 跳过TLS证书验证(仅开发环境)
	Timeout      time.Duration
}

// SendGridConfig SendGrid API配置
type SendGridConfig struct {
	APIKey      string
	SandboxMode bool
}

// MailgunConfig Mailgun API配置
type MailgunConfig struct {
	Domain      string
	APIKey      string
	BaseURL     string // eu域名可自定义
}

// SESConfig Amazon SES配置
type SESConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	ConfigSet       string // SES配置集(可选)
}

// AlertConfig 告警配置
type AlertConfig struct {
	// 去重配置
	DedupeWindow        time.Duration // 去重时间窗口
	SilentPeriod        time.Duration // 静默期
	EscalationThreshold int           // 升级阈值
	LockTimeout         time.Duration // 分布式锁超时

	// 检查配置
	CheckInterval       time.Duration // 检查间隔
	DeviceOfflineThreshold time.Duration // 设备离线阈值
	HighLatencyThreshold   int           // 高延迟阈值（毫秒）
}

// LoadConfig 从环境变量加载配置（Fx兼容）
func LoadConfig() (*Config, error) {
	return Load()
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "edgelink"),
			Password:        getEnv("DB_PASSWORD", "edgelink_dev_password"),
			DBName:          getEnv("DB_NAME", "edgelink"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", "edgelink_redis_password"),
			DB:       getEnvAsInt("REDIS_DB", 0),
			PoolSize: getEnvAsInt("REDIS_POOL_SIZE", 10),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			OutputPath: getEnv("LOG_OUTPUT", "stdout"),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvAsBool("METRICS_ENABLED", true),
			Port:    getEnvAsInt("METRICS_PORT", 9090),
		},
		Email: EmailConfig{
			Provider: getEnv("EMAIL_PROVIDER", "smtp"),

			SMTP: SMTPConfig{
				Host:        getEnv("SMTP_HOST", "smtp.gmail.com"),
				Port:        getEnvAsInt("SMTP_PORT", 587),
				Username:    getEnv("SMTP_USERNAME", ""),
				Password:    getEnv("SMTP_PASSWORD", ""),
				UseTLS:      getEnvAsBool("SMTP_USE_TLS", false),
				UseStartTLS: getEnvAsBool("SMTP_USE_STARTTLS", true),
				SkipVerify:  getEnvAsBool("SMTP_SKIP_VERIFY", false),
				Timeout:     getEnvAsDuration("SMTP_TIMEOUT", 10*time.Second),
			},

			SendGrid: SendGridConfig{
				APIKey:      getEnv("SENDGRID_API_KEY", ""),
				SandboxMode: getEnvAsBool("SENDGRID_SANDBOX_MODE", false),
			},

			Mailgun: MailgunConfig{
				Domain:  getEnv("MAILGUN_DOMAIN", ""),
				APIKey:  getEnv("MAILGUN_API_KEY", ""),
				BaseURL: getEnv("MAILGUN_BASE_URL", "https://api.mailgun.net"),
			},

			SES: SESConfig{
				Region:          getEnv("AWS_SES_REGION", "us-east-1"),
				AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
				SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
				ConfigSet:       getEnv("AWS_SES_CONFIG_SET", ""),
			},

			FromAddress:  getEnv("EMAIL_FROM_ADDRESS", "noreply@edgelink.com"),
			FromName:     getEnv("EMAIL_FROM_NAME", "EdgeLink Alert"),
			ReplyTo:      getEnv("EMAIL_REPLY_TO", ""),

			QueueSize:    getEnvAsInt("EMAIL_QUEUE_SIZE", 1000),
			MaxRetries:   getEnvAsInt("EMAIL_MAX_RETRIES", 3),
			RetryDelay:   getEnvAsDuration("EMAIL_RETRY_DELAY", 5*time.Second),

			RateLimit:    getEnvAsInt("EMAIL_RATE_LIMIT", 100),
			RatePeriod:   getEnvAsDuration("EMAIL_RATE_PERIOD", time.Minute),

			TemplateDir:  getEnv("EMAIL_TEMPLATE_DIR", "./templates/email"),
			DefaultLang:  getEnv("EMAIL_DEFAULT_LANG", "zh-CN"),
		},
		Alert: AlertConfig{
			// 去重配置
			DedupeWindow:        getEnvAsDuration("ALERT_DEDUPE_WINDOW", 30*time.Minute),
			SilentPeriod:        getEnvAsDuration("ALERT_SILENT_PERIOD", 5*time.Minute),
			EscalationThreshold: getEnvAsInt("ALERT_ESCALATION_THRESHOLD", 10),
			LockTimeout:         getEnvAsDuration("ALERT_LOCK_TIMEOUT", 5*time.Second),

			// 检查配置
			CheckInterval:          getEnvAsDuration("ALERT_CHECK_INTERVAL", 1*time.Minute),
			DeviceOfflineThreshold: getEnvAsDuration("ALERT_DEVICE_OFFLINE_THRESHOLD", 5*time.Minute),
			HighLatencyThreshold:   getEnvAsInt("ALERT_HIGH_LATENCY_THRESHOLD", 200),
		},
	}, nil
}

// DSN 生成PostgreSQL连接字符串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// RedisAddr 生成Redis地址
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// 辅助函数
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
