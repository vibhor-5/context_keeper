package connectors

import (
	"context"
	"time"
)

// PlatformConnector defines the common interface for all platform integrations
type PlatformConnector interface {
	// Authenticate handles platform-specific OAuth flows and token management
	Authenticate(ctx context.Context, config AuthConfig) (*AuthResult, error)
	
	// FetchEvents retrieves new events since last sync with pagination support
	FetchEvents(ctx context.Context, since time.Time, limit int) ([]PlatformEvent, error)
	
	// NormalizeData converts platform-specific events to common format
	NormalizeData(ctx context.Context, events []PlatformEvent) ([]NormalizedEvent, error)
	
	// ScheduleSync determines next sync interval based on platform limits
	ScheduleSync(ctx context.Context, lastSync time.Time) (time.Duration, error)
	
	// GetPlatformInfo returns connector metadata and capabilities
	GetPlatformInfo() PlatformInfo
}

// AuthConfig contains platform-specific authentication configuration
type AuthConfig struct {
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	RedirectURL  string            `json:"redirect_url"`
	Scopes       []string          `json:"scopes"`
	Metadata     map[string]string `json:"metadata"`
}

// AuthResult contains the result of platform authentication
type AuthResult struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id"`
	UserLogin    string    `json:"user_login"`
	Scopes       []string  `json:"scopes"`
}

// EventType represents the type of platform event
type EventType string

const (
	EventTypePullRequest     EventType = "pull_request"
	EventTypeIssue          EventType = "issue"
	EventTypeCommit         EventType = "commit"
	EventTypeMessage        EventType = "message"
	EventTypeThread         EventType = "thread"
	EventTypeReaction       EventType = "reaction"
	EventTypeFileChange     EventType = "file_change"
	EventTypeDiscussion     EventType = "discussion"
)

// PlatformEvent represents a raw event from a platform
type PlatformEvent struct {
	ID          string                 `json:"id"`
	Type        EventType             `json:"type"`
	Timestamp   time.Time             `json:"timestamp"`
	Author      string                `json:"author"`
	Content     string                `json:"content"`
	Title       string                `json:"title,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	References  []string              `json:"references"`
	Platform    string                `json:"platform"`
}

// NormalizedEvent represents a platform event in common format
type NormalizedEvent struct {
	PlatformID   string                 `json:"platform_id"`
	EventType    EventType             `json:"event_type"`
	Timestamp    time.Time             `json:"timestamp"`
	Author       string                `json:"author"`
	Content      string                `json:"content"`
	Title        string                `json:"title,omitempty"`
	ThreadID     *string               `json:"thread_id,omitempty"`
	ParentID     *string               `json:"parent_id,omitempty"`
	FileRefs     []string              `json:"file_refs"`
	FeatureRefs  []string              `json:"feature_refs"`
	Labels       []string              `json:"labels,omitempty"`
	State        string                `json:"state,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	Platform     string                `json:"platform"`
}

// PlatformInfo contains metadata about a platform connector
type PlatformInfo struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	SupportedEvents []EventType `json:"supported_events"`
	RateLimits   RateLimitInfo `json:"rate_limits"`
	AuthType     string   `json:"auth_type"`
	RequiredScopes []string `json:"required_scopes"`
}

// RateLimitInfo contains rate limiting information for a platform
type RateLimitInfo struct {
	RequestsPerHour   int           `json:"requests_per_hour"`
	RequestsPerMinute int           `json:"requests_per_minute"`
	BurstLimit        int           `json:"burst_limit"`
	BackoffStrategy   string        `json:"backoff_strategy"`
	RetryAfterHeader  string        `json:"retry_after_header,omitempty"`
}

// ConnectorError represents errors from platform connectors
type ConnectorError struct {
	Platform    string `json:"platform"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Retryable   bool   `json:"retryable"`
	RetryAfter  *time.Duration `json:"retry_after,omitempty"`
}

func (e *ConnectorError) Error() string {
	return e.Message
}

// IsRetryable returns whether the error is retryable
func (e *ConnectorError) IsRetryable() bool {
	return e.Retryable
}

// GetRetryAfter returns the suggested retry delay
func (e *ConnectorError) GetRetryAfter() time.Duration {
	if e.RetryAfter != nil {
		return *e.RetryAfter
	}
	return 0
}