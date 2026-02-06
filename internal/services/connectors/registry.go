package connectors

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConnectorFactory creates platform connector instances
type ConnectorFactory func(config ConnectorConfig) (PlatformConnector, error)

// ConnectorConfig contains configuration for a platform connector
type ConnectorConfig struct {
	Platform    string                 `json:"platform"`
	Enabled     bool                   `json:"enabled"`
	AuthConfig  AuthConfig            `json:"auth_config"`
	RateLimit   RateLimitConfig       `json:"rate_limit"`
	SyncConfig  SyncConfig            `json:"sync_config"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	RequestsPerHour   int `json:"requests_per_hour"`
	RequestsPerMinute int `json:"requests_per_minute"`
	BurstLimit        int `json:"burst_limit"`
	BackoffMultiplier float64 `json:"backoff_multiplier"`
	MaxRetries        int `json:"max_retries"`
}

// SyncConfig contains synchronization configuration
type SyncConfig struct {
	BatchSize       int           `json:"batch_size"`
	SyncInterval    time.Duration `json:"sync_interval"`
	MaxLookback     time.Duration `json:"max_lookback"`
	IncrementalSync bool          `json:"incremental_sync"`
}

// Registry manages platform connector registration and creation
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ConnectorFactory
	configs   map[string]ConnectorConfig
}

// NewRegistry creates a new connector registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ConnectorFactory),
		configs:   make(map[string]ConnectorConfig),
	}
}

// Register registers a connector factory for a platform
func (r *Registry) Register(platform string, factory ConnectorFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.factories[platform]; exists {
		return fmt.Errorf("connector for platform %s already registered", platform)
	}
	
	r.factories[platform] = factory
	return nil
}

// RegisterDefaultConnectors registers all built-in connectors
func (r *Registry) RegisterDefaultConnectors() error {
	// Register GitHub connector
	if err := r.Register("github", NewGitHubConnector); err != nil {
		return fmt.Errorf("failed to register GitHub connector: %w", err)
	}

	// Register Slack connector
	if err := r.Register("slack", NewSlackConnector); err != nil {
		return fmt.Errorf("failed to register Slack connector: %w", err)
	}

	// Register Discord connector
	if err := r.Register("discord", NewDiscordConnector); err != nil {
		return fmt.Errorf("failed to register Discord connector: %w", err)
	}

	return nil
}

// CreateConnector creates a connector instance for the specified platform
func (r *Registry) CreateConnector(platform string) (PlatformConnector, error) {
	r.mu.RLock()
	factory, exists := r.factories[platform]
	config, hasConfig := r.configs[platform]
	r.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("no connector registered for platform: %s", platform)
	}
	
	if !hasConfig {
		return nil, fmt.Errorf("no configuration found for platform: %s", platform)
	}
	
	if !config.Enabled {
		return nil, fmt.Errorf("connector for platform %s is disabled", platform)
	}
	
	return factory(config)
}

// SetConfig sets the configuration for a platform connector
func (r *Registry) SetConfig(platform string, config ConnectorConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[platform] = config
}

// GetConfig gets the configuration for a platform connector
func (r *Registry) GetConfig(platform string) (ConnectorConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	config, exists := r.configs[platform]
	return config, exists
}

// ListPlatforms returns all registered platform names
func (r *Registry) ListPlatforms() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	platforms := make([]string, 0, len(r.factories))
	for platform := range r.factories {
		platforms = append(platforms, platform)
	}
	return platforms
}

// ListEnabledPlatforms returns all enabled platform names
func (r *Registry) ListEnabledPlatforms() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var platforms []string
	for platform, config := range r.configs {
		if config.Enabled {
			platforms = append(platforms, platform)
		}
	}
	return platforms
}

// ConnectorManager manages multiple platform connectors
type ConnectorManager struct {
	registry   *Registry
	connectors map[string]PlatformConnector
	mu         sync.RWMutex
}

// NewConnectorManager creates a new connector manager
func NewConnectorManager(registry *Registry) *ConnectorManager {
	return &ConnectorManager{
		registry:   registry,
		connectors: make(map[string]PlatformConnector),
	}
}

// InitializeConnectors initializes all enabled connectors
func (cm *ConnectorManager) InitializeConnectors(ctx context.Context) error {
	platforms := cm.registry.ListEnabledPlatforms()
	
	for _, platform := range platforms {
		connector, err := cm.registry.CreateConnector(platform)
		if err != nil {
			return fmt.Errorf("failed to create connector for %s: %w", platform, err)
		}
		
		cm.mu.Lock()
		cm.connectors[platform] = connector
		cm.mu.Unlock()
	}
	
	return nil
}

// GetConnector returns a connector for the specified platform
func (cm *ConnectorManager) GetConnector(platform string) (PlatformConnector, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	connector, exists := cm.connectors[platform]
	if !exists {
		return nil, fmt.Errorf("no connector available for platform: %s", platform)
	}
	
	return connector, nil
}

// GetAllConnectors returns all active connectors
func (cm *ConnectorManager) GetAllConnectors() map[string]PlatformConnector {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	result := make(map[string]PlatformConnector)
	for platform, connector := range cm.connectors {
		result[platform] = connector
	}
	return result
}

// Shutdown gracefully shuts down all connectors
func (cm *ConnectorManager) Shutdown(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Clear connectors map
	cm.connectors = make(map[string]PlatformConnector)
	
	return nil
}