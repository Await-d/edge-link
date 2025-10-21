package integrations

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"go.uber.org/zap"
)

// Manager 集成管理器，负责协调多个告警平台
type Manager struct {
	integrations map[string]Integration // 注册的集成
	configs      map[string]IntegrationConfig
	logger       *zap.Logger
	metrics      map[string]*IntegrationMetrics
	mu           sync.RWMutex
}

// NewManager 创建集成管理器
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		integrations: make(map[string]Integration),
		configs:      make(map[string]IntegrationConfig),
		logger:       logger,
		metrics:      make(map[string]*IntegrationMetrics),
	}
}

// Register 注册集成
func (m *Manager) Register(integration Integration, config IntegrationConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := integration.Name()

	// 验证配置
	if err := integration.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid config for %s: %w", name, err)
	}

	m.integrations[name] = integration
	m.configs[name] = config
	m.metrics[name] = &IntegrationMetrics{}

	m.logger.Info("Registered integration",
		zap.String("integration", name),
		zap.Bool("enabled", config.Enabled()),
		zap.Int("priority", config.Priority()),
	)

	return nil
}

// Unregister 注销集成
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.integrations, name)
	delete(m.configs, name)
	delete(m.metrics, name)

	m.logger.Info("Unregistered integration", zap.String("integration", name))
}

// SendAlert 发送告警到所有启用的集成
func (m *Manager) SendAlert(ctx context.Context, alert *domain.Alert) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取启用的集成并按优先级排序
	enabled := m.getEnabledIntegrationsSorted()
	if len(enabled) == 0 {
		return fmt.Errorf("no enabled integrations")
	}

	// 并发发送到所有集成
	var wg sync.WaitGroup
	errChan := make(chan error, len(enabled))

	for _, item := range enabled {
		wg.Add(1)
		go func(integration Integration, config IntegrationConfig) {
			defer wg.Done()

			// 发送告警（带重试）
			err := m.sendWithRetry(ctx, integration, alert, config.RetryConfig())
			if err != nil {
				m.logger.Error("Failed to send alert",
					zap.String("integration", integration.Name()),
					zap.String("alert_id", alert.ID.String()),
					zap.Error(err),
				)
				errChan <- err
			} else {
				m.logger.Info("Alert sent successfully",
					zap.String("integration", integration.Name()),
					zap.String("alert_id", alert.ID.String()),
				)
			}
		}(item.integration, item.config)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// 如果所有集成都失败，返回错误
	if len(errors) == len(enabled) {
		return fmt.Errorf("all integrations failed: %d errors", len(errors))
	}

	// 部分成功也视为成功（至少一个平台收到告警）
	return nil
}

// ResolveAlert 解决告警
func (m *Manager) ResolveAlert(ctx context.Context, alertID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	enabled := m.getEnabledIntegrationsSorted()
	if len(enabled) == 0 {
		return fmt.Errorf("no enabled integrations")
	}

	var wg sync.WaitGroup
	for _, item := range enabled {
		wg.Add(1)
		go func(integration Integration) {
			defer wg.Done()
			if err := integration.ResolveAlert(ctx, alertID); err != nil {
				m.logger.Warn("Failed to resolve alert",
					zap.String("integration", integration.Name()),
					zap.String("alert_id", alertID),
					zap.Error(err),
				)
			}
		}(item.integration)
	}

	wg.Wait()
	return nil
}

// UpdateAlert 更新告警状态
func (m *Manager) UpdateAlert(ctx context.Context, alertID string, status domain.AlertStatus) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	enabled := m.getEnabledIntegrationsSorted()
	if len(enabled) == 0 {
		return fmt.Errorf("no enabled integrations")
	}

	var wg sync.WaitGroup
	for _, item := range enabled {
		wg.Add(1)
		go func(integration Integration) {
			defer wg.Done()
			if err := integration.UpdateAlert(ctx, alertID, status); err != nil {
				m.logger.Warn("Failed to update alert",
					zap.String("integration", integration.Name()),
					zap.String("alert_id", alertID),
					zap.String("status", string(status)),
					zap.Error(err),
				)
			}
		}(item.integration)
	}

	wg.Wait()
	return nil
}

// HealthCheck 检查所有集成的健康状态
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)
	for name, integration := range m.integrations {
		if m.configs[name].Enabled() {
			results[name] = integration.HealthCheck(ctx)
		}
	}

	return results
}

// GetMetrics 获取所有集成的指标
func (m *Manager) GetMetrics() map[string]*IntegrationMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本
	metrics := make(map[string]*IntegrationMetrics, len(m.metrics))
	for name, metric := range m.metrics {
		metricCopy := *metric
		metrics[name] = &metricCopy
	}

	return metrics
}

// sendWithRetry 带重试的发送
func (m *Manager) sendWithRetry(ctx context.Context, integration Integration, alert *domain.Alert, retryConfig RetryConfig) error {
	var lastErr error
	delay := retryConfig.InitialDelay

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// 等待后重试
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			m.logger.Info("Retrying alert send",
				zap.String("integration", integration.Name()),
				zap.Int("attempt", attempt),
				zap.String("alert_id", alert.ID.String()),
			)
		}

		// 记录开始时间
		startTime := time.Now()

		// 发送告警
		err := integration.SendAlert(ctx, alert)

		// 更新指标
		m.updateMetrics(integration.Name(), err, time.Since(startTime))

		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否可重试
		if integrationErr, ok := err.(*IntegrationError); ok && !integrationErr.Retryable {
			return integrationErr
		}

		// 计算下次延迟（指数退避）
		delay = time.Duration(float64(delay) * retryConfig.BackoffFactor)
		if delay > retryConfig.MaxDelay {
			delay = retryConfig.MaxDelay
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", retryConfig.MaxRetries+1, lastErr)
}

// updateMetrics 更新集成指标
func (m *Manager) updateMetrics(name string, err error, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric := m.metrics[name]
	metric.TotalSent++
	metric.LastSentTime = time.Now()

	if err == nil {
		metric.SuccessCount++
	} else {
		metric.FailureCount++
	}

	// 更新平均响应时间（简单移动平均）
	if metric.AvgResponseTime == 0 {
		metric.AvgResponseTime = duration
	} else {
		metric.AvgResponseTime = (metric.AvgResponseTime + duration) / 2
	}
}

// integrationItem 集成项（用于排序）
type integrationItem struct {
	integration Integration
	config      IntegrationConfig
	priority    int
}

// getEnabledIntegrationsSorted 获取启用的集成并按优先级排序
func (m *Manager) getEnabledIntegrationsSorted() []integrationItem {
	var items []integrationItem

	for name, integration := range m.integrations {
		config := m.configs[name]
		if config.Enabled() {
			items = append(items, integrationItem{
				integration: integration,
				config:      config,
				priority:    config.Priority(),
			})
		}
	}

	// 按优先级排序（冒泡排序，因为数量较少）
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].priority > items[j].priority {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	return items
}
