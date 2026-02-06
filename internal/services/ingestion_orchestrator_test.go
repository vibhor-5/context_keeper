package services

import (
	"context"
	"testing"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/DevAnuragT/context_keeper/internal/services/connectors"
)

// MockConnectorForOrchestrator implements PlatformConnector for testing
type MockConnectorForOrchestrator struct {
	events          []connectors.PlatformEvent
	normalizedEvents []connectors.NormalizedEvent
	fetchError      error
	normalizeError  error
}

func (m *MockConnectorForOrchestrator) Authenticate(ctx context.Context, config connectors.AuthConfig) (*connectors.AuthResult, error) {
	return &connectors.AuthResult{Success: true}, nil
}

func (m *MockConnectorForOrchestrator) FetchEvents(ctx context.Context, since time.Time, limit int) ([]connectors.PlatformEvent, error) {
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	return m.events, nil
}

func (m *MockConnectorForOrchestrator) NormalizeData(ctx context.Context, events []connectors.PlatformEvent) ([]connectors.NormalizedEvent, error) {
	if m.normalizeError != nil {
		return nil, m.normalizeError
	}
	return m.normalizedEvents, nil
}

func (m *MockConnectorForOrchestrator) ScheduleSync(ctx context.Context, lastSync time.Time) (time.Duration, error) {
	return time.Hour, nil
}

func (m *MockConnectorForOrchestrator) GetPlatformInfo() connectors.PlatformInfo {
	return connectors.PlatformInfo{
		Name:        "mock",
		DisplayName: "Mock Platform",
		Version:     "1.0",
	}
}

// MockContextProcessor implements ContextProcessorService for testing
type MockContextProcessor struct {
	processError error
	result       *ProcessingResult
}

func (m *MockContextProcessor) ProcessEvents(ctx context.Context, events []NormalizedEvent) (*ProcessingResult, error) {
	if m.processError != nil {
		return nil, m.processError
	}
	if m.result != nil {
		return m.result, nil
	}
	return &ProcessingResult{
		DecisionRecords:    []models.DecisionRecord{},
		DiscussionSummaries: []models.DiscussionSummary{},
		FeatureContexts:    []models.FeatureContext{},
		FileContexts:       []models.FileContextHistory{},
		Relationships:      []models.KnowledgeRelationship{},
	}, nil
}

func setupTestOrchestrator(t *testing.T, ctx context.Context, store RepositoryStore) (IngestionOrchestrator, *connectors.ConnectorManager) {
	// Create connector manager
	registry := connectors.NewRegistry()
	mockConnector := &MockConnectorForOrchestrator{
		events: []connectors.PlatformEvent{
			{
				ID:        "event-1",
				Type:      connectors.EventTypePullRequest,
				Timestamp: time.Now(),
				Author:    "user-1",
				Content:   "Test PR",
				Metadata:  make(map[string]interface{}),
			},
		},
		normalizedEvents: []connectors.NormalizedEvent{
			{
				PlatformID: "event-1",
				EventType:  connectors.EventTypePullRequest,
				Timestamp:  time.Now(),
				Author:     "user-1",
				Content:    "Test PR",
				Platform:   "github",
				Metadata:   make(map[string]interface{}),
			},
		},
	}
	
	registry.Register("github", func(config connectors.ConnectorConfig) (connectors.PlatformConnector, error) {
		return mockConnector, nil
	})
	registry.Register("slack", func(config connectors.ConnectorConfig) (connectors.PlatformConnector, error) {
		return mockConnector, nil
	})
	
	registry.SetConfig("github", connectors.ConnectorConfig{
		Platform: "github",
		Enabled:  true,
	})
	registry.SetConfig("slack", connectors.ConnectorConfig{
		Platform: "slack",
		Enabled:  true,
	})
	
	connectorManager := connectors.NewConnectorManager(registry)
	connectorManager.InitializeConnectors(ctx)
	
	logger := &SimpleLogger{}
	encryptSvc := NewEncryptionService("test-key-32-bytes-long-for-aes")
	contextProcessor := &MockContextProcessor{}
	
	orchestrator := NewIngestionOrchestrator(
		store,
		connectorManager,
		contextProcessor,
		encryptSvc,
		logger,
	)
	
	return orchestrator, connectorManager
}

