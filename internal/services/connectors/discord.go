package connectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// DiscordConnector implements PlatformConnector for Discord integration
type DiscordConnector struct {
	*BaseConnector
	session    *discordgo.Session
	normalizer *EventNormalizer
}

// NewDiscordConnector creates a new Discord connector
func NewDiscordConnector(config ConnectorConfig) (PlatformConnector, error) {
	base := NewBaseConnector(config)
	
	// Get bot token from config
	botToken, ok := config.AuthConfig.Metadata["bot_token"]
	if !ok {
		return nil, &ConnectorError{
			Platform:  "discord",
			Code:      "missing_bot_token",
			Message:   "Discord bot token not provided in config metadata",
			Retryable: false,
		}
	}

	// Create Discord session
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, &ConnectorError{
			Platform:  "discord",
			Code:      "session_creation_failed",
			Message:   fmt.Sprintf("Failed to create Discord session: %v", err),
			Retryable: false,
		}
	}

	normalizer := NewEventNormalizer("discord")
	
	return &DiscordConnector{
		BaseConnector: base,
		session:       session,
		normalizer:    normalizer,
	}, nil
}

// Authenticate handles Discord bot authentication
func (dc *DiscordConnector) Authenticate(ctx context.Context, config AuthConfig) (*AuthResult, error) {
	// For Discord, we expect the bot token to be provided in the config metadata
	botToken, ok := config.Metadata["bot_token"]
	if !ok {
		return nil, &ConnectorError{
			Platform:  "discord",
			Code:      "missing_bot_token",
			Message:   "Discord bot token not provided in config metadata",
			Retryable: false,
		}
	}

	// Create session and test authentication
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, &ConnectorError{
			Platform:  "discord",
			Code:      "session_creation_failed",
			Message:   fmt.Sprintf("Failed to create Discord session: %v", err),
			Retryable: false,
		}
	}

	// Test authentication by getting user info
	user, err := session.User("@me")
	if err != nil {
		return nil, &ConnectorError{
			Platform:  "discord",
			Code:      "auth_failed",
			Message:   fmt.Sprintf("Discord authentication failed: %v", err),
			Retryable: true,
		}
	}

	return &AuthResult{
		AccessToken: botToken,
		UserID:      user.ID,
		UserLogin:   user.Username,
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Discord tokens don't expire, but set a reasonable check interval
		Scopes:      config.Scopes,
	}, nil
}

// FetchEvents retrieves Discord events (messages, threads) since the last sync
func (dc *DiscordConnector) FetchEvents(ctx context.Context, since time.Time, limit int) ([]PlatformEvent, error) {
	config := dc.GetConfig()
	
	// Wait for rate limit if necessary
	if err := dc.WaitForRateLimit(ctx); err != nil {
		return nil, err
	}

	var events []PlatformEvent

	// Get channels to monitor
	channelIDs, ok := config.Metadata["channel_ids"].([]string)
	if !ok {
		// Try to convert from interface{} slice
		if channelsInterface, exists := config.Metadata["channel_ids"].([]interface{}); exists {
			channelIDs = make([]string, len(channelsInterface))
			for i, ch := range channelsInterface {
				if chStr, ok := ch.(string); ok {
					channelIDs[i] = chStr
				}
			}
		}
	}

	if len(channelIDs) == 0 {
		return events, nil // No channels configured
	}

	// Fetch messages from each channel
	for _, channelID := range channelIDs {
		if channelID == "" {
			continue
		}

		channelEvents, err := dc.fetchChannelEvents(ctx, channelID, since, limit/len(channelIDs))
		if err != nil {
			// Log error but continue with other channels
			continue
		}
		events = append(events, channelEvents...)
	}

	return events, nil
}

// fetchChannelEvents retrieves messages from a specific Discord channel
func (dc *DiscordConnector) fetchChannelEvents(ctx context.Context, channelID string, since time.Time, limit int) ([]PlatformEvent, error) {
	var events []PlatformEvent

	// Get messages from the channel
	messages, err := dc.session.ChannelMessages(channelID, limit, "", "", "")
	if err != nil {
		return nil, dc.handleDiscordError(err)
	}

	// Convert messages to platform events
	for _, msg := range messages {
		// Parse message timestamp
		msgTime := msg.Timestamp
		
		// Only include messages after the since timestamp
		if msgTime.After(since) {
			event := dc.convertMessageToEvent(msg)
			events = append(events, event)

			// Fetch thread messages if this is a thread starter
			if msg.Thread != nil {
				threadEvents, err := dc.fetchThreadMessages(ctx, msg.Thread.ID, since)
				if err == nil {
					events = append(events, threadEvents...)
				}
			}
		}
	}

	return events, nil
}

// fetchThreadMessages retrieves messages from a Discord thread
func (dc *DiscordConnector) fetchThreadMessages(ctx context.Context, threadID string, since time.Time) ([]PlatformEvent, error) {
	var events []PlatformEvent

	// Get thread depth from config
	config := dc.GetConfig()
	threadDepth, ok := config.Metadata["thread_depth"].(int)
	if !ok {
		threadDepth = 10 // Default thread depth
	}

	messages, err := dc.session.ChannelMessages(threadID, threadDepth, "", "", "")
	if err != nil {
		return nil, dc.handleDiscordError(err)
	}

	// Convert thread messages to platform events
	for _, msg := range messages {
		msgTime := msg.Timestamp
		
		if msgTime.After(since) {
			event := dc.convertMessageToEvent(msg)
			event.Metadata["thread_id"] = threadID
			event.Metadata["is_thread_message"] = true
			events = append(events, event)
		}
	}

	return events, nil
}

