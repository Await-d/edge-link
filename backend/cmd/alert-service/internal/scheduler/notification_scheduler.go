package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/edgelink/backend/cmd/alert-service/internal/notifier"
	"github.com/edgelink/backend/cmd/alert-service/internal/rules"
	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"go.uber.org/zap"
)

// NotificationTask 通知任务
type NotificationTask struct {
	Alert      *domain.Alert
	Priority   int       // 优先级 (1-5, 1最高)
	ScheduledAt time.Time
	RetryCount int
}

// NotificationScheduler 通知调度器
type NotificationScheduler struct {
	emailNotifier   *notifier.EmailNotifier
	webhookNotifier *notifier.WebhookNotifier
	ruleEngine      *rules.Engine
	deviceRepo      repository.DeviceRepository
	logger          *zap.Logger

	queue      []*NotificationTask
	queueMutex sync.Mutex
	taskChan   chan *NotificationTask

	// 速率限制
	rateLimiter *RateLimiter

	// 规则引擎配置
	rulesFilePath string
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	RulesFile          string
	EnableRuleEngine   bool
	EnableHotReload    bool
	ReloadInterval     time.Duration
}

// NewNotificationScheduler 创建通知调度器
func NewNotificationScheduler(
	emailNotifier *notifier.EmailNotifier,
	webhookNotifier *notifier.WebhookNotifier,
	deviceRepo repository.DeviceRepository,
	logger *zap.Logger,
	config SchedulerConfig,
) *NotificationScheduler {
	scheduler := &NotificationScheduler{
		emailNotifier:   emailNotifier,
		webhookNotifier: webhookNotifier,
		deviceRepo:      deviceRepo,
		logger:          logger,
		queue:           make([]*NotificationTask, 0),
		taskChan:        make(chan *NotificationTask, 1000),
		rateLimiter:     NewRateLimiter(100, time.Minute), // 每分钟最多100个通知
		rulesFilePath:   config.RulesFile,
	}

	// 初始化规则引擎
	if config.EnableRuleEngine && config.RulesFile != "" {
		scheduler.ruleEngine = rules.NewEngine(
			emailNotifier,
			webhookNotifier,
			deviceRepo,
			logger,
		)

		// 加载规则文件
		if err := scheduler.ruleEngine.LoadRules(config.RulesFile); err != nil {
			logger.Error("Failed to load rules, rule engine disabled",
				zap.Error(err),
				zap.String("rules_file", config.RulesFile),
			)
			scheduler.ruleEngine = nil
		} else {
			logger.Info("Rule engine initialized successfully")
		}
	}

	return scheduler
}

// Schedule 调度告警通知
func (ns *NotificationScheduler) Schedule(ctx context.Context, alert *domain.Alert) error {
	// 如果启用了规则引擎，使用规则引擎处理
	if ns.ruleEngine != nil {
		return ns.scheduleWithRules(ctx, alert)
	}

	// 否则使用传统调度方式
	return ns.scheduleTraditional(ctx, alert)
}

