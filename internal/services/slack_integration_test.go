package services

import (
	"context"
	"testing"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/config"
	"github.com/DevAnuragT/context_keeper/internal/models"
)

// MockSlackStore implements RepositoryStore for testing
type MockSlackStore struct {
	integrations map[string]*models.ProjectIntegration
	dataSources  map[string]*models.ProjectDataSource
}

func NewMockSlackStore() *MockSlackStore {
	return &MockSlackStore{
		integrations: make(map[string]*models.ProjectIntegration),
		dataSources:  make(map[string]*models.ProjectDataSource),
	}
}

func (m *MockSlackStore) GetProjectIntegrationByPlatform(ctx context.Context, projectID, platform string) (*models.ProjectIntegration, error) {
	for _, integration := range m.integrations {
		if integration.ProjectID == projectID && integration.Platform == platform {
			return integration, nil
		}
	}
	return nil, nil
}

func (m *MockSlackStore) CreateProjectIntegration(ctx context.Context, integration *models.ProjectIntegration) error {
	m.integrations[integration.ID] = integration
	return nil
}

func (m *MockSlackStore) GetProjectIntegration(ctx context.Context, integrationID string) (*models.ProjectIntegration, error) {
	if integration, ok := m.integrations[integrationID]; ok {
		return integration, nil
	}
	return nil, nil
}

func (m *MockSlackStore) UpdateProjectIntegration(ctx context.Context, integrationID string, updates map[string]interface{}) error {
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

func (m *MockSlackStore) DeleteProjectIntegration(ctx context.Context, integrationID string) error {
	delete(m.integrations, integrationID)
	return nil
}

func (m *MockSlackStore) GetProjectDataSourcesByIntegration(ctx context.Context, integrationID string) ([]models.ProjectDataSource, error) {
	var sources []models.ProjectDataSource
	for _, source := range m.dataSources {
		if source.IntegrationID == integrationID {
			sources = append(sources, *source)
		}
	}
	return sources, nil
}

func (m *MockSlackStore) GetProjectDataSource(ctx context.Context, dataSourceID string) (*models.ProjectDataSource, error) {
	if source, ok := m.dataSources[dataSourceID]; ok {
		return source, nil
	}
	return nil, nil
}

func (m *MockSlackStore) CreateProjectDataSource(ctx context.Context, dataSource *models.ProjectDataSource) error {
	m.dataSources[dataSource.ID] = dataSource
	return nil
}

func (m *MockSlackStore) DeleteProjectDataSource(ctx context.Context, dataSourceID string) error {
	delete(m.dataSources, dataSourceID)
	return nil
}

// TestSlackIntegrationServiceCreation tests service creation
func TestSlackIntegrationServiceCreation(t *testing.T) {
	cfg := &config.Config{
		SlackOAuth: config.SlackOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/api/auth/slack",
		},
	}

	store := NewMockSlackStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}

	svc := NewSlackIntegrationService(cfg, store, encryptSvc, logger)

	if svc == nil {
		t.Fatal("Expected service to be created")
	}
}

// TestSlackIntegrationValidateConfiguration tests configuration validation
func TestSlackIntegrationValidateConfiguration(t *testing.T) {
	cfg := &config.Config{
		SlackOAuth: config.SlackOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	store := NewMockSlackStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}

	svc := NewSlackIntegrationService(cfg, store, encryptSvc, logger).(*SlackIntegrationServiceImpl)

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
	}{
		{
			name: "valid configuration",
			config: map[string]interface{}{
				"thread_depth": float64(10),
				"include_dms":  true,
			},
			expectError: false,
		},
		{
			name: "invalid thread_depth - too low",
			config: map[string]interface{}{
				"thread_depth": float64(0),
			},
			expectError: true,
		},
		{
			name: "invalid thread_depth - too high",
			config: map[string]interface{}{
				"thread_depth": float64(101),
			},
			expectError: true,
		},
		{
			name: "invalid include_dms type",
			config: map[string]interface{}{
				"include_dms": "yes",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateConfiguration(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestSlackIntegrationDeleteIntegration tests integration deletion
func TestSlackIntegrationDeleteIntegration(t *testing.T) {
	cfg := &config.Config{
		SlackOAuth: config.SlackOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	store := NewMockSlackStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}

	svc := NewSlackIntegrationService(cfg, store, encryptSvc, logger)

	// Create test integration
	integration := &models.ProjectIntegration{
		ID:        "test-integration-id",
		ProjectID: "test-project-id",
		Platform:  string(models.PlatformSlack),
		Status:    string(models.IntegrationStatusActive),
	}
	store.integrations[integration.ID] = integration

	// Create test data sources
	dataSource1 := &models.ProjectDataSource{
		ID:            "test-source-1",
		ProjectID:     "test-project-id",
		IntegrationID: "test-integration-id",
		SourceType:    string(models.SourceTypeChannel),
	}
	dataSource2 := &models.ProjectDataSource{
		ID:            "test-source-2",
		ProjectID:     "test-project-id",
		IntegrationID: "test-integration-id",
		SourceType:    string(models.SourceTypeChannel),
	}
	store.dataSources[dataSource1.ID] = dataSource1
	store.dataSources[dataSource2.ID] = dataSource2

	// Delete integration
	ctx := context.Background()
	err := svc.DeleteIntegration(ctx, "test-project-id", "test-integration-id", "test-user-id")
	if err != nil {
		t.Fatalf("DeleteIntegration failed: %v", err)
	}

	// Verify integration is deleted
	if _, ok := store.integrations["test-integration-id"]; ok {
		t.Error("Integration should be deleted")
	}

	// Verify data sources are deleted
	if _, ok := store.dataSources["test-source-1"]; ok {
		t.Error("Data source 1 should be deleted")
	}
	if _, ok := store.dataSources["test-source-2"]; ok {
		t.Error("Data source 2 should be deleted")
	}
}

// TestSlackIntegrationUpdateConfiguration tests configuration updates
func TestSlackIntegrationUpdateConfiguration(t *testing.T) {
	cfg := &config.Config{
		SlackOAuth: config.SlackOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	store := NewMockSlackStore()
	encryptSvc := NewEncryptionService(cfg)
	logger := &SimpleLogger{}

	svc := NewSlackIntegrationService(cfg, store, encryptSvc, logger)

	// Create test integration
	integration := &models.ProjectIntegration{
		ID:        "test-integration-id",
		ProjectID: "test-project-id",
		Platform:  string(models.PlatformSlack),
		Status:    string(models.IntegrationStatusActive),
		Configuration: map[string]interface{}{
			"workspace_id": "T1234567890",
		},
	}
	store.integrations[integration.ID] = integration

	// Update configuration
	ctx := context.Background()
	req := &IntegrationConfigurationRequest{
		ProjectID:     "test-project-id",
		IntegrationID: "test-integration-id",
		Configuration: map[string]interface{}{
			"thread_depth": float64(20),
			"include_dms":  true,
		},
	}

	updatedIntegration, err := svc.UpdateConfiguration(ctx, req, "test-user-id")
	if err != nil {
		t.Fatalf("UpdateConfiguration failed: %v", err)
	}

	// Verify configuration is updated
	if updatedIntegration.Configuration["thread_depth"] != float64(20) {
		t.Error("thread_depth should be updated to 20")
	}
	if updatedIntegration.Configuration["include_dms"] != true {
		t.Error("include_dms should be updated to true")
	}
	// Verify existing configuration is preserved
	if updatedIntegration.Configuration["workspace_id"] != "T1234567890" {
		t.Error("workspace_id should be preserved")
	}
}
