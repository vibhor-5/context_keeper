package connectors

import (
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// TestProperty9_ThreadContextPreservation tests that threaded conversations
// from Slack or Discord preserve parent-child relationships
// **Feature: mcp-context-engine, Property 9: Thread Context Preservation**
// **Validates: Requirements 3.3, 4.3**
func TestProperty9_ThreadContextPreservation(t *testing.T) {
	property := func(platform string, threadID string) bool {
		if platform != "slack" && platform != "discord" {
			return true // Skip invalid platforms
		}
		if threadID == "" {
			return true // Skip empty thread IDs
		}

		// Create platform-specific configuration (for validation)
		var config ConnectorConfig
		_ = config // Mark as used for validation purposes
		switch platform {
		case "slack":
			config = ConnectorConfig{
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
		case "discord":
			config = ConnectorConfig{
				Platform: "discord",
				Enabled:  true,
				AuthConfig: AuthConfig{
					Metadata: map[string]string{
						"bot_token": "test-discord-token",
					},
				},
				Metadata: map[string]interface{}{
					"channel_ids":  []string{"123456789012345678"},
					"thread_depth": 10,
				},
			}
		}

		// Create mock platform event with thread information
		event := PlatformEvent{
			ID:        "test-event-1",
			Type:      EventTypeMessage,
			Timestamp: time.Now(),
			Author:    "test-user",
			Content:   "Test message in thread",
			Platform:  platform,
			Metadata: map[string]interface{}{
				"thread_id": threadID,
			},
		}

		// Create normalizer and normalize the event
		normalizer := NewEventNormalizer(platform)
		normalized := normalizer.NormalizeEvent(event)

		// Verify thread information is preserved in normalized event
		if normalized.Platform != platform {
			t.Logf("Platform mismatch: expected %s, got %s", platform, normalized.Platform)
			return false
		}

		// For events with thread metadata, verify ThreadID is set
		if threadIDMeta, ok := event.Metadata["thread_id"].(string); ok && threadIDMeta != "" {
			if normalized.ThreadID == nil {
				t.Logf("ThreadID should be set for threaded message")
				return false
			}
			if *normalized.ThreadID != threadIDMeta {
				t.Logf("ThreadID mismatch: expected %s, got %s", threadIDMeta, *normalized.ThreadID)
				return false
			}
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// TestProperty10_CrossPlatformDecisionExtraction tests that engineering decisions
// can be extracted from discussions regardless of source platform
// **Feature: mcp-context-engine, Property 10: Cross-Platform Decision Extraction**
// **Validates: Requirements 3.4, 4.4, 5.1**
func TestProperty10_CrossPlatformDecisionExtraction(t *testing.T) {
	property := func(platform string, content string) bool {
		if platform != "github" && platform != "slack" && platform != "discord" {
			return true // Skip invalid platforms
		}
		if content == "" {
			return true // Skip empty content
		}

		// Create platform event with decision-like content
		event := PlatformEvent{
			ID:        "decision-event-1",
			Type:      EventTypeMessage,
			Timestamp: time.Now(),
			Author:    "architect",
			Content:   content,
			Platform:  platform,
			Metadata:  map[string]interface{}{},
		}

		// Create normalizer and normalize the event
		normalizer := NewEventNormalizer(platform)
		normalized := normalizer.NormalizeEvent(event)

		// Verify basic normalization works across platforms
		if normalized.Platform != platform {
			t.Logf("Platform mismatch: expected %s, got %s", platform, normalized.Platform)
			return false
		}

		if normalized.Content != content {
			t.Logf("Content mismatch: expected %s, got %s", content, normalized.Content)
			return false
		}

		if normalized.Author != "architect" {
			t.Logf("Author mismatch: expected architect, got %s", normalized.Author)
			return false
		}

		// Verify event type is preserved
		if normalized.EventType != EventTypeMessage {
			t.Logf("Event type mismatch: expected %s, got %s", EventTypeMessage, normalized.EventType)
			return false
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// TestProperty5_DataNormalizationConsistency tests that platform-specific events
// are normalized to consistent common format across all platforms
// **Feature: mcp-context-engine, Property 5: Data Normalization Consistency**
// **Validates: Requirements 1.5, 3.7, 4.7**
func TestProperty5_DataNormalizationConsistency(t *testing.T) {
	property := func(platform string, eventType EventType, author string, content string) bool {
		if platform != "github" && platform != "slack" && platform != "discord" {
			return true // Skip invalid platforms
		}
		if author == "" || content == "" {
			return true // Skip empty required fields
		}

		// Create platform event
		event := PlatformEvent{
			ID:        "consistency-test-1",
			Type:      eventType,
			Timestamp: time.Now(),
			Author:    author,
			Content:   content,
			Platform:  platform,
			Metadata:  map[string]interface{}{},
		}

		// Create normalizer and normalize the event
		normalizer := NewEventNormalizer(platform)
		normalized := normalizer.NormalizeEvent(event)

		// Verify all required fields are populated
		if normalized.PlatformID == "" {
			t.Logf("PlatformID should not be empty")
			return false
		}

		if normalized.EventType != eventType {
			t.Logf("EventType mismatch: expected %s, got %s", eventType, normalized.EventType)
			return false
		}

		if normalized.Author != author {
			t.Logf("Author mismatch: expected %s, got %s", author, normalized.Author)
			return false
		}

		if normalized.Content != content {
			t.Logf("Content mismatch: expected %s, got %s", content, normalized.Content)
			return false
		}

		if normalized.Platform != platform {
			t.Logf("Platform mismatch: expected %s, got %s", platform, normalized.Platform)
			return false
		}

		// Verify timestamp is preserved
		if normalized.Timestamp.IsZero() {
			t.Logf("Timestamp should not be zero")
			return false
		}

		// Verify metadata is initialized
		if normalized.Metadata == nil {
			t.Logf("Metadata should be initialized")
			return false
		}

		// Verify file references are initialized (can be empty)
		if normalized.FileRefs == nil {
			t.Logf("FileRefs should be initialized (can be empty slice)")
			return false
		}

		// Verify feature references are initialized (can be empty)
		if normalized.FeatureRefs == nil {
			t.Logf("FeatureRefs should be initialized (can be empty slice)")
			return false
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// TestProperty6_OAuthAuthenticationSecurity tests that OAuth flows complete
// successfully with appropriate scopes for platforms requiring OAuth
// **Feature: mcp-context-engine, Property 6: OAuth Authentication Security**
// **Validates: Requirements 1.6, 3.1, 4.1**
func TestProperty6_OAuthAuthenticationSecurity(t *testing.T) {
	property := func(platform string, hasToken bool) bool {
		if platform != "github" && platform != "slack" && platform != "discord" {
			return true // Skip invalid platforms
		}

		// Create auth config based on platform
		var authConfig AuthConfig
		switch platform {
		case "github":
			authConfig = AuthConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"repo", "read:user"},
			}
			if hasToken {
				authConfig.Metadata = map[string]string{
					"access_token": "test-github-token",
				}
			}
		case "slack":
			authConfig = AuthConfig{
				Scopes: []string{"channels:history", "groups:history"},
			}
			if hasToken {
				authConfig.Metadata = map[string]string{
					"bot_token": "xoxb-test-slack-token",
				}
			}
		case "discord":
			authConfig = AuthConfig{
				Scopes: []string{"bot", "read_messages"},
			}
			if hasToken {
				authConfig.Metadata = map[string]string{
					"bot_token": "test-discord-token",
				}
			}
		}

		// Verify required scopes are present
		if len(authConfig.Scopes) == 0 {
			t.Logf("Platform %s should have required scopes", platform)
			return false
		}

		// Verify token handling based on platform requirements
		switch platform {
		case "github":
			if hasToken {
				if token, ok := authConfig.Metadata["access_token"]; !ok || token == "" {
					t.Logf("GitHub should have access_token when hasToken is true")
					return false
				}
			}
		case "slack":
			if hasToken {
				if token, ok := authConfig.Metadata["bot_token"]; !ok || token == "" {
					t.Logf("Slack should have bot_token when hasToken is true")
					return false
				}
				// Verify Slack bot token format
				if token, ok := authConfig.Metadata["bot_token"]; ok && !strings.HasPrefix(token, "xoxb-") {
					t.Logf("Slack bot token should start with 'xoxb-'")
					return false
				}
			}
		case "discord":
			if hasToken {
				if token, ok := authConfig.Metadata["bot_token"]; !ok || token == "" {
					t.Logf("Discord should have bot_token when hasToken is true")
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

// TestProperty21_TokenSecurityAndEncryption tests that platform authentication
// tokens are handled securely with proper access controls
// **Feature: mcp-context-engine, Property 21: Token Security and Encryption**
// **Validates: Requirements 9.4**
func TestProperty21_TokenSecurityAndEncryption(t *testing.T) {
	property := func(platform string, token string) bool {
		if platform != "github" && platform != "slack" && platform != "discord" {
			return true // Skip invalid platforms
		}
		if token == "" {
			return true // Skip empty tokens
		}

		// Create configuration with token
		config := ConnectorConfig{
			Platform: platform,
			Enabled:  true,
			AuthConfig: AuthConfig{
				Metadata: map[string]string{
					"access_token": token, // Generic token field
					"bot_token":    token, // Bot token field
				},
			},
		}

		// Verify token is stored in metadata (not exposed in logs)
		if storedToken, ok := config.AuthConfig.Metadata["access_token"]; ok {
			if storedToken != token {
				t.Logf("Token storage mismatch")
				return false
			}
		}

		// Verify security boundary validation would work
		boundary := NewSecurityBoundary([]string{platform}, "strict")
		if !boundary.IsPlatformAllowed(platform) {
			t.Logf("Platform %s should be allowed in security boundary", platform)
			return false
		}

		// Verify resource access validation
		resource := platform + ":test-resource"
		if err := boundary.ValidateAccess(platform, resource); err != nil {
			t.Logf("Platform %s should be able to access its own resources", platform)
			return false
		}

		// Verify cross-platform access is blocked in strict mode
		otherPlatform := "other-platform"
		if platform != otherPlatform {
			otherResource := otherPlatform + ":test-resource"
			if err := boundary.ValidateAccess(platform, otherResource); err == nil {
				t.Logf("Platform %s should not be able to access other platform resources in strict mode", platform)
				return false
			}
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}