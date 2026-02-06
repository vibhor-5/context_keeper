package connectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConfigManager handles loading and managing connector configurations
type ConfigManager struct {
	configPath string
	configs    map[string]ConnectorConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
		configs:    make(map[string]ConnectorConfig),
	}
}

// LoadConfigurations loads connector configurations from file or environment
func (cm *ConfigManager) LoadConfigurations() error {
	// Try to load from file first
	if err := cm.loadFromFile(); err != nil {
		// If file doesn't exist, try to load from environment variables
		if os.IsNotExist(err) {
			return cm.loadFromEnvironment()
		}
		return fmt.Errorf("failed to load configurations from file: %w", err)
	}
	return nil
}

// loadFromFile loads configurations from a JSON file
func (cm *ConfigManager) loadFromFile() error {
	if cm.configPath == "" {
		return fmt.Errorf("config path not specified")
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return err
	}

	var fileConfig struct {
		Connectors map[string]ConnectorConfig `json:"connectors"`
	}

	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	cm.configs = fileConfig.Connectors
	return nil
}

// loadFromEnvironment loads configurations from environment variables
func (cm *ConfigManager) loadFromEnvironment() error {
	// Load GitHub configuration from environment
	if githubConfig := cm.loadGitHubFromEnv(); githubConfig != nil {
		cm.configs["github"] = *githubConfig
	}

	// Load Slack configuration from environment (placeholder for future implementation)
	if slackConfig := cm.loadSlackFromEnv(); slackConfig != nil {
		cm.configs["slack"] = *slackConfig
	}

	// Load Discord configuration from environment (placeholder for future implementation)
	if discordConfig := cm.loadDiscordFromEnv(); discordConfig != nil {
		cm.configs["discord"] = *discordConfig
	}

	return nil
}

// loadGitHubFromEnv loads GitHub configuration from environment variables
func (cm *ConfigManager) loadGitHubFromEnv() *ConnectorConfig {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	
	if clientID == "" || clientSecret == "" {
		return nil
	}

	enabled := strings.ToLower(os.Getenv("GITHUB_CONNECTOR_ENABLED")) != "false"
	
	config := &ConnectorConfig{
		Platform: "github",
		Enabled:  enabled,
		AuthConfig: AuthConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
			Scopes:       []string{"repo", "read:user"},
		},
		RateLimit: RateLimitConfig{
			RequestsPerHour:   5000, // GitHub API limit
			RequestsPerMinute: 100,
			BurstLimit:        10,
			BackoffMultiplier: 2.0,
			MaxRetries:        3,
		},
		SyncConfig: SyncConfig{
			BatchSize:       100,
			SyncInterval:    5 * time.Minute,
			MaxLookback:     30 * 24 * time.Hour, // 30 days
			IncrementalSync: true,
		},
		Metadata: map[string]interface{}{
			"api_url": "https://api.github.com",
		},
	}

	return config
}

// loadSlackFromEnv loads Slack configuration from environment variables
func (cm *ConfigManager) loadSlackFromEnv() *ConnectorConfig {
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	
	if botToken == "" {
		return nil
	}

	enabled := strings.ToLower(os.Getenv("SLACK_CONNECTOR_ENABLED")) != "false"
	
	config := &ConnectorConfig{
		Platform: "slack",
		Enabled:  enabled,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": botToken,
			},
		},
		RateLimit: RateLimitConfig{
			RequestsPerHour:   1000, // Slack API limit
			RequestsPerMinute: 50,
			BurstLimit:        5,
			BackoffMultiplier: 2.0,
			MaxRetries:        3,
		},
		SyncConfig: SyncConfig{
			BatchSize:       50,
			SyncInterval:    2 * time.Minute,
			MaxLookback:     7 * 24 * time.Hour, // 7 days
			IncrementalSync: true,
		},
		Metadata: map[string]interface{}{
			"channels":     strings.Split(os.Getenv("SLACK_CHANNELS"), ","),
			"include_dms":  strings.ToLower(os.Getenv("SLACK_INCLUDE_DMS")) == "true",
			"thread_depth": 10,
		},
	}

	return config
}

// loadDiscordFromEnv loads Discord configuration from environment variables
func (cm *ConfigManager) loadDiscordFromEnv() *ConnectorConfig {
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	
	if botToken == "" {
		return nil
	}

	enabled := strings.ToLower(os.Getenv("DISCORD_CONNECTOR_ENABLED")) != "false"
	
	config := &ConnectorConfig{
		Platform: "discord",
		Enabled:  enabled,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": botToken,
			},
		},
		RateLimit: RateLimitConfig{
			RequestsPerHour:   1000, // Discord API limit
			RequestsPerMinute: 50,
			BurstLimit:        5,
			BackoffMultiplier: 2.0,
			MaxRetries:        3,
		},
		SyncConfig: SyncConfig{
			BatchSize:       50,
			SyncInterval:    2 * time.Minute,
			MaxLookback:     7 * 24 * time.Hour, // 7 days
			IncrementalSync: true,
		},
		Metadata: map[string]interface{}{
			"guild_ids":    strings.Split(os.Getenv("DISCORD_GUILD_IDS"), ","),
			"channel_ids":  strings.Split(os.Getenv("DISCORD_CHANNEL_IDS"), ","),
			"thread_depth": 10,
		},
	}

	return config
}

