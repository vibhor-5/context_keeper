package connectors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// SlackConnector implements PlatformConnector for Slack integration
type SlackConnector struct {
	*BaseConnector
	client     *slack.Client
	normalizer *EventNormalizer
}

// NewSlackConnector creates a new Slack connector
func NewSlackConnector(config ConnectorConfig) (PlatformConnector, error) {
	base := NewBaseConnector(config)
	
	// Get bot token from config
	botToken, ok := config.AuthConfig.Metadata["bot_token"]
	if !ok {
		return nil, &ConnectorError{
			Platform:  "slack",
			Code:      "missing_bot_token",
			Message:   "Slack bot token not provided in config metadata",
			Retryable: false,
		}
	}

	client := slack.New(botToken)
	normalizer := NewEventNormalizer("slack")
	
	return &SlackConnector{
		BaseConnector: base,
		client:        client,
		normalizer:    normalizer,
	}, nil
}

// Authenticate handles Slack OAuth authentication
func (sc *SlackConnector) Authenticate(ctx context.Context, config AuthConfig) (*AuthResult, error) {
	// For Slack, we expect the bot token to be provided in the config metadata
	botToken, ok := config.Metadata["bot_token"]
	if !ok {
		return nil, &ConnectorError{
			Platform:  "slack",
			Code:      "missing_bot_token",
			Message:   "Slack bot token not provided in config metadata",
			Retryable: false,
		}
	}

	// Validate the token by testing auth
	client := slack.New(botToken)
	authTest, err := client.AuthTestContext(ctx)
	if err != nil {
		return nil, &ConnectorError{
			Platform:  "slack",
			Code:      "auth_failed",
			Message:   fmt.Sprintf("Slack authentication failed: %v", err),
			Retryable: true,
		}
	}

	return &AuthResult{
		AccessToken: botToken,
		UserID:      authTest.UserID,
		UserLogin:   authTest.User,
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Slack tokens don't expire, but set a reasonable check interval
		Scopes:      config.Scopes,
	}, nil
}

// FetchEvents retrieves Slack events (messages, threads) since the last sync
func (sc *SlackConnector) FetchEvents(ctx context.Context, since time.Time, limit int) ([]PlatformEvent, error) {
	config := sc.GetConfig()
	
	// Wait for rate limit if necessary
	if err := sc.WaitForRateLimit(ctx); err != nil {
		return nil, err
	}

	var events []PlatformEvent

	// Get channels to monitor
	channels, ok := config.Metadata["channels"].([]string)
	if !ok {
		// Try to convert from interface{} slice
		if channelsInterface, exists := config.Metadata["channels"].([]interface{}); exists {
			channels = make([]string, len(channelsInterface))
			for i, ch := range channelsInterface {
				if chStr, ok := ch.(string); ok {
					channels[i] = chStr
				}
			}
		}
	}

	if len(channels) == 0 {
		return events, nil // No channels configured
	}

	// Fetch messages from each channel
	for _, channelID := range channels {
		if channelID == "" {
			continue
		}

		channelEvents, err := sc.fetchChannelEvents(ctx, channelID, since, limit/len(channels))
		if err != nil {
			// Log error but continue with other channels
			continue
		}
		events = append(events, channelEvents...)
	}

	// Include DMs if configured
	includeDMs, ok := config.Metadata["include_dms"].(bool)
	if ok && includeDMs {
		dmEvents, err := sc.fetchDirectMessages(ctx, since, limit/4)
		if err == nil {
			events = append(events, dmEvents...)
		}
	}

	return events, nil
}

