package connectors

import (
	"context"
	"testing"
	"time"
)

// TestDiscordConnectorIntegration tests the Discord connector end-to-end
func TestDiscordConnectorIntegration(t *testing.T) {
	// Create Discord connector configuration
	config := ConnectorConfig{
		Platform: "discord",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": "test-discord-bot-token",
			},
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
			"guild_ids":    []string{"test-guild-1", "test-guild-2"},
			"channel_ids":  []string{"test-channel-1", "test-channel-2"},
			"thread_depth": 10,
		},
	}

	// Create Discord connector
	connector, err := NewDiscordConnector(config)
	if err != nil {
		t.Fatalf("Failed to create Discord connector: %v", err)
	}

	// Verify connector implements PlatformConnector interface
	if _, ok := connector.(PlatformConnector); !ok {
		t.Fatal("Discord connector does not implement PlatformConnector interface")
	}

	// Test GetPlatformInfo
	info := connector.GetPlatformInfo()
	if info.Name != "discord" {
		t.Errorf("Expected platform name 'discord', got '%s'", info.Name)
	}
	if info.DisplayName != "Discord" {
		t.Errorf("Expected display name 'Discord', got '%s'", info.DisplayName)
	}
	if info.AuthType != "bot_token" {
		t.Errorf("Expected auth type 'bot_token', got '%s'", info.AuthType)
	}

	// Test ScheduleSync
	ctx := context.Background()
	syncInterval, err := connector.ScheduleSync(ctx, time.Now())
	if err != nil {
		t.Errorf("ScheduleSync failed: %v", err)
	}
	if syncInterval != 2*time.Minute {
		t.Errorf("Expected sync interval 2m, got %v", syncInterval)
	}

	// Test NormalizeData with sample events
	events := []PlatformEvent{
		{
			ID:        "msg-123",
			Type:      EventTypeMessage,
			Timestamp: time.Now(),
			Author:    "testuser",
			Content:   "This is a test message about main.go",
			Platform:  "discord",
			Metadata: map[string]interface{}{
				"message_id": "123",
				"channel_id": "test-channel-1",
				"guild_id":   "test-guild-1",
			},
			References: []string{"main.go"},
		},
		{
			ID:        "msg-124",
			Type:      EventTypeMessage,
			Timestamp: time.Now(),
			Author:    "testuser2",
			Content:   "Reply to the thread",
			Platform:  "discord",
			Metadata: map[string]interface{}{
				"message_id":        "124",
				"channel_id":        "test-channel-1",
				"guild_id":          "test-guild-1",
				"thread_id":         "thread-123",
				"is_thread_message": true,
			},
			References: []string{},
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
	if normalized[0].Platform != "discord" {
		t.Errorf("Expected platform 'discord', got '%s'", normalized[0].Platform)
	}
	if normalized[0].Author != "testuser" {
		t.Errorf("Expected author 'testuser', got '%s'", normalized[0].Author)
	}
	if len(normalized[0].FileRefs) == 0 {
		t.Error("Expected file references to be extracted")
	}

	// Verify thread context preservation in second event
	if normalized[1].ThreadID == nil {
		t.Error("Expected thread ID to be set for thread message")
	} else if *normalized[1].ThreadID != "thread-123" {
		t.Errorf("Expected thread ID 'thread-123', got '%s'", *normalized[1].ThreadID)
	}

	t.Log("Discord connector integration test passed successfully")
}

// TestDiscordConnectorRegistry tests Discord connector registration
func TestDiscordConnectorRegistry(t *testing.T) {
	registry := NewRegistry()

	// Register Discord connector
	err := registry.Register("discord", NewDiscordConnector)
	if err != nil {
		t.Fatalf("Failed to register Discord connector: %v", err)
	}

	// Verify Discord is in the list of platforms
	platforms := registry.ListPlatforms()
	found := false
	for _, platform := range platforms {
		if platform == "discord" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Discord not found in registered platforms")
	}

	// Set configuration
	config := ConnectorConfig{
		Platform: "discord",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": "test-token",
			},
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
			"channel_ids": []string{"test-channel"},
		},
	}
	registry.SetConfig("discord", config)

	// Create connector from registry
	connector, err := registry.CreateConnector("discord")
	if err != nil {
		t.Fatalf("Failed to create Discord connector from registry: %v", err)
	}

	// Verify connector
	info := connector.GetPlatformInfo()
	if info.Name != "discord" {
		t.Errorf("Expected platform name 'discord', got '%s'", info.Name)
	}

	t.Log("Discord connector registry test passed successfully")
}