// GetConfig returns the configuration for a specific platform
func (cm *ConfigManager) GetConfig(platform string) (ConnectorConfig, bool) {
	config, exists := cm.configs[platform]
	return config, exists
}

// SetConfig sets the configuration for a specific platform
func (cm *ConfigManager) SetConfig(platform string, config ConnectorConfig) {
	cm.configs[platform] = config
}

// GetAllConfigs returns all loaded configurations
func (cm *ConfigManager) GetAllConfigs() map[string]ConnectorConfig {
	result := make(map[string]ConnectorConfig)
	for platform, config := range cm.configs {
		result[platform] = config
	}
	return result
}

// GetEnabledPlatforms returns a list of enabled platform names
func (cm *ConfigManager) GetEnabledPlatforms() []string {
	var platforms []string
	for platform, config := range cm.configs {
		if config.Enabled {
			platforms = append(platforms, platform)
		}
	}
	return platforms
}

// SaveConfigurations saves current configurations to file
func (cm *ConfigManager) SaveConfigurations() error {
	if cm.configPath == "" {
		return fmt.Errorf("config path not specified")
	}

	// Ensure directory exists
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	fileConfig := struct {
		Connectors map[string]ConnectorConfig `json:"connectors"`
	}{
		Connectors: cm.configs,
	}

	data, err := json.MarshalIndent(fileConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates a connector configuration
func (cm *ConfigManager) ValidateConfig(config ConnectorConfig) error {
	if config.Platform == "" {
		return fmt.Errorf("platform name is required")
	}

	// Validate rate limit configuration
	if config.RateLimit.RequestsPerHour <= 0 {
		return fmt.Errorf("requests per hour must be positive")
	}

	if config.RateLimit.RequestsPerMinute <= 0 {
		return fmt.Errorf("requests per minute must be positive")
	}

	if config.RateLimit.BurstLimit <= 0 {
		return fmt.Errorf("burst limit must be positive")
	}

	if config.RateLimit.BackoffMultiplier <= 1.0 {
		return fmt.Errorf("backoff multiplier must be greater than 1.0")
	}

	// Validate sync configuration
	if config.SyncConfig.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	if config.SyncConfig.SyncInterval <= 0 {
		return fmt.Errorf("sync interval must be positive")
	}

	// Platform-specific validation
	switch config.Platform {
	case "github":
		return cm.validateGitHubConfig(config)
	case "slack":
		return cm.validateSlackConfig(config)
	case "discord":
		return cm.validateDiscordConfig(config)
	}

	return nil
}

// validateGitHubConfig validates GitHub-specific configuration
func (cm *ConfigManager) validateGitHubConfig(config ConnectorConfig) error {
	if config.AuthConfig.ClientID == "" {
		return fmt.Errorf("GitHub client ID is required")
	}

	if config.AuthConfig.ClientSecret == "" {
		return fmt.Errorf("GitHub client secret is required")
	}

	// Validate required scopes
	requiredScopes := map[string]bool{"repo": false, "read:user": false}
	for _, scope := range config.AuthConfig.Scopes {
		if _, exists := requiredScopes[scope]; exists {
			requiredScopes[scope] = true
		}
	}

	for scope, found := range requiredScopes {
		if !found {
			return fmt.Errorf("GitHub connector requires scope: %s", scope)
		}
	}

	return nil
}

// validateSlackConfig validates Slack-specific configuration
func (cm *ConfigManager) validateSlackConfig(config ConnectorConfig) error {
	botToken, ok := config.AuthConfig.Metadata["bot_token"]
	if !ok || botToken == "" {
		return fmt.Errorf("Slack bot token is required")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		return fmt.Errorf("Slack bot token must start with 'xoxb-'")
	}

	return nil
}

// validateDiscordConfig validates Discord-specific configuration
func (cm *ConfigManager) validateDiscordConfig(config ConnectorConfig) error {
	botToken, ok := config.AuthConfig.Metadata["bot_token"]
	if !ok || botToken == "" {
		return fmt.Errorf("Discord bot token is required")
	}

	return nil
}

// SecurityBoundary enforces security boundaries between connectors
type SecurityBoundary struct {
	allowedPlatforms map[string]bool
	isolationMode    string
}

// NewSecurityBoundary creates a new security boundary manager
func NewSecurityBoundary(allowedPlatforms []string, isolationMode string) *SecurityBoundary {
	allowed := make(map[string]bool)
	for _, platform := range allowedPlatforms {
		allowed[platform] = true
	}

	return &SecurityBoundary{
		allowedPlatforms: allowed,
		isolationMode:    isolationMode,
	}
}

// ValidateAccess validates that a platform is allowed to access resources
func (sb *SecurityBoundary) ValidateAccess(platform string, resource string) error {
	if !sb.allowedPlatforms[platform] {
		return fmt.Errorf("platform %s is not allowed", platform)
	}

	// In strict isolation mode, platforms can only access their own resources
	if sb.isolationMode == "strict" {
		if !strings.HasPrefix(resource, platform+":") {
			return fmt.Errorf("platform %s cannot access resource %s in strict isolation mode", platform, resource)
		}
	}

	return nil
}

// IsPlatformAllowed checks if a platform is allowed
func (sb *SecurityBoundary) IsPlatformAllowed(platform string) bool {
	return sb.allowedPlatforms[platform]
}