// fetchChannelEvents retrieves messages from a specific Slack channel
func (sc *SlackConnector) fetchChannelEvents(ctx context.Context, channelID string, since time.Time, limit int) ([]PlatformEvent, error) {
	var events []PlatformEvent

	// Convert timestamp to Slack format
	oldest := fmt.Sprintf("%.6f", float64(since.Unix()))

	// Get conversation history
	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Oldest:    oldest,
		Limit:     limit,
	}

	history, err := sc.client.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return nil, sc.handleSlackError(err)
	}

	// Convert messages to platform events
	for _, msg := range history.Messages {
		if msg.Timestamp == "" {
			continue
		}

		event := sc.convertMessageToEvent(msg, channelID)
		events = append(events, event)

		// Fetch thread replies if message has replies
		if msg.ReplyCount > 0 {
			threadEvents, err := sc.fetchThreadReplies(ctx, channelID, msg.Timestamp)
			if err == nil {
				events = append(events, threadEvents...)
			}
		}
	}

	return events, nil
}

// fetchThreadReplies retrieves replies to a threaded message
func (sc *SlackConnector) fetchThreadReplies(ctx context.Context, channelID, threadTS string) ([]PlatformEvent, error) {
	var events []PlatformEvent

	// Get thread depth from config
	config := sc.GetConfig()
	threadDepth, ok := config.Metadata["thread_depth"].(int)
	if !ok {
		threadDepth = 10 // Default thread depth
	}

	params := &slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
		Limit:     threadDepth,
	}

	replies, _, _, err := sc.client.GetConversationRepliesContext(ctx, params)
	if err != nil {
		return nil, sc.handleSlackError(err)
	}

	// Convert replies to platform events (skip the parent message)
	for i, msg := range replies {
		if i == 0 {
			continue // Skip parent message
		}

		event := sc.convertMessageToEvent(msg, channelID)
		event.Metadata["thread_ts"] = threadTS
		event.Metadata["parent_id"] = threadTS
		events = append(events, event)
	}

	return events, nil
}

// fetchDirectMessages retrieves direct messages
func (sc *SlackConnector) fetchDirectMessages(ctx context.Context, since time.Time, limit int) ([]PlatformEvent, error) {
	var events []PlatformEvent

	// Get list of DM conversations
	params := &slack.GetConversationsParameters{
		Types: []string{"im"},
		Limit: 100,
	}

	conversations, _, err := sc.client.GetConversationsContext(ctx, params)
	if err != nil {
		return nil, sc.handleSlackError(err)
	}

	// Fetch messages from each DM conversation
	for _, conv := range conversations {
		dmEvents, err := sc.fetchChannelEvents(ctx, conv.ID, since, limit/len(conversations))
		if err == nil {
			events = append(events, dmEvents...)
		}
	}

	return events, nil
}

// convertMessageToEvent converts a Slack message to a platform event
func (sc *SlackConnector) convertMessageToEvent(msg slack.Message, channelID string) PlatformEvent {
	// Parse timestamp
	timestamp, _ := strconv.ParseFloat(msg.Timestamp, 64)
	eventTime := time.Unix(int64(timestamp), 0)

	// Extract file references from message text
	fileRefs := sc.extractFileReferences(msg.Text)

	return PlatformEvent{
		ID:        fmt.Sprintf("msg-%s", msg.Timestamp),
		Type:      EventTypeMessage,
		Timestamp: eventTime,
		Author:    msg.User,
		Content:   msg.Text,
		Platform:  "slack",
		Metadata: map[string]interface{}{
			"channel_id":   channelID,
			"timestamp":    msg.Timestamp,
			"thread_ts":    msg.ThreadTimestamp,
			"reply_count":  msg.ReplyCount,
			"subtype":      msg.SubType,
			"bot_id":       msg.BotID,
			"attachments":  len(msg.Attachments),
			"reactions":    len(msg.Reactions),
		},
		References: fileRefs,
	}
}

