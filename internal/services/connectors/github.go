package connectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/DevAnuragT/context_keeper/internal/services"
)

// GitHubConnector implements PlatformConnector for GitHub integration
type GitHubConnector struct {
	*BaseConnector
	githubService services.GitHubService
	normalizer    *EventNormalizer
}

// NewGitHubConnector creates a new GitHub connector
func NewGitHubConnector(config ConnectorConfig) (PlatformConnector, error) {
	base := NewBaseConnector(config)
	githubService := services.NewGitHubService()
	normalizer := NewEventNormalizer("github")
	
	return &GitHubConnector{
		BaseConnector: base,
		githubService: githubService,
		normalizer:    normalizer,
	}, nil
}

// Authenticate handles GitHub OAuth authentication
func (gc *GitHubConnector) Authenticate(ctx context.Context, config AuthConfig) (*AuthResult, error) {
	// For GitHub, we expect the access token to be provided in the config metadata
	// The actual OAuth flow is handled by the existing auth service
	token, ok := config.Metadata["access_token"]
	if !ok {
		return nil, &ConnectorError{
			Platform:  "github",
			Code:      "missing_token",
			Message:   "GitHub access token not provided in config metadata",
			Retryable: false,
		}
	}

	// Validate the token by getting user info
	user, err := gc.githubService.GetUserInfo(ctx, token)
	if err != nil {
		return nil, &ConnectorError{
			Platform:  "github",
			Code:      "auth_failed",
			Message:   fmt.Sprintf("GitHub authentication failed: %v", err),
			Retryable: true,
		}
	}

	return &AuthResult{
		AccessToken: token,
		UserID:      user.ID,
		UserLogin:   user.Login,
		ExpiresAt:   time.Now().Add(24 * time.Hour), // GitHub tokens don't expire, but set a reasonable check interval
		Scopes:      config.Scopes,
	}, nil
}

// FetchEvents retrieves GitHub events (PRs, Issues, Commits) since the last sync
func (gc *GitHubConnector) FetchEvents(ctx context.Context, since time.Time, limit int) ([]PlatformEvent, error) {
	config := gc.GetConfig()
	token, ok := config.AuthConfig.Metadata["access_token"]
	if !ok {
		return nil, &ConnectorError{
			Platform:  "github",
			Code:      "missing_token",
			Message:   "GitHub access token not configured",
			Retryable: false,
		}
	}

	owner, ok := config.Metadata["owner"].(string)
	if !ok {
		return nil, &ConnectorError{
			Platform:  "github",
			Code:      "missing_owner",
			Message:   "GitHub repository owner not configured",
			Retryable: false,
		}
	}

	repo, ok := config.Metadata["repo"].(string)
	if !ok {
		return nil, &ConnectorError{
			Platform:  "github",
			Code:      "missing_repo",
			Message:   "GitHub repository name not configured",
			Retryable: false,
		}
	}

	// Wait for rate limit if necessary
	if err := gc.WaitForRateLimit(ctx); err != nil {
		return nil, err
	}

	var events []PlatformEvent

	// Fetch pull requests (limit: 50 as per existing system)
	prs, err := gc.githubService.GetPullRequests(ctx, token, owner, repo, min(limit/3, 50))
	if err != nil {
		return nil, gc.handleGitHubError(err)
	}

	// Convert PRs to platform events
	for _, pr := range prs {
		if pr.CreatedAt.After(since) || (pr.MergedAt != nil && pr.MergedAt.After(since)) {
			event := gc.convertPRToEvent(pr)
			events = append(events, event)
		}
	}

	// Fetch issues (limit: 50 as per existing system)
	issues, err := gc.githubService.GetIssues(ctx, token, owner, repo, min(limit/3, 50))
	if err != nil {
		return nil, gc.handleGitHubError(err)
	}

	// Convert issues to platform events
	for _, issue := range issues {
		if issue.CreatedAt.After(since) || (issue.ClosedAt != nil && issue.ClosedAt.After(since)) {
			event := gc.convertIssueToEvent(issue)
			events = append(events, event)
		}
	}

	// Fetch commits (limit: 100 as per existing system)
	commits, err := gc.githubService.GetCommits(ctx, token, owner, repo, min(limit/3, 100))
	if err != nil {
		return nil, gc.handleGitHubError(err)
	}

	// Convert commits to platform events
	for _, commit := range commits {
		if commit.CreatedAt.After(since) {
			event := gc.convertCommitToEvent(commit)
			events = append(events, event)
		}
	}

	return events, nil
}

// NormalizeData converts GitHub platform events to normalized format
func (gc *GitHubConnector) NormalizeData(ctx context.Context, events []PlatformEvent) ([]NormalizedEvent, error) {
	normalized := make([]NormalizedEvent, len(events))
	for i, event := range events {
		normalized[i] = gc.normalizer.NormalizeEvent(event)
	}
	return normalized, nil
}

