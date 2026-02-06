package services

import (
	"context"
	"testing"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/config"
	"github.com/DevAnuragT/context_keeper/internal/models"
)

// MockDiscordStore implements RepositoryStore for testing
type MockDiscordStore struct {
	integrations map[string]*models.ProjectIntegration
	dataSources  map[string]*models.ProjectDataSource
}

func NewMockDiscordStore() *MockDiscordStore {
	return &MockDiscordStore{
		integrations: make(map[string]*models.ProjectIntegration),
		dataSources:  make(map[string]*models.ProjectDataSource),
	}
}

func (m *MockDiscordStore) GetProjectIntegrationByPlatform(ctx context.Context, projectID, platform string) (*models.ProjectIntegration, error) {
	for _, integration := range m.integrations {
		if integration.ProjectID == projectID && integration.Platform == platform {
			return integration, nil
		}
	}
	return nil, nil
}

func (m *MockDiscordStore) CreateProjectIntegration(ctx context.Context, integration *models.ProjectIntegration) error {
	m.integrations[integration.ID] = integration
	return nil
}

func (m *MockDiscordStore) GetProjectIntegration(ctx context.Context, integrationID string) (*models.ProjectIntegration, error) {
	if integration, ok := m.integrations[integrationID]; ok {
		return integration, nil
	}
	return nil, nil
}

func (m *MockDiscordStore) UpdateProjectIntegration(ctx context.Context, integrationID string, updates map[string]interface{}) error {
	if integration, ok := m.integrations[integrationID]; ok {
		if config, ok := updates["configuration"].(map[string]interface{}); ok {
			integration.Configuration = config
		}
		if updatedAt, ok := updates["updated_at"].(time.Time); ok {
			integration.UpdatedAt = updatedAt
		}
	}
	return nil
}

func (m *MockDiscordStore) DeleteProjectIntegration(ctx context.Context, integrationID string) error {
	delete(m.integrations, integrationID)
	return nil
}

func (m *MockDiscordStore) GetProjectDataSourcesByIntegration(ctx context.Context, integrationID string) ([]models.ProjectDataSource, error) {
	var sources []models.ProjectDataSource
	for _, source := range m.dataSources {
		if source.IntegrationID == integrationID {
			sources = append(sources, *source)
		}
	}
	return sources, nil
}

func (m *MockDiscordStore) GetProjectDataSource(ctx context.Context, dataSourceID string) (*models.ProjectDataSource, error) {
	if source, ok := m.dataSources[dataSourceID]; ok {
		return source, nil
	}
	return nil, nil
}

func (m *MockDiscordStore) CreateProjectDataSource(ctx context.Context, dataSource *models.ProjectDataSource) error {
	m.dataSources[dataSource.ID] = dataSource
	return nil
}

func (m *MockDiscordStore) DeleteProjectDataSource(ctx context.Context, dataSourceID string) error {
	delete(m.dataSources, dataSourceID)
	return nil
}

// TestDiscordIntegrationServiceCreation tests that the Discord integration service can be created
func TestDiscordIntegrationServiceCreation(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
	}
	
	store := NewMockDiscordStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}
	
	svc := NewDiscordIntegrationService(cfg, store, encryptSvc, logger)
	
	if svc == nil {
		t.Fatal("Expected Discord integration service to be created, got nil")
	}
}

// TestDiscordBotInstallationValidation tests bot installation validation
func TestDiscordBotInstallationValidation(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
	}
	
	store := NewMockDiscordStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}
	
	svc := NewDiscordIntegrationService(cfg, store, encryptSvc, logger)
	
	ctx := context.Background()
	
	// Test with empty bot token
	req := &DiscordBotInstallationRequest{
		ProjectID: "test-project",
		BotToken:  "",
	}
	
	_, err := svc.ProcessBotInstallation(ctx, req, "test-user")
	if err == nil {
		t.Error("Expected error for empty bot token, got nil")
	}
	if err != nil && err.Error() != "bot token required" {
		t.Errorf("Expected 'bot token required' error, got: %v", err)
	}
}

