package integrations

import (
	"context"
	"fmt"

	"github.com/edgelink/backend/cmd/alert-service/internal/config"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/discord"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/opsgenie"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/pagerduty"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/slack"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/teams"
	"go.uber.org/zap"
)

// Factory 集成工厂
type Factory struct {
	logger *zap.Logger
}

// NewFactory 创建集成工厂
func NewFactory(logger *zap.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateManager 根据配置创建并初始化集成管理器
func (f *Factory) CreateManager(cfg *config.IntegrationsConfig) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid integrations config: %w", err)
	}

	manager := NewManager(f.logger)

	// 注册PagerDuty
	if cfg.PagerDuty != nil && cfg.PagerDuty.Enabled {
		pdConfig := cfg.PagerDuty.ToPagerDutyConfig()
		pdIntegration := pagerduty.NewIntegration(pdConfig, f.logger.Named("pagerduty"))
		if err := manager.Register(pdIntegration, pdConfig); err != nil {
			return nil, fmt.Errorf("failed to register pagerduty: %w", err)
		}
	}

	// 注册Opsgenie
	if cfg.Opsgenie != nil && cfg.Opsgenie.Enabled {
		ogConfig := cfg.Opsgenie.ToOpsgenieConfig()
		ogIntegration := opsgenie.NewIntegration(ogConfig, f.logger.Named("opsgenie"))
		if err := manager.Register(ogIntegration, ogConfig); err != nil {
			return nil, fmt.Errorf("failed to register opsgenie: %w", err)
		}
	}

	// 注册Slack
	if cfg.Slack != nil && cfg.Slack.Enabled {
		slackConfig := cfg.Slack.ToSlackConfig()
		slackIntegration := slack.NewIntegration(slackConfig, f.logger.Named("slack"))
		if err := manager.Register(slackIntegration, slackConfig); err != nil {
			return nil, fmt.Errorf("failed to register slack: %w", err)
		}
	}

	// 注册Discord
	if cfg.Discord != nil && cfg.Discord.Enabled {
		discordConfig := cfg.Discord.ToDiscordConfig()
		discordIntegration := discord.NewIntegration(discordConfig, f.logger.Named("discord"))
		if err := manager.Register(discordIntegration, discordConfig); err != nil {
			return nil, fmt.Errorf("failed to register discord: %w", err)
		}
	}

	// 注册Teams
	if cfg.Teams != nil && cfg.Teams.Enabled {
		teamsConfig := cfg.Teams.ToTeamsConfig()
		teamsIntegration := teams.NewIntegration(teamsConfig, f.logger.Named("teams"))
		if err := manager.Register(teamsIntegration, teamsConfig); err != nil {
			return nil, fmt.Errorf("failed to register teams: %w", err)
		}
	}

	f.logger.Info("Integrations initialized",
		zap.Int("total_enabled", cfg.GetEnabledCount()),
	)

	return manager, nil
}

// HealthCheckAll 执行所有集成的健康检查
func (f *Factory) HealthCheckAll(ctx context.Context, manager *Manager) map[string]error {
	results := manager.HealthCheck(ctx)

	// 记录健康检查结果
	for name, err := range results {
		if err != nil {
			f.logger.Error("Integration health check failed",
				zap.String("integration", name),
				zap.Error(err),
			)
		} else {
			f.logger.Info("Integration health check passed",
				zap.String("integration", name),
			)
		}
	}

	return results
}

// CreateIntegration 创建单个集成（用于动态添加）
func (f *Factory) CreateIntegration(integrationType string, configData interface{}) (Integration, error) {
	switch integrationType {
	case "pagerduty":
		cfg, ok := configData.(*pagerduty.Config)
		if !ok {
			return nil, fmt.Errorf("invalid pagerduty config type")
		}
		return pagerduty.NewIntegration(cfg, f.logger.Named("pagerduty")), nil

	case "opsgenie":
		cfg, ok := configData.(*opsgenie.Config)
		if !ok {
			return nil, fmt.Errorf("invalid opsgenie config type")
		}
		return opsgenie.NewIntegration(cfg, f.logger.Named("opsgenie")), nil

	case "slack":
		cfg, ok := configData.(*slack.Config)
		if !ok {
			return nil, fmt.Errorf("invalid slack config type")
		}
		return slack.NewIntegration(cfg, f.logger.Named("slack")), nil

	case "discord":
		cfg, ok := configData.(*discord.Config)
		if !ok {
			return nil, fmt.Errorf("invalid discord config type")
		}
		return discord.NewIntegration(cfg, f.logger.Named("discord")), nil

	case "teams":
		cfg, ok := configData.(*teams.Config)
		if !ok {
			return nil, fmt.Errorf("invalid teams config type")
		}
		return teams.NewIntegration(cfg, f.logger.Named("teams")), nil

	default:
		return nil, fmt.Errorf("unsupported integration type: %s", integrationType)
	}
}
