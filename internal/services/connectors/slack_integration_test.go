package connectors

import (
	"context"
	"testing"
	"time"
)

// TestSlackConnectorIntegration tests the complete Slack connector workflow
func TestSlackConnectorIntegration(t *testing.T) {
	// Create Slack connector configuration
	config := ConnectorConfig{
		Platform: "slack",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": "xoxb-test-token-12345",
			},
			Scopes: []string{"channels:history", "groups:history", "im:history"},
		},
		RateLimit: RateLimitConfig{
			RequestsPerHour:   1000,
			RequestsPerMinute: 50,
			BurstLimit:        5,
			BackoffMultiplier: 2.0,
			MaxRetries:        3,
		},
		SyncConfig: SyncConfig{
			BatchSize:       50,
			SyncInterval:    2 * time.Minute,
			MaxLookback:     7 * 24 * time.Hour,
			IncrementalSync: true,
		},
		Metadata: map[string]interface{}{
			"channels":     []string{"C1234567890", "C0987654321"},
			"include_dms":  true,
			"thread_depth": 10,
		},
	}

	// Create Slack connector
	connector, err := NewSlackConnector(config)
	if err != nil {
		t.Fatalf("Failed to create Slack connector: %v", err)
	}

	// Verify connector implements PlatformConnector interface
	if connector == nil {
		t.Fatal("Connector should not be nil")
	}

	// Test GetPlatformInfo
	info := connector.GetPlatformInfo()
	if info.Name != "slack" {
		t.Errorf("Expected platform name 'slack', got '%s'", info.Name)
	}
	if info.DisplayName != "Slack" {
		t.Errorf("Expected display name 'Slack', got '%s'", info.DisplayName)
	}
	if len(info.SupportedEvents) == 0 {
		t.Error("Expected supported events to be populated")
	}
	if len(info.RequiredScopes) == 0 {
		t.Error("Expected required scopes to be populated")
	}

	// Test ScheduleSync
	ctx := context.Background()
	lastSync := time.Now().Add(-5 * time.Minute)
	syncInterval, err := connector.ScheduleSync(ctx, lastSync)
	if err != nil {
		t.Errorf("ScheduleSync failed: %v", err)
	}
	if syncInterval <= 0 {
		t.Error("Sync interval should be positive")
	}

	// Test NormalizeData with sample events
	events := []PlatformEvent{
		{
			ID:        "msg-1234567890.123456",
			Type:      EventTypeMessage,
			Timestamp: time.Now(),
			Author:    "U1234567890",
			Content:   "Let's discuss the authentication feature in src/auth.go",
			Platform:  "slack",
			Metadata: map[string]interface{}{
				"channel_id": "C1234567890",
				"timestamp":  "1234567890.123456",
				"thread_ts":  "1234567890.123456",
			},
		},
		{
			ID:        "msg-1234567891.123456",
			Type:      EventTypeMessage,
			Timestamp: time.Now(),
			Author:    "U0987654321",
			Content:   "Good idea! We should also update the tests.",
			Platform:  "slack",
			Metadata: map[string]interface{}{
				"channel_id": "C1234567890",
				"timestamp":  "1234567891.123456",
				"thread_ts":  "1234567890.123456",
				"parent_id":  "1234567890.123456",
			},
		},
	}

	normalized, err := connector.NormalizeData(ctx, events)
	if err != nil {
		t.Fatalf("NormalizeData failed: %v", err)
	}

	if len(normalized) != len(events) {
		t.Errorf("Expected %d normalized events, got %d", len(events), len(normalized))
	}

	// Verify first event normalization
	if normalized[0].Platform != "slack" {
		t.Errorf("Expected platform 'slack', got '%s'", normalized[0].Platform)
	}
	if normalized[0].Author != "U1234567890" {
		t.Errorf("Expected author 'U1234567890', got '%s'", normalized[0].Author)
	}
	if normalized[0].Content != events[0].Content {
		t.Error("Content should be preserved in normalization")
	}

	// Verify thread context preservation in second event
	if normalized[1].ThreadID == nil {
		t.Error("ThreadID should be set for threaded message")
	}
	if normalized[1].ParentID == nil {
		t.Error("ParentID should be set for reply message")
	}
	if *normalized[1].ParentID != "1234567890.123456" {
		t.Errorf("Expected parent ID '1234567890.123456', got '%s'", *normalized[1].ParentID)
	}

	t.Log("Slack connector integration test passed successfully")
}

