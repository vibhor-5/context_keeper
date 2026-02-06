package connectors

import (
	"context"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

// TestProperty1_PlatformConnectorInterfaceCompliance tests that any platform connector
// implementation has all required interface methods with correct signatures
// **Feature: mcp-context-engine, Property 1: Platform Connector Interface Compliance**
// **Validates: Requirements 1.1, 2.6, 4.1**
func TestProperty1_PlatformConnectorInterfaceCompliance(t *testing.T) {
	property := func() bool {
		// Test that the PlatformConnector interface has all required methods
		connectorType := reflect.TypeOf((*PlatformConnector)(nil)).Elem()
		
		// Check that interface has exactly 5 methods
		if connectorType.NumMethod() != 5 {
			t.Logf("Expected 5 methods, got %d", connectorType.NumMethod())
			return false
		}
		
		// Check each required method exists with correct signature
		requiredMethods := map[string]struct {
			numIn  int
			numOut int
		}{
			"Authenticate":    {numIn: 2, numOut: 2}, // (ctx, config) -> (*AuthResult, error)
			"FetchEvents":     {numIn: 3, numOut: 2}, // (ctx, since, limit) -> ([]PlatformEvent, error)
			"NormalizeData":   {numIn: 2, numOut: 2}, // (ctx, events) -> ([]NormalizedEvent, error)
			"ScheduleSync":    {numIn: 2, numOut: 2}, // (ctx, lastSync) -> (time.Duration, error)
			"GetPlatformInfo": {numIn: 0, numOut: 1}, // () -> PlatformInfo
		}
		
		for i := 0; i < connectorType.NumMethod(); i++ {
			method := connectorType.Method(i)
			expected, exists := requiredMethods[method.Name]
			
			if !exists {
				t.Logf("Unexpected method: %s", method.Name)
				return false
			}
			
			if method.Type.NumIn() != expected.numIn {
				t.Logf("Method %s: expected %d inputs, got %d", method.Name, expected.numIn, method.Type.NumIn())
				return false
			}
			
			if method.Type.NumOut() != expected.numOut {
				t.Logf("Method %s: expected %d outputs, got %d", method.Name, expected.numOut, method.Type.NumOut())
				return false
			}
		}
		
		return true
	}
	
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// TestProperty2_ConnectorIsolationAndExtensibility tests that adding or removing
// connectors doesn't affect other connectors
// **Feature: mcp-context-engine, Property 2: Connector Isolation and Extensibility**
// **Validates: Requirements 1.2, 9.3**
func TestProperty2_ConnectorIsolationAndExtensibility(t *testing.T) {
	property := func(platformCount uint8) bool {
		if platformCount == 0 || platformCount > 10 {
			return true // Skip invalid inputs
		}
		
		registry := NewRegistry()
		
		// Register multiple mock connectors
		platforms := make([]string, platformCount)
		for i := uint8(0); i < platformCount; i++ {
			platform := string(rune('A' + i))
			platforms[i] = platform
			
			// Register a mock factory
			registry.Register(platform, func(config ConnectorConfig) (PlatformConnector, error) {
				return &mockConnector{platform: platform}, nil
			})
			
			// Set config
			registry.SetConfig(platform, ConnectorConfig{
				Platform: platform,
				Enabled:  true,
			})
		}
		
		// Create connectors for all platforms
		initialConnectors := make(map[string]PlatformConnector)
		for _, platform := range platforms {
			connector, err := registry.CreateConnector(platform)
			if err != nil {
				t.Logf("Failed to create connector for %s: %v", platform, err)
				return false
			}
			initialConnectors[platform] = connector
		}
		
		// Remove one platform and verify others still work
		if len(platforms) > 1 {
			removedPlatform := platforms[0]
			registry.SetConfig(removedPlatform, ConnectorConfig{
				Platform: removedPlatform,
				Enabled:  false,
			})
			
			// Verify other platforms still work
			for _, platform := range platforms[1:] {
				_, err := registry.CreateConnector(platform)
				if err != nil {
					t.Logf("Connector isolation violated: %s failed after removing %s", platform, removedPlatform)
					return false
				}
			}
		}
		
		return true
	}
	
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// TestProperty20_ConfigurationDrivenConnectorManagement tests that connectors
// can be enabled/disabled through configuration
// **Feature: mcp-context-engine, Property 20: Configuration-Driven Connector Management**
// **Validates: Requirements 9.1, 9.2**
func TestProperty20_ConfigurationDrivenConnectorManagement(t *testing.T) {
	property := func(enabled bool) bool {
		registry := NewRegistry()
		platform := "test-platform"
		
		// Register mock connector
		registry.Register(platform, func(config ConnectorConfig) (PlatformConnector, error) {
			return &mockConnector{platform: platform}, nil
		})
		
		// Set configuration with enabled flag
		registry.SetConfig(platform, ConnectorConfig{
			Platform: platform,
			Enabled:  enabled,
		})
		
		// Try to create connector
		connector, err := registry.CreateConnector(platform)
		
		if enabled {
			// Should succeed when enabled
			if err != nil {
				t.Logf("Expected success when enabled, got error: %v", err)
				return false
			}
			if connector == nil {
				t.Logf("Expected connector when enabled, got nil")
				return false
			}
		} else {
			// Should fail when disabled
			if err == nil {
				t.Logf("Expected error when disabled, got success")
				return false
			}
			if connector != nil {
				t.Logf("Expected nil connector when disabled, got: %v", connector)
				return false
			}
		}
		
		return true
	}
	
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// mockConnector is a simple mock implementation for testing
type mockConnector struct {
	platform string
}

func (m *mockConnector) Authenticate(ctx context.Context, config AuthConfig) (*AuthResult, error) {
	return &AuthResult{
		AccessToken: "mock-token",
		UserID:      "mock-user",
		UserLogin:   "mock-login",
		ExpiresAt:   time.Now().Add(time.Hour),
	}, nil
}

func (m *mockConnector) FetchEvents(ctx context.Context, since time.Time, limit int) ([]PlatformEvent, error) {
	return []PlatformEvent{
		{
			ID:        "mock-event-1",
			Type:      EventTypeMessage,
			Timestamp: time.Now(),
			Author:    "mock-author",
			Content:   "mock content",
			Platform:  m.platform,
		},
	}, nil
}

func (m *mockConnector) NormalizeData(ctx context.Context, events []PlatformEvent) ([]NormalizedEvent, error) {
	normalizer := NewEventNormalizer(m.platform)
	normalized := make([]NormalizedEvent, len(events))
	for i, event := range events {
		normalized[i] = normalizer.NormalizeEvent(event)
	}
	return normalized, nil
}

func (m *mockConnector) ScheduleSync(ctx context.Context, lastSync time.Time) (time.Duration, error) {
	return 5 * time.Minute, nil
}

func (m *mockConnector) GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		Name:        m.platform,
		DisplayName: "Mock Platform",
		Version:     "1.0.0",
		Description: "Mock connector for testing",
		SupportedEvents: []EventType{EventTypeMessage},
		AuthType:    "oauth2",
	}
}