// scheduleWithRules 使用规则引擎调度
func (ns *NotificationScheduler) scheduleWithRules(ctx context.Context, alert *domain.Alert) error {
	ns.logger.Debug("Processing alert with rule engine",
		zap.String("alert_id", alert.ID.String()),
	)

	// 规则引擎会自动处理匹配、执行和速率限制
	if err := ns.ruleEngine.Process(ctx, alert); err != nil {
		ns.logger.Error("Rule engine processing failed",
			zap.String("alert_id", alert.ID.String()),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// scheduleTraditional 传统调度方式（向后兼容）
func (ns *NotificationScheduler) scheduleTraditional(ctx context.Context, alert *domain.Alert) error {
	priority := ns.calculatePriority(alert)

	task := &NotificationTask{
		Alert:       alert,
		Priority:    priority,
		ScheduledAt: time.Now(),
		RetryCount:  0,
	}

	// 添加到任务队列
	select {
	case ns.taskChan <- task:
		ns.logger.Debug("Notification task scheduled",
			zap.String("alert_id", alert.ID.String()),
			zap.Int("priority", priority),
		)
		return nil
	default:
		ns.logger.Warn("Notification task channel full, dropping task",
			zap.String("alert_id", alert.ID.String()),
		)
		return nil
	}
}

// Start 启动调度器
func (ns *NotificationScheduler) Start(ctx context.Context) {
	ns.logger.Info("Notification scheduler started")

	// 如果启用了规则引擎热重载，启动自动重载
	if ns.ruleEngine != nil && ns.rulesFilePath != "" {
		engineConfig := rules.EngineConfig{
			RulesFile:       ns.rulesFilePath,
			EnableHotReload: true,
			ReloadInterval:  5 * time.Minute,
		}
		go ns.ruleEngine.StartAutoReload(ctx, engineConfig)
	}

	// 启动worker池（用于传统调度方式）
	workerCount := 5
	for i := 0; i < workerCount; i++ {
		go ns.worker(ctx, i)
	}

	<-ctx.Done()
	ns.logger.Info("Notification scheduler shutting down")
}

// worker 处理通知任务
func (ns *NotificationScheduler) worker(ctx context.Context, workerID int) {
	ns.logger.Info("Notification worker started", zap.Int("worker_id", workerID))

	for {
		select {
		case <-ctx.Done():
			ns.logger.Info("Notification worker shutting down", zap.Int("worker_id", workerID))
			return

		case task := <-ns.taskChan:
			// 检查速率限制
			if !ns.rateLimiter.Allow() {
				ns.logger.Warn("Rate limit exceeded, requeuing task",
					zap.String("alert_id", task.Alert.ID.String()),
				)
				// 延迟后重新调度
				time.Sleep(1 * time.Second)
				ns.taskChan <- task
				continue
			}

			// 处理任务
			ns.processTask(ctx, task)
		}
	}
}

// processTask 处理单个通知任务（传统方式）
func (ns *NotificationScheduler) processTask(ctx context.Context, task *NotificationTask) {
	alert := task.Alert

	ns.logger.Info("Processing notification task",
		zap.String("alert_id", alert.ID.String()),
		zap.String("severity", string(alert.Severity)),
	)

	// 发送邮件通知 (Critical和High级别)
	if alert.Severity == domain.SeverityCritical || alert.Severity == domain.SeverityHigh {
		recipients := []string{} // TODO: 从配置读取收件人列表
		if len(recipients) > 0 {
			if err := ns.emailNotifier.SendAlert(ctx, alert, recipients); err != nil {
				ns.logger.Error("Failed to send email notification",
					zap.Error(err),
					zap.String("alert_id", alert.ID.String()),
				)
			}
		}
	}

	// 发送Webhook通知
	webhookURL := "" // TODO: 从配置读取Webhook URL
	if webhookURL != "" {
		if err := ns.webhookNotifier.SendAlert(ctx, alert, webhookURL); err != nil {
			ns.logger.Error("Failed to send webhook notification",
				zap.Error(err),
				zap.String("alert_id", alert.ID.String()),
			)

			// 重试逻辑
			if task.RetryCount < 3 {
				task.RetryCount++
				time.Sleep(time.Duration(task.RetryCount) * 5 * time.Second)
				ns.taskChan <- task
				return
			}
		}
	}
}

// calculatePriority 计算任务优先级
func (ns *NotificationScheduler) calculatePriority(alert *domain.Alert) int {
	switch alert.Severity {
	case domain.SeverityCritical:
		return 1
	case domain.SeverityHigh:
		return 2
	case domain.SeverityMedium:
		return 3
	case domain.SeverityLow:
		return 4
	default:
		return 5
	}
}

// GetRuleEngine 获取规则引擎实例
func (ns *NotificationScheduler) GetRuleEngine() *rules.Engine {
	return ns.ruleEngine
}

// ReloadRules 重新加载规则
func (ns *NotificationScheduler) ReloadRules() error {
	if ns.ruleEngine == nil {
		return nil
	}
	return ns.ruleEngine.ReloadRules(ns.rulesFilePath)
}

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	maxRequests int
	interval    time.Duration
	tokens      int
	lastRefill  time.Time
	mutex       sync.Mutex
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(maxRequests int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		interval:    interval,
		tokens:      maxRequests,
		lastRefill:  time.Now(),
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// 补充令牌
	if elapsed >= rl.interval {
		rl.tokens = rl.maxRequests
		rl.lastRefill = now
	}

	// 检查并消耗令牌
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}
