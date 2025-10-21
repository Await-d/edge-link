package logger

import (
	"github.com/edgelink/backend/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志器包装
type Logger struct {
	*zap.Logger
}

// NewLogger 创建日志器（Fx兼容）
func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	logger, err := New(&cfg.Logging)
	if err != nil {
		return nil, err
	}
	return logger.Logger, nil
}

// New 创建日志器
func New(cfg *config.LoggingConfig) (*Logger, error) {
	var zapConfig zap.Config

	if cfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// 设置日志级别
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// 设置输出路径
	if cfg.OutputPath != "stdout" {
		zapConfig.OutputPaths = []string{cfg.OutputPath}
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{logger}, nil
}

// WithContext 添加上下文字段
func (l *Logger) WithContext(fields ...zap.Field) *Logger {
	return &Logger{l.With(fields...)}
}

// Close 关闭日志器
func (l *Logger) Close() error {
	return l.Sync()
}

// 快捷方法
func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.Sugar().Debugf(msg, args...)
}

func (l *Logger) Infof(msg string, args ...interface{}) {
	l.Sugar().Infof(msg, args...)
}

func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.Sugar().Warnf(msg, args...)
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.Sugar().Errorf(msg, args...)
}

func (l *Logger) Fatalf(msg string, args ...interface{}) {
	l.Sugar().Fatalf(msg, args...)
}
