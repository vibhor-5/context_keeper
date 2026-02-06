package connectors

import (
	"context"
	"math"
	"sync"
	"time"
)

// BaseConnector provides common functionality for all platform connectors
type BaseConnector struct {
	config      ConnectorConfig
	rateLimiter *RateLimiter
	lastSync    time.Time
	mu          sync.RWMutex
}

// NewBaseConnector creates a new base connector
func NewBaseConnector(config ConnectorConfig) *BaseConnector {
	return &BaseConnector{
		config:      config,
		rateLimiter: NewRateLimiter(config.RateLimit),
		lastSync:    time.Time{},
	}
}

// GetConfig returns the connector configuration
func (bc *BaseConnector) GetConfig() ConnectorConfig {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.config
}

// UpdateLastSync updates the last synchronization timestamp
func (bc *BaseConnector) UpdateLastSync(timestamp time.Time) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.lastSync = timestamp
}

// GetLastSync returns the last synchronization timestamp
func (bc *BaseConnector) GetLastSync() time.Time {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.lastSync
}

// WaitForRateLimit waits for rate limit if necessary
func (bc *BaseConnector) WaitForRateLimit(ctx context.Context) error {
	return bc.rateLimiter.Wait(ctx)
}

// HandleRateLimitError handles rate limit errors with exponential backoff
func (bc *BaseConnector) HandleRateLimitError(err error) time.Duration {
	if connErr, ok := err.(*ConnectorError); ok && connErr.IsRetryable() {
		if connErr.RetryAfter != nil {
			return *connErr.RetryAfter
		}
	}
	
	// Default exponential backoff
	return bc.rateLimiter.GetBackoffDelay()
}

// RateLimiter implements rate limiting with exponential backoff
type RateLimiter struct {
	config        RateLimitConfig
	tokens        int
	lastRefill    time.Time
	backoffCount  int
	mu            sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:     config,
		tokens:     config.BurstLimit,
		lastRefill: time.Now(),
	}
}

// Wait waits for a token to become available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	
	// Calculate tokens to add based on requests per minute
	tokensToAdd := int(elapsed.Minutes() * float64(rl.config.RequestsPerMinute))
	if tokensToAdd > 0 {
		rl.tokens = min(rl.tokens+tokensToAdd, rl.config.BurstLimit)
		rl.lastRefill = now
	}
	
	// If no tokens available, wait
	if rl.tokens <= 0 {
		waitTime := time.Minute / time.Duration(rl.config.RequestsPerMinute)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			rl.tokens = 1
		}
	}
	
	// Consume a token
	rl.tokens--
	return nil
}

// GetBackoffDelay returns the current backoff delay
func (rl *RateLimiter) GetBackoffDelay() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if rl.config.BackoffMultiplier <= 0 {
		rl.config.BackoffMultiplier = 2.0
	}
	
	delay := time.Duration(math.Pow(rl.config.BackoffMultiplier, float64(rl.backoffCount))) * time.Second
	rl.backoffCount++
	
	// Cap at 5 minutes
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	
	return delay
}

// ResetBackoff resets the backoff counter
func (rl *RateLimiter) ResetBackoff() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.backoffCount = 0
}

// EventNormalizer provides common event normalization functionality
type EventNormalizer struct {
	platform string
}

// NewEventNormalizer creates a new event normalizer
func NewEventNormalizer(platform string) *EventNormalizer {
	return &EventNormalizer{
		platform: platform,
	}
}

// NormalizeEvent converts a platform event to normalized format
func (en *EventNormalizer) NormalizeEvent(event PlatformEvent) NormalizedEvent {
	// Start with file references from the event's References field
	fileRefs := make([]string, len(event.References))
	copy(fileRefs, event.References)
	
	// Add additional file references extracted from content and metadata
	extractedRefs := en.extractFileReferences(event.Content, event.Metadata)
	fileRefs = append(fileRefs, extractedRefs...)
	
	normalized := NormalizedEvent{
		PlatformID:  event.ID,
		EventType:   event.Type,
		Timestamp:   event.Timestamp,
		Author:      event.Author,
		Content:     event.Content,
		Title:       event.Title,
		FileRefs:    fileRefs,
		FeatureRefs: en.extractFeatureReferences(event.Content, event.Metadata),
		Metadata:    event.Metadata,
		Platform:    en.platform,
	}
	
	// Extract thread information if available
	if threadID, ok := event.Metadata["thread_id"].(string); ok {
		normalized.ThreadID = &threadID
	}
	
	if parentID, ok := event.Metadata["parent_id"].(string); ok {
		normalized.ParentID = &parentID
	}
	
	// Extract labels if available
	if labels, ok := event.Metadata["labels"].([]string); ok {
		normalized.Labels = labels
	}
	
	// Extract state if available
	if state, ok := event.Metadata["state"].(string); ok {
		normalized.State = state
	}
	
	return normalized
}

// extractFileReferences extracts file references from content and metadata
func (en *EventNormalizer) extractFileReferences(content string, metadata map[string]interface{}) []string {
	var fileRefs []string
	
	// Extract from metadata first
	if files, ok := metadata["files_changed"].([]string); ok {
		fileRefs = append(fileRefs, files...)
	}
	
	// TODO: Add content-based file reference extraction using regex patterns
	// This would look for patterns like:
	// - src/main.go
	// - internal/services/auth.go
	// - *.js, *.ts, *.go, etc.
	
	return fileRefs
}

// extractFeatureReferences extracts feature references from content and metadata
func (en *EventNormalizer) extractFeatureReferences(content string, metadata map[string]interface{}) []string {
	var featureRefs []string
	
	// Extract from metadata
	if features, ok := metadata["features"].([]string); ok {
		featureRefs = append(featureRefs, features...)
	}
	
	// TODO: Add content-based feature reference extraction
	// This would look for patterns like:
	// - feat: authentication
	// - feature/user-auth
	// - [FEATURE-123]
	
	return featureRefs
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isCodeFileExtension checks if the extension is a common code file extension
func isCodeFileExtension(ext string) bool {
	codeExts := map[string]bool{
		"go": true, "js": true, "ts": true, "py": true, "java": true,
		"cpp": true, "c": true, "h": true, "hpp": true, "cs": true,
		"rb": true, "php": true, "swift": true, "kt": true, "rs": true,
		"sql": true, "json": true, "yaml": true, "yml": true, "xml": true,
		"html": true, "css": true, "scss": true, "less": true,
		"md": true, "txt": true, "log": true, "config": true, "conf": true,
	}
	return codeExts[ext]
}