func TestIngestionOrchestrator_StartProjectIngestion(t *testing.T) {
	ctx := context.Background()
	
	// Create mock store
	store := NewMockRepositoryStore()
	
	// Create test project
	projectID := "test-project-1"
	project := &models.ProjectWorkspace{
		ID:        projectID,
		Name:      "Test Project",
		OwnerID:   "user-1",
		Settings:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.CreateProjectWorkspace(ctx, project)
	
	// Create test integrations
	integration1 := &models.ProjectIntegration{
		ID:              "integration-1",
		ProjectID:       projectID,
		Platform:        string(models.PlatformGitHub),
		IntegrationType: string(models.IntegrationTypeOAuth),
		Status:          string(models.IntegrationStatusActive),
		Configuration:   make(map[string]interface{}),
		Credentials:     make(map[string]interface{}),
		SyncCheckpoint:  make(map[string]interface{}),
		CreatedBy:       "user-1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.CreateProjectIntegration(ctx, integration1)
	
	integration2 := &models.ProjectIntegration{
		ID:              "integration-2",
		ProjectID:       projectID,
		Platform:        string(models.PlatformSlack),
		IntegrationType: string(models.IntegrationTypeOAuth),
		Status:          string(models.IntegrationStatusActive),
		Configuration:   make(map[string]interface{}),
		Credentials:     make(map[string]interface{}),
		SyncCheckpoint:  make(map[string]interface{}),
		CreatedBy:       "user-1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.CreateProjectIntegration(ctx, integration2)
	
	// Create data sources
	dataSource1 := &models.ProjectDataSource{
		ID:            "ds-1",
		ProjectID:     projectID,
		IntegrationID: "integration-1",
		SourceType:    string(models.SourceTypeRepository),
		SourceID:      "repo-1",
		SourceName:    "test/repo",
		Configuration: make(map[string]interface{}),
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	store.CreateProjectDataSource(ctx, dataSource1)
	
	orchestrator, _ := setupTestOrchestrator(t, ctx, store)
	
	// Test starting project ingestion
	err := orchestrator.StartProjectIngestion(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to start project ingestion: %v", err)
	}
	
	// Give some time for background tasks to start
	time.Sleep(100 * time.Millisecond)
	
	// Verify integrations were started
	impl := orchestrator.(*IngestionOrchestratorImpl)
	impl.mu.RLock()
	activeCount := len(impl.activeIngestions)
	impl.mu.RUnlock()
	
	if activeCount != 2 {
		t.Errorf("Expected 2 active ingestions, got %d", activeCount)
	}
}

func TestIngestionOrchestrator_GetIngestionHealth(t *testing.T) {
	ctx := context.Background()
	
	// Create mock store
	store := NewMockRepositoryStore()
	
	// Create test project
	projectID := "test-project-1"
	project := &models.ProjectWorkspace{
		ID:        projectID,
		Name:      "Test Project",
		OwnerID:   "user-1",
		Settings:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.CreateProjectWorkspace(ctx, project)
	
	// Create test integrations with different statuses
	now := time.Now()
	successStatus := "success"
	
	integration1 := &models.ProjectIntegration{
		ID:              "integration-1",
		ProjectID:       projectID,
		Platform:        string(models.PlatformGitHub),
		IntegrationType: string(models.IntegrationTypeOAuth),
		Status:          string(models.IntegrationStatusActive),
		Configuration:   make(map[string]interface{}),
		Credentials:     make(map[string]interface{}),
		SyncCheckpoint:  make(map[string]interface{}),
		LastSyncAt:      &now,
		LastSyncStatus:  &successStatus,
		CreatedBy:       "user-1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.CreateProjectIntegration(ctx, integration1)
	
	errorMsg := "test error"
	failedStatus := "failed"
	integration2 := &models.ProjectIntegration{
		ID:              "integration-2",
		ProjectID:       projectID,
		Platform:        string(models.PlatformSlack),
		IntegrationType: string(models.IntegrationTypeOAuth),
		Status:          string(models.IntegrationStatusError),
		Configuration:   make(map[string]interface{}),
		Credentials:     make(map[string]interface{}),
		SyncCheckpoint:  make(map[string]interface{}),
		LastSyncAt:      &now,
		LastSyncStatus:  &failedStatus,
		ErrorMessage:    &errorMsg,
		CreatedBy:       "user-1",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.CreateProjectIntegration(ctx, integration2)
	
	// Create data sources
	dataSource1 := &models.ProjectDataSource{
		ID:            "ds-1",
		ProjectID:     projectID,
		IntegrationID: "integration-1",
		SourceType:    string(models.SourceTypeRepository),
		SourceID:      "repo-1",
		SourceName:    "test/repo",
		Configuration: make(map[string]interface{}),
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	store.CreateProjectDataSource(ctx, dataSource1)
	
	dataSource2 := &models.ProjectDataSource{
		ID:            "ds-2",
		ProjectID:     projectID,
		IntegrationID: "integration-2",
		SourceType:    string(models.SourceTypeChannel),
		SourceID:      "channel-1",
		SourceName:    "test-channel",
		Configuration: make(map[string]interface{}),
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	store.CreateProjectDataSource(ctx, dataSource2)
	
	orchestrator, _ := setupTestOrchestrator(t, ctx, store)
	
	// Test getting ingestion health
	health, err := orchestrator.GetIngestionHealth(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to get ingestion health: %v", err)
	}
	
	// Verify health status
	if health.ProjectID != projectID {
		t.Errorf("Expected project ID %s, got %s", projectID, health.ProjectID)
	}
	
	if health.ActiveIntegrations != 2 {
		t.Errorf("Expected 2 active integrations, got %d", health.ActiveIntegrations)
	}
	
	if health.FailedIntegrations != 1 {
		t.Errorf("Expected 1 failed integration, got %d", health.FailedIntegrations)
	}
	
	if health.OverallStatus != "degraded" {
		t.Errorf("Expected overall status 'degraded', got '%s'", health.OverallStatus)
	}
	
	if len(health.Integrations) != 2 {
		t.Errorf("Expected 2 integration health statuses, got %d", len(health.Integrations))
	}
}

func TestIngestionOrchestrator_SyncCheckpoint(t *testing.T) {
	ctx := context.Background()
	
	// Create mock store
	store := NewMockRepositoryStore()
	
	// Create test project and integration
	projectID := "test-project-1"
	project := &models.ProjectWorkspace{
		ID:        projectID,
		Name:      "Test Project",
		OwnerID:   "user-1",
		Settings:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.CreateProjectWorkspace(ctx, project)
	
	integration := &models.ProjectIntegration{
		ID:              "integration-1",
		ProjectID:       projectID,
		Platform:        string(models.PlatformGitHub),
		IntegrationType: string(models.IntegrationTypeOAuth),
		Status:          string(models.IntegrationStatusActive),
		Configuration:   make(map[string]interface{}),
		Credentials:     make(map[string]interface{}),
		SyncCheckpoint: map[string]interface{}{
			"last_sync_time": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			"total_events_processed": 100,
		},
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.CreateProjectIntegration(ctx, integration)
	
	orchestrator, _ := setupTestOrchestrator(t, ctx, store)
	
	// Test getting sync checkpoint
	checkpoint, err := orchestrator.GetSyncCheckpoint(ctx, "integration-1")
	if err != nil {
		t.Fatalf("Failed to get sync checkpoint: %v", err)
	}
	
	if checkpoint["total_events_processed"] != 100 {
		t.Errorf("Expected total_events_processed to be 100, got %v", checkpoint["total_events_processed"])
	}
	
	// Test updating sync checkpoint
	newCheckpoint := map[string]interface{}{
		"last_sync_time":         time.Now().Format(time.RFC3339),
		"total_events_processed": 150,
		"latest_event_timestamp": time.Now().Format(time.RFC3339),
	}
	
	err = orchestrator.UpdateSyncCheckpoint(ctx, "integration-1", newCheckpoint)
	if err != nil {
		t.Fatalf("Failed to update sync checkpoint: %v", err)
	}
	
	// Verify checkpoint was updated
	updatedCheckpoint, err := orchestrator.GetSyncCheckpoint(ctx, "integration-1")
	if err != nil {
		t.Fatalf("Failed to get updated sync checkpoint: %v", err)
	}
	
	if updatedCheckpoint["total_events_processed"] != 150 {
		t.Errorf("Expected total_events_processed to be 150, got %v", updatedCheckpoint["total_events_processed"])
	}
}

func TestIngestionOrchestrator_Deduplication(t *testing.T) {
	ctx := context.Background()
	
	// Create mock store
	store := NewMockRepositoryStore()
	
	orchestrator, _ := setupTestOrchestrator(t, ctx, store)
	impl := orchestrator.(*IngestionOrchestratorImpl)
	
	// Create test events
	events := []connectors.PlatformEvent{
		{
			ID:        "event-1",
			Type:      connectors.EventTypePullRequest,
			Timestamp: time.Now(),
			Author:    "user-1",
			Content:   "Test PR 1",
			Metadata:  make(map[string]interface{}),
		},
		{
			ID:        "event-2",
			Type:      connectors.EventTypePullRequest,
			Timestamp: time.Now(),
			Author:    "user-1",
			Content:   "Test PR 2",
			Metadata:  make(map[string]interface{}),
		},
		{
			ID:        "event-3",
			Type:      connectors.EventTypePullRequest,
			Timestamp: time.Now(),
			Author:    "user-1",
			Content:   "Test PR 3",
			Metadata:  make(map[string]interface{}),
		},
	}
	
	// Create checkpoint with some processed events
	checkpoint := map[string]interface{}{
		"processed_event_ids": []interface{}{"event-1", "event-2"},
	}
	
	// Test deduplication
	deduplicated := impl.deduplicateEvents(ctx, "integration-1", events, checkpoint)
	
	// Verify only event-3 remains
	if len(deduplicated) != 1 {
		t.Errorf("Expected 1 deduplicated event, got %d", len(deduplicated))
	}
	
	if len(deduplicated) > 0 && deduplicated[0].ID != "event-3" {
		t.Errorf("Expected event-3 to remain, got %s", deduplicated[0].ID)
	}
}