// ScheduleSync determines the next sync interval for GitHub
func (gc *GitHubConnector) ScheduleSync(ctx context.Context, lastSync time.Time) (time.Duration, error) {
	// GitHub has rate limits, so we sync every 5 minutes by default
	// This can be configured via the sync config
	config := gc.GetConfig()
	if config.SyncConfig.SyncInterval > 0 {
		return config.SyncConfig.SyncInterval, nil
	}
	return 5 * time.Minute, nil
}

// GetPlatformInfo returns GitHub connector metadata
func (gc *GitHubConnector) GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		Name:        "github",
		DisplayName: "GitHub",
		Version:     "1.0.0",
		Description: "GitHub repository connector for PRs, Issues, and Commits",
		SupportedEvents: []EventType{
			EventTypePullRequest,
			EventTypeIssue,
			EventTypeCommit,
		},
		RateLimits: RateLimitInfo{
			RequestsPerHour:   5000, // GitHub API limit
			RequestsPerMinute: 100,
			BurstLimit:        10,
			BackoffStrategy:   "exponential",
			RetryAfterHeader:  "X-RateLimit-Reset",
		},
		AuthType:       "oauth2",
		RequiredScopes: []string{"repo", "read:user"},
	}
}

// convertPRToEvent converts a GitHub PR to a platform event
func (gc *GitHubConnector) convertPRToEvent(pr models.PullRequest) PlatformEvent {
	labels := make([]string, len(pr.Labels))
	for i, label := range pr.Labels {
		labels[i] = label
	}

	files := make([]string, len(pr.FilesChanged))
	for i, file := range pr.FilesChanged {
		files[i] = file
	}

	return PlatformEvent{
		ID:        fmt.Sprintf("pr-%d", pr.ID),
		Type:      EventTypePullRequest,
		Timestamp: pr.CreatedAt,
		Author:    pr.Author,
		Content:   pr.Body,
		Title:     pr.Title,
		Platform:  "github",
		Metadata: map[string]interface{}{
			"number":        pr.Number,
			"state":         pr.State,
			"merged_at":     pr.MergedAt,
			"labels":        labels,
			"files_changed": files,
		},
		References: files,
	}
}

// convertIssueToEvent converts a GitHub issue to a platform event
func (gc *GitHubConnector) convertIssueToEvent(issue models.Issue) PlatformEvent {
	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label
	}

	return PlatformEvent{
		ID:        fmt.Sprintf("issue-%d", issue.ID),
		Type:      EventTypeIssue,
		Timestamp: issue.CreatedAt,
		Author:    issue.Author,
		Content:   issue.Body,
		Title:     issue.Title,
		Platform:  "github",
		Metadata: map[string]interface{}{
			"state":     issue.State,
			"closed_at": issue.ClosedAt,
			"labels":    labels,
		},
	}
}

// convertCommitToEvent converts a GitHub commit to a platform event
func (gc *GitHubConnector) convertCommitToEvent(commit models.Commit) PlatformEvent {
	files := make([]string, len(commit.FilesChanged))
	for i, file := range commit.FilesChanged {
		files[i] = file
	}

	return PlatformEvent{
		ID:        fmt.Sprintf("commit-%s", commit.SHA),
		Type:      EventTypeCommit,
		Timestamp: commit.CreatedAt,
		Author:    commit.Author,
		Content:   commit.Message,
		Platform:  "github",
		Metadata: map[string]interface{}{
			"sha":           commit.SHA,
			"files_changed": files,
		},
		References: files,
	}
}

// handleGitHubError converts GitHub service errors to connector errors
func (gc *GitHubConnector) handleGitHubError(err error) error {
	errStr := err.Error()
	
	// Check for rate limit errors
	if strings.Contains(errStr, "rate limit") {
		retryAfter := 1 * time.Hour // Default GitHub rate limit reset
		return &ConnectorError{
			Platform:   "github",
			Code:       "rate_limit",
			Message:    "GitHub API rate limit exceeded",
			Retryable:  true,
			RetryAfter: &retryAfter,
		}
	}

	// Check for authentication errors
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		return &ConnectorError{
			Platform:  "github",
			Code:      "auth_error",
			Message:   "GitHub authentication failed",
			Retryable: false,
		}
	}

	// Check for network errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") {
		return &ConnectorError{
			Platform:  "github",
			Code:      "network_error",
			Message:   fmt.Sprintf("GitHub API network error: %v", err),
			Retryable: true,
		}
	}

	// Generic error
	return &ConnectorError{
		Platform:  "github",
		Code:      "api_error",
		Message:   fmt.Sprintf("GitHub API error: %v", err),
		Retryable: true,
	}
}