package config

import (
	"fmt"

	"github.com/edgelink/backend/cmd/alert-service/internal/integrations"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/discord"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/opsgenie"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/pagerduty"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/slack"
	"github.com/edgelink/backend/cmd/alert-service/internal/integrations/teams"
)

// IntegrationsConfig 集成配置
type IntegrationsConfig struct {
	PagerDuty *PagerDutyConfig `yaml:"pagerduty" json:"pagerduty"`
	Opsgenie  *OpsgenieConfig  `yaml:"opsgenie" json:"opsgenie"`
	Slack     *SlackConfig     `yaml:"slack" json:"slack"`
	Discord   *DiscordConfig   `yaml:"discord" json:"discord"`
	Teams     *TeamsConfig     `yaml:"teams" json:"teams"`
}

// PagerDutyConfig PagerDuty配置
type PagerDutyConfig struct {
	Enabled        bool                                `yaml:"enabled" json:"enabled"`
	Priority       int                                 `yaml:"priority" json:"priority"`
	IntegrationKey string                              `yaml:"integration_key" json:"integration_key" env:"PAGERDUTY_INTEGRATION_KEY"`
	DefaultService string                              `yaml:"default_service" json:"default_service"`
	SeverityMap    *integrations.AlertSeverityMapping  `yaml:"severity_map" json:"severity_map"`
	RetryConfig    *integrations.RetryConfig           `yaml:"retry_config" json:"retry_config"`
}

// ToPagerDutyConfig 转换为PagerDuty集成配置
func (c *PagerDutyConfig) ToPagerDutyConfig() *pagerduty.Config {
	config := &pagerduty.Config{
		IntegrationKey: c.IntegrationKey,
		Enabled:        c.Enabled,
		Priority:       c.Priority,
		DefaultService: c.DefaultService,
	}

	if c.SeverityMap != nil {
		config.SeverityMap = *c.SeverityMap
	}

	if c.RetryConfig != nil {
		config.RetryConfig = *c.RetryConfig
	} else {
		config.RetryConfig = integrations.DefaultRetryConfig()
	}

	return config
}

// OpsgenieConfig Opsgenie配置
type OpsgenieConfig struct {
	Enabled      bool                          `yaml:"enabled" json:"enabled"`
	Priority     int                           `yaml:"priority" json:"priority"`
	APIKey       string                        `yaml:"api_key" json:"api_key" env:"OPSGENIE_API_KEY"`
	DefaultTeams []string                      `yaml:"default_teams" json:"default_teams"`
	DefaultTags  []string                      `yaml:"default_tags" json:"default_tags"`
	PriorityMap  *opsgenie.PriorityMapping     `yaml:"priority_map" json:"priority_map"`
	RetryConfig  *integrations.RetryConfig     `yaml:"retry_config" json:"retry_config"`
}

// ToOpsgenieConfig 转换为Opsgenie集成配置
func (c *OpsgenieConfig) ToOpsgenieConfig() *opsgenie.Config {
	config := &opsgenie.Config{
		APIKey:       c.APIKey,
		Enabled:      c.Enabled,
		Priority:     c.Priority,
		DefaultTeams: c.DefaultTeams,
		DefaultTags:  c.DefaultTags,
	}

	if c.PriorityMap != nil {
		config.PriorityMap = *c.PriorityMap
	}

	if c.RetryConfig != nil {
		config.RetryConfig = *c.RetryConfig
	} else {
		config.RetryConfig = integrations.DefaultRetryConfig()
	}

	return config
}

// SlackConfig Slack配置
type SlackConfig struct {
	Enabled     bool                      `yaml:"enabled" json:"enabled"`
	Priority    int                       `yaml:"priority" json:"priority"`
	WebhookURL  string                    `yaml:"webhook_url" json:"webhook_url" env:"SLACK_WEBHOOK_URL"`
	Channel     string                    `yaml:"channel" json:"channel"`
	Username    string                    `yaml:"username" json:"username"`
	IconEmoji   string                    `yaml:"icon_emoji" json:"icon_emoji"`
	ColorMap    *slack.ColorMapping       `yaml:"color_map" json:"color_map"`
	RetryConfig *integrations.RetryConfig `yaml:"retry_config" json:"retry_config"`
}

// ToSlackConfig 转换为Slack集成配置
func (c *SlackConfig) ToSlackConfig() *slack.Config {
	config := &slack.Config{
		WebhookURL: c.WebhookURL,
		Enabled:    c.Enabled,
		Priority:   c.Priority,
		Channel:    c.Channel,
		Username:   c.Username,
		IconEmoji:  c.IconEmoji,
	}

	if c.ColorMap != nil {
		config.ColorMap = *c.ColorMap
	}

	if c.RetryConfig != nil {
		config.RetryConfig = *c.RetryConfig
	} else {
		config.RetryConfig = integrations.DefaultRetryConfig()
	}

	return config
}