// extractFileReferences extracts file references from Slack message text
func (sc *SlackConnector) extractFileReferences(text string) []string {
	var fileRefs []string

	// Look for common file patterns in Slack messages
	// This is a simple implementation - could be enhanced with regex
	words := strings.Fields(text)
	for _, word := range words {
		// Check for file extensions
		if strings.Contains(word, ".") {
			parts := strings.Split(word, ".")
			if len(parts) > 1 {
				ext := strings.ToLower(parts[len(parts)-1])
				// Common code file extensions
				if isCodeFileExtension(ext) {
					fileRefs = append(fileRefs, word)
				}
			}
		}
	}

	return fileRefs
}

// NormalizeData converts Slack platform events to normalized format
func (sc *SlackConnector) NormalizeData(ctx context.Context, events []PlatformEvent) ([]NormalizedEvent, error) {
	normalized := make([]NormalizedEvent, len(events))
	for i, event := range events {
		normalizedEvent := sc.normalizer.NormalizeEvent(event)
		
		// Add Slack-specific normalization
		if threadTS, ok := event.Metadata["thread_ts"].(string); ok && threadTS != "" {
			normalizedEvent.ThreadID = &threadTS
		}
		
		if parentID, ok := event.Metadata["parent_id"].(string); ok && parentID != "" {
			normalizedEvent.ParentID = &parentID
		}

		normalized[i] = normalizedEvent
	}
	return normalized, nil
}

// ScheduleSync determines the next sync interval for Slack
func (sc *SlackConnector) ScheduleSync(ctx context.Context, lastSync time.Time) (time.Duration, error) {
	// Slack has rate limits, so we sync every 2 minutes by default
	config := sc.GetConfig()
	if config.SyncConfig.SyncInterval > 0 {
		return config.SyncConfig.SyncInterval, nil
	}
	return 2 * time.Minute, nil
}

// GetPlatformInfo returns Slack connector metadata
func (sc *SlackConnector) GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		Name:        "slack",
		DisplayName: "Slack",
		Version:     "1.0.0",
		Description: "Slack workspace connector for messages and threads",
		SupportedEvents: []EventType{
			EventTypeMessage,
			EventTypeThread,
			EventTypeReaction,
		},
		RateLimits: RateLimitInfo{
			RequestsPerHour:   1000, // Slack API limit
			RequestsPerMinute: 50,
			BurstLimit:        5,
			BackoffStrategy:   "exponential",
			RetryAfterHeader:  "Retry-After",
		},
		AuthType:       "bot_token",
		RequiredScopes: []string{"channels:history", "groups:history", "im:history", "mpim:history"},
	}
}

// handleSlackError converts Slack API errors to connector errors
func (sc *SlackConnector) handleSlackError(err error) error {
	errStr := err.Error()
	
	// Check for rate limit errors
	if strings.Contains(errStr, "rate_limited") {
		retryAfter := 1 * time.Minute // Default Slack rate limit
		return &ConnectorError{
			Platform:   "slack",
			Code:       "rate_limit",
			Message:    "Slack API rate limit exceeded",
			Retryable:  true,
			RetryAfter: &retryAfter,
		}
	}

	// Check for authentication errors
	if strings.Contains(errStr, "invalid_auth") || strings.Contains(errStr, "token_revoked") {
		return &ConnectorError{
			Platform:  "slack",
			Code:      "auth_error",
			Message:   "Slack authentication failed",
			Retryable: false,
		}
	}

	// Check for permission errors
	if strings.Contains(errStr, "missing_scope") || strings.Contains(errStr, "not_in_channel") {
		return &ConnectorError{
			Platform:  "slack",
			Code:      "permission_error",
			Message:   fmt.Sprintf("Slack permission error: %v", err),
			Retryable: false,
		}
	}

	// Check for network errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") {
		return &ConnectorError{
			Platform:  "slack",
			Code:      "network_error",
			Message:   fmt.Sprintf("Slack API network error: %v", err),
			Retryable: true,
		}
	}

	// Generic error
	return &ConnectorError{
		Platform:  "slack",
		Code:      "api_error",
		Message:   fmt.Sprintf("Slack API error: %v", err),
		Retryable: true,
	}
}