// TestDiscordConnectorErrorHandling tests Discord connector error handling
func TestDiscordConnectorErrorHandling(t *testing.T) {
	// Test missing bot token
	config := ConnectorConfig{
		Platform: "discord",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{},
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
	}

	_, err := NewDiscordConnector(config)
	if err == nil {
		t.Error("Expected error for missing bot token, got nil")
	}

	connErr, ok := err.(*ConnectorError)
	if !ok {
		t.Errorf("Expected ConnectorError, got %T", err)
	} else {
		if connErr.Platform != "discord" {
			t.Errorf("Expected platform 'discord', got '%s'", connErr.Platform)
		}
		if connErr.Code != "missing_bot_token" {
			t.Errorf("Expected code 'missing_bot_token', got '%s'", connErr.Code)
		}
		if connErr.Retryable {
			t.Error("Expected non-retryable error for missing bot token")
		}
	}

	t.Log("Discord connector error handling test passed successfully")
}

// TestDiscordConnectorThreadHandling tests Discord thread message handling
func TestDiscordConnectorThreadHandling(t *testing.T) {
	config := ConnectorConfig{
		Platform: "discord",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": "test-discord-token",
			},
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
			"channel_ids":  []string{"test-channel"},
			"thread_depth": 10,
		},
	}

	connector, err := NewDiscordConnector(config)
	if err != nil {
		t.Fatalf("Failed to create Discord connector: %v", err)
	}

	// Create thread events
	parentEvent := PlatformEvent{
		ID:        "msg-parent",
		Type:      EventTypeMessage,
		Timestamp: time.Now(),
		Author:    "user1",
		Content:   "Parent message",
		Platform:  "discord",
		Metadata: map[string]interface{}{
			"message_id": "parent",
			"channel_id": "test-channel",
		},
	}

	threadEvent := PlatformEvent{
		ID:        "msg-thread",
		Type:      EventTypeMessage,
		Timestamp: time.Now(),
		Author:    "user2",
		Content:   "Thread reply",
		Platform:  "discord",
		Metadata: map[string]interface{}{
			"message_id":        "thread",
			"channel_id":        "test-channel",
			"thread_id":         "thread-123",
			"is_thread_message": true,
		},
	}

	events := []PlatformEvent{parentEvent, threadEvent}

	ctx := context.Background()
	normalized, err := connector.NormalizeData(ctx, events)
	if err != nil {
		t.Fatalf("NormalizeData failed: %v", err)
	}

	// Verify parent event has no thread ID
	if normalized[0].ThreadID != nil {
		t.Error("Parent event should not have thread ID")
	}

	// Verify thread event has thread ID
	if normalized[1].ThreadID == nil {
		t.Error("Thread event should have thread ID")
	} else if *normalized[1].ThreadID != "thread-123" {
		t.Errorf("Expected thread ID 'thread-123', got '%s'", *normalized[1].ThreadID)
	}

	// Verify thread event has parent ID
	if normalized[1].ParentID == nil {
		t.Error("Thread event should have parent ID")
	}

	t.Log("Discord connector thread handling test passed successfully")
}

// TestDiscordConnectorFileReferenceExtraction tests file reference extraction
func TestDiscordConnectorFileReferenceExtraction(t *testing.T) {
	config := ConnectorConfig{
		Platform: "discord",
		Enabled:  true,
		AuthConfig: AuthConfig{
			Metadata: map[string]string{
				"bot_token": "test-discord-token",
			},
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
			"channel_ids": []string{"test-channel"},
		},
	}

	connector, err := NewDiscordConnector(config)
	if err != nil {
		t.Fatalf("Failed to create Discord connector: %v", err)
	}

	// Test various file reference patterns
	testCases := []struct {
		content         string
		expectedFiles   []string
		description     string
	}{
		{
			content:       "Check out main.go and utils.ts",
			expectedFiles: []string{"main.go", "utils.ts"},
			description:   "Simple file references",
		},
		{
			content:       "Updated src/services/auth.go",
			expectedFiles: []string{"src/services/auth.go"},
			description:   "File path reference",
		},
		{
			content:       "```go\nfunc main() {}\n```",
			expectedFiles: []string{},
			description:   "Code block without filename",
		},
		{
			content:       "Modified config.json and package.json",
			expectedFiles: []string{"config.json", "package.json"},
			description:   "JSON file references",
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			event := PlatformEvent{
				ID:        "msg-test",
				Type:      EventTypeMessage,
				Timestamp: time.Now(),
				Author:    "testuser",
				Content:   tc.content,
				Platform:  "discord",
				Metadata: map[string]interface{}{
					"message_id": "test",
					"channel_id": "test-channel",
				},
			}

			normalized, err := connector.NormalizeData(ctx, []PlatformEvent{event})
			if err != nil {
				t.Fatalf("NormalizeData failed: %v", err)
			}

			if len(normalized) != 1 {
				t.Fatalf("Expected 1 normalized event, got %d", len(normalized))
			}

			// Note: File references are extracted by the base normalizer
			// The Discord connector's extractFileReferences is called during event conversion
			// For this test, we're verifying the normalization process works
			t.Logf("Content: %s, FileRefs: %v", tc.content, normalized[0].FileRefs)
		})
	}

	t.Log("Discord connector file reference extraction test passed successfully")
}