// TestDiscordChannelSelectionValidation tests channel selection validation
func TestDiscordChannelSelectionValidation(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
	}
	
	store := NewMockDiscordStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}
	
	svc := NewDiscordIntegrationService(cfg, store, encryptSvc, logger)
	
	ctx := context.Background()
	
	// Create a mock integration
	integration := &models.ProjectIntegration{
		ID:        "test-integration",
		ProjectID: "test-project",
		Platform:  string(models.PlatformDiscord),
		Status:    string(models.IntegrationStatusActive),
		Configuration: map[string]interface{}{
			"bot_id": "123456789",
		},
		Credentials: map[string]interface{}{
			"bot_token": "encrypted-token",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.integrations[integration.ID] = integration
	
	// Test with mismatched project ID
	req := &DiscordChannelSelectionRequest{
		ProjectID:     "wrong-project",
		IntegrationID: "test-integration",
		GuildID:       "test-guild",
		ChannelIDs:    []string{"channel1"},
	}
	
	_, err := svc.SelectChannels(ctx, req, "test-user")
	if err == nil {
		t.Error("Expected error for mismatched project ID, got nil")
	}
	if err != nil && err.Error() != "integration does not belong to project" {
		t.Errorf("Expected 'integration does not belong to project' error, got: %v", err)
	}
}

// TestDiscordConfigurationValidation tests configuration validation
func TestDiscordConfigurationValidation(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
	}
	
	store := NewMockDiscordStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}
	
	svc := NewDiscordIntegrationService(cfg, store, encryptSvc, logger)
	
	ctx := context.Background()
	
	// Create a mock integration
	integration := &models.ProjectIntegration{
		ID:        "test-integration",
		ProjectID: "test-project",
		Platform:  string(models.PlatformDiscord),
		Status:    string(models.IntegrationStatusActive),
		Configuration: map[string]interface{}{
			"bot_id": "123456789",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.integrations[integration.ID] = integration
	
	// Test with invalid thread_depth (too high)
	req := &IntegrationConfigurationRequest{
		ProjectID:     "test-project",
		IntegrationID: "test-integration",
		Configuration: map[string]interface{}{
			"thread_depth": float64(150),
		},
	}
	
	_, err := svc.UpdateConfiguration(ctx, req, "test-user")
	if err == nil {
		t.Error("Expected error for invalid thread_depth, got nil")
	}
	if err != nil && err.Error() != "invalid configuration: thread_depth must be between 1 and 100" {
		t.Errorf("Expected thread_depth validation error, got: %v", err)
	}
	
	// Test with valid thread_depth
	req.Configuration["thread_depth"] = float64(50)
	_, err = svc.UpdateConfiguration(ctx, req, "test-user")
	if err != nil {
		t.Errorf("Expected no error for valid configuration, got: %v", err)
	}
}

// TestDiscordIntegrationDeletion tests integration deletion
func TestDiscordIntegrationDeletion(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
	}
	
	store := NewMockDiscordStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}
	
	svc := NewDiscordIntegrationService(cfg, store, encryptSvc, logger)
	
	ctx := context.Background()
	
	// Create a mock integration with data sources
	integration := &models.ProjectIntegration{
		ID:        "test-integration",
		ProjectID: "test-project",
		Platform:  string(models.PlatformDiscord),
		Status:    string(models.IntegrationStatusActive),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.integrations[integration.ID] = integration
	
	dataSource := &models.ProjectDataSource{
		ID:            "test-datasource",
		ProjectID:     "test-project",
		IntegrationID: "test-integration",
		SourceType:    string(models.SourceTypeChannel),
		SourceID:      "channel1",
		SourceName:    "general",
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	store.dataSources[dataSource.ID] = dataSource
	
	// Delete integration
	err := svc.DeleteIntegration(ctx, "test-project", "test-integration", "test-user")
	if err != nil {
		t.Errorf("Expected no error for valid deletion, got: %v", err)
	}
	
	// Verify integration was deleted
	if _, exists := store.integrations["test-integration"]; exists {
		t.Error("Expected integration to be deleted")
	}
	
	// Verify data source was deleted
	if _, exists := store.dataSources["test-datasource"]; exists {
		t.Error("Expected data source to be deleted")
	}
}