// TestSlackConnectorRegistry tests that Slack connector is properly registered
func TestSlackConnectorRegistry(t *testing.T) {
	registry := NewRegistry()
	
	// Register default connectors
	if err := registry.RegisterDefaultConnectors(); err != nil {
		t.Fatalf("Failed to register default connectors: %v", err)
	}

	// Verify Slack is registered
	platforms := registry.ListPlatforms()
	slackFound := false
	for _, platform := range platforms {
		if platform == "slack" {
			slackFound = true
			break
		}
	}

	if !slackFound {
		t.Error("Slack connector should be registered in default connectors")
	}

	// Set Slack configuration
	config := ConnectorConfig{
		Platform: "slack",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": "xoxb-test-token",
			},
		},
		RateLimit: RateLimitConfig{
			RequestsPerHour:   1000,
			RequestsPerMinute: 50,
			BurstLimit:        5,
			BackoffMultiplier: 2.0,
		},
		SyncConfig: SyncConfig{
			BatchSize:    50,
			SyncInterval: 2 * time.Minute,
		},
		Metadata: map[string]interface{}{
			"channels": []string{"C1234567890"},
		},
	}

	registry.SetConfig("slack", config)

	// Create Slack connector from registry
	connector, err := registry.CreateConnector("slack")
	if err != nil {
		t.Fatalf("Failed to create Slack connector from registry: %v", err)
	}

	if connector == nil {
		t.Fatal("Connector should not be nil")
	}

	// Verify it's a Slack connector
	info := connector.GetPlatformInfo()
	if info.Name != "slack" {
		t.Errorf("Expected platform name 'slack', got '%s'", info.Name)
	}

	t.Log("Slack connector registry test passed successfully")
}

// TestSlackConnectorErrorHandling tests error handling in Slack connector
func TestSlackConnectorErrorHandling(t *testing.T) {
	// Test missing bot token
	config := ConnectorConfig{
		Platform: "slack",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{},
		},
	}

	_, err := NewSlackConnector(config)
	if err == nil {
		t.Error("Expected error when bot token is missing")
	}

	connErr, ok := err.(*ConnectorError)
	if !ok {
		t.Error("Expected ConnectorError type")
	}
	if connErr.Platform != "slack" {
		t.Errorf("Expected platform 'slack', got '%s'", connErr.Platform)
	}
	if connErr.Code != "missing_bot_token" {
		t.Errorf("Expected error code 'missing_bot_token', got '%s'", connErr.Code)
	}
	if connErr.Retryable {
		t.Error("Missing bot token error should not be retryable")
	}

	t.Log("Slack connector error handling test passed successfully")
}

// TestSlackConnectorThreadHandling tests thread context preservation
func TestSlackConnectorThreadHandling(t *testing.T) {
	config := ConnectorConfig{
		Platform: "slack",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": "xoxb-test-token",
			},
		},
		Metadata: map[string]interface{}{
			"channels":     []string{"C1234567890"},
			"thread_depth": 10,
		},
	}

	connector, err := NewSlackConnector(config)
	if err != nil {
		t.Fatalf("Failed to create Slack connector: %v", err)
	}

	// Create parent message
	parentEvent := PlatformEvent{
		ID:        "msg-parent",
		Type:      EventTypeMessage,
		Timestamp: time.Now(),
		Author:    "U1234567890",
		Content:   "Parent message",
		Platform:  "slack",
		Metadata: map[string]interface{}{
			"channel_id":  "C1234567890",
			"timestamp":   "1234567890.123456",
			"thread_ts":   "1234567890.123456",
			"reply_count": 2,
		},
	}

	// Create reply messages
	replyEvent1 := PlatformEvent{
		ID:        "msg-reply-1",
		Type:      EventTypeMessage,
		Timestamp: time.Now(),
		Author:    "U0987654321",
		Content:   "Reply 1",
		Platform:  "slack",
		Metadata: map[string]interface{}{
			"channel_id": "C1234567890",
			"timestamp":  "1234567891.123456",
			"thread_ts":  "1234567890.123456",
			"parent_id":  "1234567890.123456",
		},
	}

	replyEvent2 := PlatformEvent{
		ID:        "msg-reply-2",
		Type:      EventTypeMessage,
		Timestamp: time.Now(),
		Author:    "U1111111111",
		Content:   "Reply 2",
		Platform:  "slack",
		Metadata: map[string]interface{}{
			"channel_id": "C1234567890",
			"timestamp":  "1234567892.123456",
			"thread_ts":  "1234567890.123456",
			"parent_id":  "1234567890.123456",
		},
	}

	events := []PlatformEvent{parentEvent, replyEvent1, replyEvent2}

	ctx := context.Background()
	normalized, err := connector.NormalizeData(ctx, events)
	if err != nil {
		t.Fatalf("NormalizeData failed: %v", err)
	}

	// Verify parent message
	if normalized[0].ThreadID == nil {
		t.Error("Parent message should have ThreadID set")
	}
	if normalized[0].ParentID != nil {
		t.Error("Parent message should not have ParentID")
	}

	// Verify reply messages
	for i := 1; i < len(normalized); i++ {
		if normalized[i].ThreadID == nil {
			t.Errorf("Reply %d should have ThreadID set", i)
		}
		if normalized[i].ParentID == nil {
			t.Errorf("Reply %d should have ParentID set", i)
		}
		if *normalized[i].ThreadID != "1234567890.123456" {
			t.Errorf("Reply %d has incorrect ThreadID", i)
		}
		if *normalized[i].ParentID != "1234567890.123456" {
			t.Errorf("Reply %d has incorrect ParentID", i)
		}
	}

	t.Log("Slack connector thread handling test passed successfully")
}