// convertMessageToEvent converts a Discord message to a platform event
func (dc *DiscordConnector) convertMessageToEvent(msg *discordgo.Message) PlatformEvent {
	// Use timestamp directly
	msgTime := msg.Timestamp

	// Extract file references from message content
	fileRefs := dc.extractFileReferences(msg.Content)

	// Add attachment references
	for _, attachment := range msg.Attachments {
		fileRefs = append(fileRefs, attachment.Filename)
	}

	return PlatformEvent{
		ID:        fmt.Sprintf("msg-%s", msg.ID),
		Type:      EventTypeMessage,
		Timestamp: msgTime,
		Author:    msg.Author.Username,
		Content:   msg.Content,
		Platform:  "discord",
		Metadata: map[string]interface{}{
			"message_id":    msg.ID,
			"channel_id":    msg.ChannelID,
			"guild_id":      msg.GuildID,
			"author_id":     msg.Author.ID,
			"message_type":  msg.Type,
			"attachments":   len(msg.Attachments),
			"embeds":        len(msg.Embeds),
			"reactions":     len(msg.Reactions),
			"pinned":        msg.Pinned,
			"edited":        msg.EditedTimestamp != nil,
		},
		References: fileRefs,
	}
}

// extractFileReferences extracts file references from Discord message content
func (dc *DiscordConnector) extractFileReferences(content string) []string {
	var fileRefs []string

	// Look for common file patterns in Discord messages
	words := strings.Fields(content)
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

		// Check for code blocks with file names
		if strings.HasPrefix(word, "```") && len(word) > 3 {
			// Extract potential filename from code block
			filename := strings.TrimPrefix(word, "```")
			if strings.Contains(filename, ".") {
				fileRefs = append(fileRefs, filename)
			}
		}
	}

	return fileRefs
}

// NormalizeData converts Discord platform events to normalized format
func (dc *DiscordConnector) NormalizeData(ctx context.Context, events []PlatformEvent) ([]NormalizedEvent, error) {
	normalized := make([]NormalizedEvent, len(events))
	for i, event := range events {
		normalizedEvent := dc.normalizer.NormalizeEvent(event)
		
		// Add Discord-specific normalization
		if threadID, ok := event.Metadata["thread_id"].(string); ok && threadID != "" {
			normalizedEvent.ThreadID = &threadID
		}
		
		if isThreadMsg, ok := event.Metadata["is_thread_message"].(bool); ok && isThreadMsg {
			if channelID, ok := event.Metadata["channel_id"].(string); ok {
				normalizedEvent.ParentID = &channelID
			}
		}

		normalized[i] = normalizedEvent
	}
	return normalized, nil
}

// ScheduleSync determines the next sync interval for Discord
func (dc *DiscordConnector) ScheduleSync(ctx context.Context, lastSync time.Time) (time.Duration, error) {
	// Discord has rate limits, so we sync every 2 minutes by default
	config := dc.GetConfig()
	if config.SyncConfig.SyncInterval > 0 {
		return config.SyncConfig.SyncInterval, nil
	}
	return 2 * time.Minute, nil
}

// GetPlatformInfo returns Discord connector metadata
func (dc *DiscordConnector) GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		Name:        "discord",
		DisplayName: "Discord",
		Version:     "1.0.0",
		Description: "Discord server connector for messages and threads",
		SupportedEvents: []EventType{
			EventTypeMessage,
			EventTypeThread,
			EventTypeReaction,
		},
		RateLimits: RateLimitInfo{
			RequestsPerHour:   1000, // Discord API limit
			RequestsPerMinute: 50,
			BurstLimit:        5,
			BackoffStrategy:   "exponential",
			RetryAfterHeader:  "X-RateLimit-Reset-After",
		},
		AuthType:       "bot_token",
		RequiredScopes: []string{"bot", "read_messages", "read_message_history"},
	}
}

// handleDiscordError converts Discord API errors to connector errors
func (dc *DiscordConnector) handleDiscordError(err error) error {
	errStr := err.Error()
	
	// Check for rate limit errors
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		retryAfter := 1 * time.Minute // Default Discord rate limit
		return &ConnectorError{
			Platform:   "discord",
			Code:       "rate_limit",
			Message:    "Discord API rate limit exceeded",
			Retryable:  true,
			RetryAfter: &retryAfter,
		}
	}

	// Check for authentication errors
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") {
		return &ConnectorError{
			Platform:  "discord",
			Code:      "auth_error",
			Message:   "Discord authentication failed",
			Retryable: false,
		}
	}

	// Check for permission errors
	if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") {
		return &ConnectorError{
			Platform:  "discord",
			Code:      "permission_error",
			Message:   fmt.Sprintf("Discord permission error: %v", err),
			Retryable: false,
		}
	}

	// Check for not found errors
	if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") {
		return &ConnectorError{
			Platform:  "discord",
			Code:      "not_found",
			Message:   fmt.Sprintf("Discord resource not found: %v", err),
			Retryable: false,
		}
	}

	// Check for network errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") {
		return &ConnectorError{
			Platform:  "discord",
			Code:      "network_error",
			Message:   fmt.Sprintf("Discord API network error: %v", err),
			Retryable: true,
		}
	}

	// Generic error
	return &ConnectorError{
		Platform:  "discord",
		Code:      "api_error",
		Message:   fmt.Sprintf("Discord API error: %v", err),
		Retryable: true,
	}
}