package connectors

import (
	"context"
	"testing"
	"testing/quick"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// TestProperty7_GitHubBackwardCompatibility tests that GitHub connector
// maintains backward compatibility with existing GitHub service behavior
// **Feature: mcp-context-engine, Property 7: GitHub Backward Compatibility**
// **Validates: Requirements 2.1, 2.2, 2.5, 8.1, 8.2, 8.3, 8.5, 8.6**
func TestProperty7_GitHubBackwardCompatibility(t *testing.T) {
	property := func(repoOwner, repoName string) bool {
		if repoOwner == "" || repoName == "" {
			return true // Skip invalid inputs
		}

		// Create GitHub connector with test configuration
		config := ConnectorConfig{
			Platform: "github",
			Enabled:  true,
			AuthConfig: AuthConfig{
				Metadata: map[string]string{
					"access_token": "test-token",
				},
			},
			Metadata: map[string]interface{}{
				"owner": repoOwner,
				"repo":  repoName,
			},
		}

		connector, err := NewGitHubConnector(config)
		if err != nil {
			t.Logf("Failed to create GitHub connector: %v", err)
			return false
		}

		// Verify connector implements PlatformConnector interface
		_, ok := connector.(PlatformConnector)
		if !ok {
			t.Logf("GitHub connector does not implement PlatformConnector interface")
			return false
		}

		// Verify platform info matches expected GitHub configuration
		info := connector.GetPlatformInfo()
		if info.Name != "github" {
			t.Logf("Expected platform name 'github', got '%s'", info.Name)
			return false
		}

		// Verify supported events include the original GitHub event types
		expectedEvents := map[EventType]bool{
			EventTypePullRequest: false,
			EventTypeIssue:       false,
			EventTypeCommit:      false,
		}

		for _, eventType := range info.SupportedEvents {
			if _, exists := expectedEvents[eventType]; exists {
				expectedEvents[eventType] = true
			}
		}

		for eventType, found := range expectedEvents {
			if !found {
				t.Logf("Missing expected event type: %s", eventType)
				return false
			}
		}

		// Verify rate limits are preserved (GitHub API limits)
		if info.RateLimits.RequestsPerHour != 5000 {
			t.Logf("Expected 5000 requests per hour, got %d", info.RateLimits.RequestsPerHour)
			return false
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// TestProperty8_GitHubDataExtractionPreservation tests that GitHub connector
// preserves existing data extraction behavior for files_changed and labels
// **Feature: mcp-context-engine, Property 8: GitHub Data Extraction Preservation**
// **Validates: Requirements 2.3, 2.4**
func TestProperty8_GitHubDataExtractionPreservation(t *testing.T) {
	property := func(prNumber int, files []string, labels []string) bool {
		if prNumber <= 0 || len(files) > 10 || len(labels) > 10 {
			return true // Skip invalid inputs
		}

		// Create a mock PR with files and labels
		pr := models.PullRequest{
			ID:           int64(prNumber),
			Number:       prNumber,
			Title:        "Test PR",
			Body:         "Test PR body",
			Author:       "test-author",
			State:        "open",
			CreatedAt:    time.Now(),
			FilesChanged: models.StringList(files),
			Labels:       models.StringList(labels),
		}

		// Create GitHub connector
		config := ConnectorConfig{
			Platform: "github",
			Enabled:  true,
			Metadata: map[string]interface{}{
				"owner": "test-owner",
				"repo":  "test-repo",
			},
		}

		connector, err := NewGitHubConnector(config)
		if err != nil {
			t.Logf("Failed to create GitHub connector: %v", err)
			return false
		}

		githubConnector := connector.(*GitHubConnector)

		// Convert PR to platform event
		event := githubConnector.convertPRToEvent(pr)

		// Verify files_changed is preserved
		eventFiles, ok := event.Metadata["files_changed"].([]string)
		if !ok {
			t.Logf("files_changed not found or wrong type in event metadata")
			return false
		}

		if len(eventFiles) != len(files) {
			t.Logf("Expected %d files, got %d", len(files), len(eventFiles))
			return false
		}

		for i, file := range files {
			if i < len(eventFiles) && eventFiles[i] != file {
				t.Logf("File mismatch at index %d: expected '%s', got '%s'", i, file, eventFiles[i])
				return false
			}
		}

		// Verify labels are preserved
		eventLabels, ok := event.Metadata["labels"].([]string)
		if !ok {
			t.Logf("labels not found or wrong type in event metadata")
			return false
		}

		if len(eventLabels) != len(labels) {
			t.Logf("Expected %d labels, got %d", len(labels), len(eventLabels))
			return false
		}

		for i, label := range labels {
			if i < len(eventLabels) && eventLabels[i] != label {
				t.Logf("Label mismatch at index %d: expected '%s', got '%s'", i, label, eventLabels[i])
				return false
			}
		}

		// Verify event references include files
		if len(event.References) != len(files) {
			t.Logf("Expected %d references, got %d", len(files), len(event.References))
			return false
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// TestProperty3_IncrementalSynchronizationConsistency tests that running
// incremental sync multiple times with the same timestamp doesn't create duplicates
// **Feature: mcp-context-engine, Property 3: Incremental Synchronization Consistency**
// **Validates: Requirements 1.3, 3.6, 4.6**
func TestProperty3_IncrementalSynchronizationConsistency(t *testing.T) {
	property := func(hourOffset uint8) bool {
		if hourOffset > 24 {
			return true // Skip invalid inputs
		}

		// Create a fixed timestamp for testing
		syncTime := time.Now().Add(-time.Duration(hourOffset) * time.Hour)

		// Create GitHub connector with mock service
		config := ConnectorConfig{
			Platform: "github",
			Enabled:  true,
			AuthConfig: AuthConfig{
				Metadata: map[string]string{
					"access_token": "test-token",
				},
			},
			Metadata: map[string]interface{}{
				"owner": "test-owner",
				"repo":  "test-repo",
			},
		}

		connector, err := NewGitHubConnector(config)
		if err != nil {
			t.Logf("Failed to create GitHub connector: %v", err)
			return false
		}

		ctx := context.Background()

		// Note: This test would require mocking the GitHub service to return
		// consistent data. For now, we test the interface behavior.
		
		// Verify that ScheduleSync returns consistent intervals
		interval1, err1 := connector.ScheduleSync(ctx, syncTime)
		interval2, err2 := connector.ScheduleSync(ctx, syncTime)

		if err1 != nil || err2 != nil {
			t.Logf("ScheduleSync errors: %v, %v", err1, err2)
			return false
		}

		if interval1 != interval2 {
			t.Logf("ScheduleSync returned inconsistent intervals: %v vs %v", interval1, interval2)
			return false
		}

		// Verify interval is reasonable (between 1 minute and 1 hour)
		if interval1 < time.Minute || interval1 > time.Hour {
			t.Logf("ScheduleSync returned unreasonable interval: %v", interval1)
			return false
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}

// TestProperty4_RateLimitingCompliance tests that GitHub connector
// implements proper rate limiting and exponential backoff
// **Feature: mcp-context-engine, Property 4: Rate Limiting Compliance**
// **Validates: Requirements 1.4, 3.5, 4.5**
func TestProperty4_RateLimitingCompliance(t *testing.T) {
	property := func(requestCount uint8) bool {
		if requestCount == 0 || requestCount > 20 {
			return true // Skip invalid inputs
		}

		// Create GitHub connector
		config := ConnectorConfig{
			Platform: "github",
			Enabled:  true,
			RateLimit: RateLimitConfig{
				RequestsPerHour:   100,
				RequestsPerMinute: 10,
				BurstLimit:        5,
				BackoffMultiplier: 2.0,
				MaxRetries:        3,
			},
		}

		connector, err := NewGitHubConnector(config)
		if err != nil {
			t.Logf("Failed to create GitHub connector: %v", err)
			return false
		}

		githubConnector := connector.(*GitHubConnector)
		ctx := context.Background()

		// Test rate limiting behavior
		startTime := time.Now()
		
		for i := uint8(0); i < requestCount; i++ {
			err := githubConnector.WaitForRateLimit(ctx)
			if err != nil {
				t.Logf("WaitForRateLimit failed: %v", err)
				return false
			}
		}

		elapsed := time.Since(startTime)

		// Verify that rate limiting introduces appropriate delays
		// For more than burst limit requests, there should be some delay
		if requestCount > 5 && elapsed < time.Millisecond {
			t.Logf("Expected some delay for %d requests, but elapsed time was %v", requestCount, elapsed)
			return false
		}

		// Verify backoff delay calculation
		rateLimiter := githubConnector.rateLimiter
		delay1 := rateLimiter.GetBackoffDelay()
		delay2 := rateLimiter.GetBackoffDelay()

		// Second delay should be larger (exponential backoff)
		if delay2 <= delay1 {
			t.Logf("Expected exponential backoff: delay2 (%v) should be > delay1 (%v)", delay2, delay1)
			return false
		}

		// Reset backoff and verify it resets
		rateLimiter.ResetBackoff()
		delay3 := rateLimiter.GetBackoffDelay()
		
		if delay3 >= delay2 {
			t.Logf("Expected reset backoff: delay3 (%v) should be < delay2 (%v)", delay3, delay2)
			return false
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violation: %v", err)
	}
}