// DiscordConfig Discord配置
type DiscordConfig struct {
	Enabled     bool                      `yaml:"enabled" json:"enabled"`
	Priority    int                       `yaml:"priority" json:"priority"`
	WebhookURL  string                    `yaml:"webhook_url" json:"webhook_url" env:"DISCORD_WEBHOOK_URL"`
	Username    string                    `yaml:"username" json:"username"`
	AvatarURL   string                    `yaml:"avatar_url" json:"avatar_url"`
	ColorMap    *discord.ColorMapping     `yaml:"color_map" json:"color_map"`
	RetryConfig *integrations.RetryConfig `yaml:"retry_config" json:"retry_config"`
}

// ToDiscordConfig 转换为Discord集成配置
func (c *DiscordConfig) ToDiscordConfig() *discord.Config {
	config := &discord.Config{
		WebhookURL: c.WebhookURL,
		Enabled:    c.Enabled,
		Priority:   c.Priority,
		Username:   c.Username,
		AvatarURL:  c.AvatarURL,
	}

	if c.ColorMap != nil {
		config.ColorMap = *c.ColorMap
	}

	if c.RetryConfig != nil {
		config.RetryConfig = *c.RetryConfig
	} else {
		config.RetryConfig = integrations.DefaultRetryConfig()
	}

	return config
}

// TeamsConfig Microsoft Teams配置
type TeamsConfig struct {
	Enabled     bool                      `yaml:"enabled" json:"enabled"`
	Priority    int                       `yaml:"priority" json:"priority"`
	WebhookURL  string                    `yaml:"webhook_url" json:"webhook_url" env:"TEAMS_WEBHOOK_URL"`
	ColorMap    *teams.ColorMapping       `yaml:"color_map" json:"color_map"`
	RetryConfig *integrations.RetryConfig `yaml:"retry_config" json:"retry_config"`
}

// ToTeamsConfig 转换为Teams集成配置
func (c *TeamsConfig) ToTeamsConfig() *teams.Config {
	config := &teams.Config{
		WebhookURL: c.WebhookURL,
		Enabled:    c.Enabled,
		Priority:   c.Priority,
	}

	if c.ColorMap != nil {
		config.ColorMap = *c.ColorMap
	}

	if c.RetryConfig != nil {
		config.RetryConfig = *c.RetryConfig
	} else {
		config.RetryConfig = integrations.DefaultRetryConfig()
	}

	return config
}

// Validate 验证配置
func (c *IntegrationsConfig) Validate() error {
	// 检查是否至少启用一个集成
	hasEnabled := false

	if c.PagerDuty != nil && c.PagerDuty.Enabled {
		if c.PagerDuty.IntegrationKey == "" {
			return fmt.Errorf("pagerduty integration_key is required when enabled")
		}
		hasEnabled = true
	}

	if c.Opsgenie != nil && c.Opsgenie.Enabled {
		if c.Opsgenie.APIKey == "" {
			return fmt.Errorf("opsgenie api_key is required when enabled")
		}
		hasEnabled = true
	}

	if c.Slack != nil && c.Slack.Enabled {
		if c.Slack.WebhookURL == "" {
			return fmt.Errorf("slack webhook_url is required when enabled")
		}
		hasEnabled = true
	}

	if c.Discord != nil && c.Discord.Enabled {
		if c.Discord.WebhookURL == "" {
			return fmt.Errorf("discord webhook_url is required when enabled")
		}
		hasEnabled = true
	}

	if c.Teams != nil && c.Teams.Enabled {
		if c.Teams.WebhookURL == "" {
			return fmt.Errorf("teams webhook_url is required when enabled")
		}
		hasEnabled = true
	}

	if !hasEnabled {
		return fmt.Errorf("at least one integration must be enabled")
	}

	return nil
}

// GetEnabledCount 获取启用的集成数量
func (c *IntegrationsConfig) GetEnabledCount() int {
	count := 0

	if c.PagerDuty != nil && c.PagerDuty.Enabled {
		count++
	}
	if c.Opsgenie != nil && c.Opsgenie.Enabled {
		count++
	}
	if c.Slack != nil && c.Slack.Enabled {
		count++
	}
	if c.Discord != nil && c.Discord.Enabled {
		count++
	}
	if c.Teams != nil && c.Teams.Enabled {
		count++
	}

	return count
}
