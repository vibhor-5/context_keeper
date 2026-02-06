package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/config"
	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/bwmarrin/discordgo"
)

// DiscordIntegrationService handles Discord integration operations
type DiscordIntegrationService interface {
	// Bot installation flow
	ProcessBotInstallation(ctx context.Context, req *DiscordBotInstallationRequest, userID string) (*models.ProjectIntegration, error)
	
	// Server and channel management
	GetAvailableServers(ctx context.Context, projectID, integrationID string) ([]DiscordServerInfo, error)
	GetAvailableChannels(ctx context.Context, projectID, integrationID, guildID string) ([]DiscordChannelInfo, error)
	SelectChannels(ctx context.Context, req *DiscordChannelSelectionRequest, userID string) ([]models.ProjectDataSource, error)
	
	// Configuration management
	UpdateConfiguration(ctx context.Context, req *IntegrationConfigurationRequest, userID string) (*models.ProjectIntegration, error)
	
	// Status and health
	GetIntegrationStatus(ctx context.Context, projectID, integrationID string) (*DiscordIntegrationStatus, error)
	
	// Integration lifecycle
	DeleteIntegration(ctx context.Context, projectID, integrationID, userID string) error
	
	// Credential management
	ValidateCredentials(ctx context.Context, integrationID string) error
	ValidateBotConnection(ctx context.Context, integrationID string) error
}

// DiscordIntegrationServiceImpl implements DiscordIntegrationService
type DiscordIntegrationServiceImpl struct {
	config     *config.Config
	store      RepositoryStore
	encryptSvc EncryptionService
	logger     Logger
}

// NewDiscordIntegrationService creates a new Discord integration service
func NewDiscordIntegrationService(
	cfg *config.Config,
	store RepositoryStore,
	encryptSvc EncryptionService,
	logger Logger,
) DiscordIntegrationService {
	return &DiscordIntegrationServiceImpl{
		config:     cfg,
		store:      store,
		encryptSvc: encryptSvc,
		logger:     logger,
	}
}

// Request/Response types
type DiscordBotInstallationRequest struct {
	ProjectID string `json:"project_id"`
	BotToken  string `json:"bot_token"`
}

type DiscordChannelSelectionRequest struct {
	ProjectID     string   `json:"project_id"`
	IntegrationID string   `json:"integration_id"`
	GuildID       string   `json:"guild_id"`
	ChannelIDs    []string `json:"channel_ids"`
}

type DiscordServerInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Icon          string    `json:"icon"`
	OwnerID       string    `json:"owner_id"`
	MemberCount   int       `json:"member_count"`
	ChannelCount  int       `json:"channel_count"`
	Description   string    `json:"description"`
	JoinedAt      time.Time `json:"joined_at"`
	Selected      bool      `json:"selected"`
}

type DiscordChannelInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       int    `json:"type"` // 0=text, 2=voice, 4=category, 5=announcement, 10-12=threads
	Position   int    `json:"position"`
	Topic      string `json:"topic"`
	NSFW       bool   `json:"nsfw"`
	ParentID   string `json:"parent_id"`
	CategoryName string `json:"category_name"`
	Selected   bool   `json:"selected"`
}

type DiscordIntegrationStatus struct {
	IntegrationID    string                 `json:"integration_id"`
	Status           string                 `json:"status"`
	LastSyncAt       *time.Time             `json:"last_sync_at"`
	LastSyncStatus   *string                `json:"last_sync_status"`
	ErrorMessage     *string                `json:"error_message"`
	CredentialsValid bool                   `json:"credentials_valid"`
	BotInfo          *DiscordBotInfo        `json:"bot_info"`
	Permissions      []string               `json:"permissions"`
	ServerCount      int                    `json:"server_count"`
	ChannelCount     int                    `json:"channel_count"`
	Configuration    map[string]interface{} `json:"configuration"`
}

type DiscordBotInfo struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Bot           bool   `json:"bot"`
	Verified      bool   `json:"verified"`
}

// ProcessBotInstallation processes a Discord bot installation
func (d *DiscordIntegrationServiceImpl) ProcessBotInstallation(ctx context.Context, req *DiscordBotInstallationRequest, userID string) (*models.ProjectIntegration, error) {
	// Validate bot token
	if req.BotToken == "" {
		return nil, fmt.Errorf("bot token required")
	}

	// Check if integration already exists for this project
	existing, err := d.store.GetProjectIntegrationByPlatform(ctx, req.ProjectID, string(models.PlatformDiscord))
	if err == nil && existing != nil {
		return nil, fmt.Errorf("Discord integration already exists for project")
	}

	// Create Discord session to validate token
	session, err := discordgo.New("Bot " + req.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Get bot user info to validate token
	botUser, err := session.User("@me")
	if err != nil {
		return nil, fmt.Errorf("failed to validate Discord bot token: %w", err)
	}

	// Get guilds the bot is in
	guilds, err := session.UserGuilds(100, "", "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to get Discord guilds: %w", err)
	}

	// Encrypt bot token
	encryptedToken, err := d.encryptSvc.Encrypt(ctx, req.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt bot token: %w", err)
	}

	// Extract guild IDs
	guildIDs := make([]string, len(guilds))
	for i, guild := range guilds {
		guildIDs[i] = guild.ID
	}

	// Create integration record
	integration := &models.ProjectIntegration{
		ID:              generateDiscordID(),
		ProjectID:       req.ProjectID,
		Platform:        string(models.PlatformDiscord),
		IntegrationType: string(models.IntegrationTypeBot),
		Status:          string(models.IntegrationStatusActive),
		Configuration: map[string]interface{}{
			"bot_id":          botUser.ID,
			"bot_username":    botUser.Username,
			"bot_discriminator": botUser.Discriminator,
			"bot_verified":    botUser.Verified,
			"guild_ids":       guildIDs,
			"guild_count":     len(guilds),
		},
		Credentials: map[string]interface{}{
			"bot_token": encryptedToken,
		},
		LastSyncAt:     nil,
		LastSyncStatus: nil,
		ErrorMessage:   nil,
		SyncCheckpoint: map[string]interface{}{},
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Save integration
	if err := d.store.CreateProjectIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}

	// Close the session
	session.Close()

	d.logger.Info("Discord bot integration created", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": integration.ID,
		"bot_id":         botUser.ID,
		"bot_username":   botUser.Username,
		"guild_count":    len(guilds),
		"user_id":        userID,
	})

	return integration, nil
}

// GetAvailableServers gets available Discord servers (guilds) for selection
func (d *DiscordIntegrationServiceImpl) GetAvailableServers(ctx context.Context, projectID, integrationID string) ([]DiscordServerInfo, error) {
	// Get integration
	integration, err := d.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get bot token
	botToken, err := d.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot token: %w", err)
	}

	// Get servers from Discord
	servers, err := d.getServersFromDiscord(ctx, botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers from Discord: %w", err)
	}

	// Get already selected servers (servers with selected channels)
	dataSources, err := d.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get selected channels: %w", err)
	}

	// Mark servers that have selected channels
	selectedGuilds := make(map[string]bool)
	for _, ds := range dataSources {
		if guildID, ok := ds.Configuration["guild_id"].(string); ok {
			selectedGuilds[guildID] = true
		}
	}

	for i := range servers {
		servers[i].Selected = selectedGuilds[servers[i].ID]
	}

	return servers, nil
}

// GetAvailableChannels gets available channels for a specific Discord server
func (d *DiscordIntegrationServiceImpl) GetAvailableChannels(ctx context.Context, projectID, integrationID, guildID string) ([]DiscordChannelInfo, error) {
	// Get integration
	integration, err := d.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get bot token
	botToken, err := d.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot token: %w", err)
	}

	// Get channels from Discord
	channels, err := d.getChannelsFromDiscord(ctx, botToken, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channels from Discord: %w", err)
	}

	// Get already selected channels
	dataSources, err := d.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get selected channels: %w", err)
	}

	// Mark selected channels
	selectedMap := make(map[string]bool)
	for _, ds := range dataSources {
		selectedMap[ds.SourceID] = true
	}

	for i := range channels {
		channels[i].Selected = selectedMap[channels[i].ID]
	}

	return channels, nil
}

// SelectChannels selects channels for the integration
func (d *DiscordIntegrationServiceImpl) SelectChannels(ctx context.Context, req *DiscordChannelSelectionRequest, userID string) ([]models.ProjectDataSource, error) {
	// Get integration
	integration, err := d.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get bot token
	botToken, err := d.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot token: %w", err)
	}

	// Create Discord session
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}
	defer session.Close()

	// Get guild info
	guild, err := session.Guild(req.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild info: %w", err)
	}

	// Validate channel IDs and create data sources
	var dataSources []models.ProjectDataSource
	for _, channelID := range req.ChannelIDs {
		// Get channel details from Discord
		channel, err := session.Channel(channelID)
		if err != nil {
			return nil, fmt.Errorf("failed to get channel details for %s: %w", channelID, err)
		}

		// Verify channel belongs to the guild
		if channel.GuildID != req.GuildID {
			return nil, fmt.Errorf("channel %s does not belong to guild %s", channelID, req.GuildID)
		}

		// Check if data source already exists
		existing, _ := d.store.GetProjectDataSource(ctx, generateDataSourceDiscordID(req.IntegrationID, channelID))
		if existing != nil {
			continue // Skip if already exists
		}

		// Get category name if channel has a parent
		categoryName := ""
		if channel.ParentID != "" {
			if parent, err := session.Channel(channel.ParentID); err == nil {
				categoryName = parent.Name
			}
		}

		// Create data source
		dataSource := models.ProjectDataSource{
			ID:            generateDataSourceDiscordID(req.IntegrationID, channelID),
			ProjectID:     req.ProjectID,
			IntegrationID: req.IntegrationID,
			SourceType:    string(models.SourceTypeChannel),
			SourceID:      channelID,
			SourceName:    channel.Name,
			Configuration: map[string]interface{}{
				"channel_id":    channelID,
				"channel_name":  channel.Name,
				"channel_type":  channel.Type,
				"guild_id":      req.GuildID,
				"guild_name":    guild.Name,
				"topic":         channel.Topic,
				"nsfw":          channel.NSFW,
				"parent_id":     channel.ParentID,
				"category_name": categoryName,
				"position":      channel.Position,
			},
			IsActive:        true,
			LastIngestionAt: nil,
			IngestionStatus: nil,
			ErrorMessage:    nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := d.store.CreateProjectDataSource(ctx, &dataSource); err != nil {
			return nil, fmt.Errorf("failed to create data source for channel %s: %w", channelID, err)
		}

		dataSources = append(dataSources, dataSource)
	}

	d.logger.Info("Channels selected for Discord integration", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": req.IntegrationID,
		"guild_id":       req.GuildID,
		"channel_ids":    req.ChannelIDs,
		"user_id":        userID,
	})

	return dataSources, nil
}

// UpdateConfiguration updates integration configuration
func (d *DiscordIntegrationServiceImpl) UpdateConfiguration(ctx context.Context, req *IntegrationConfigurationRequest, userID string) (*models.ProjectIntegration, error) {
	// Get integration
	integration, err := d.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Validate configuration
	if err := d.validateConfiguration(req.Configuration); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Merge configuration
	if integration.Configuration == nil {
		integration.Configuration = make(map[string]interface{})
	}

	for key, value := range req.Configuration {
		integration.Configuration[key] = value
	}

	// Update integration
	updates := map[string]interface{}{
		"configuration": integration.Configuration,
		"updated_at":    time.Now(),
	}

	if err := d.store.UpdateProjectIntegration(ctx, req.IntegrationID, updates); err != nil {
		return nil, fmt.Errorf("failed to update integration: %w", err)
	}

	// Get updated integration
	updatedIntegration, err := d.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated integration: %w", err)
	}

	d.logger.Info("Discord integration configuration updated", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": req.IntegrationID,
		"user_id":        userID,
	})

	return updatedIntegration, nil
}

// GetIntegrationStatus gets integration status and health
func (d *DiscordIntegrationServiceImpl) GetIntegrationStatus(ctx context.Context, projectID, integrationID string) (*DiscordIntegrationStatus, error) {
	// Get integration
	integration, err := d.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get data sources
	dataSources, err := d.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources: %w", err)
	}

	// Count unique guilds
	guilds := make(map[string]bool)
	for _, ds := range dataSources {
		if guildID, ok := ds.Configuration["guild_id"].(string); ok {
			guilds[guildID] = true
		}
	}

	// Check credentials validity and get bot info
	credentialsValid := false
	var botInfo *DiscordBotInfo
	var permissions []string

	botToken, err := d.getDecryptedBotToken(ctx, integration)
	if err == nil {
		// Test credentials and get bot info
		credentialsValid, botInfo, permissions = d.testCredentials(ctx, botToken)
	}

	status := &DiscordIntegrationStatus{
		IntegrationID:    integrationID,
		Status:           integration.Status,
		LastSyncAt:       integration.LastSyncAt,
		LastSyncStatus:   integration.LastSyncStatus,
		ErrorMessage:     integration.ErrorMessage,
		CredentialsValid: credentialsValid,
		BotInfo:          botInfo,
		Permissions:      permissions,
		ServerCount:      len(guilds),
		ChannelCount:     len(dataSources),
		Configuration:    integration.Configuration,
	}

	return status, nil
}

// DeleteIntegration deletes a Discord integration
func (d *DiscordIntegrationServiceImpl) DeleteIntegration(ctx context.Context, projectID, integrationID, userID string) error {
	// Get integration
	integration, err := d.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return fmt.Errorf("integration does not belong to project")
	}

	// Delete all data sources
	dataSources, err := d.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get data sources: %w", err)
	}

	for _, dataSource := range dataSources {
		if err := d.store.DeleteProjectDataSource(ctx, dataSource.ID); err != nil {
			d.logger.Error("Failed to delete data source", err, map[string]interface{}{
				"data_source_id": dataSource.ID,
			})
		}
	}

	// Delete integration
	if err := d.store.DeleteProjectIntegration(ctx, integrationID); err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	d.logger.Info("Discord integration deleted", map[string]interface{}{
		"project_id":     projectID,
		"integration_id": integrationID,
		"user_id":        userID,
	})

	return nil
}

// ValidateCredentials validates integration credentials
func (d *DiscordIntegrationServiceImpl) ValidateCredentials(ctx context.Context, integrationID string) error {
	// Get integration
	integration, err := d.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	// Get bot token
	botToken, err := d.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return fmt.Errorf("failed to get bot token: %w", err)
	}

	// Test credentials
	valid, _, _ := d.testCredentials(ctx, botToken)
	if !valid {
		return fmt.Errorf("credentials are invalid")
	}

	return nil
}

// ValidateBotConnection validates the bot connection
func (d *DiscordIntegrationServiceImpl) ValidateBotConnection(ctx context.Context, integrationID string) error {
	// Get integration
	integration, err := d.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	// Get bot token
	botToken, err := d.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return fmt.Errorf("failed to get bot token: %w", err)
	}

	// Create Discord session and test connection
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
	}
	defer session.Close()

	// Get bot user info
	botUser, err := session.User("@me")
	if err != nil {
		return fmt.Errorf("bot connection validation failed: %w", err)
	}

	// Verify bot ID matches
	botID, ok := integration.Configuration["bot_id"].(string)
	if ok && botID != botUser.ID {
		return fmt.Errorf("bot ID mismatch: expected %s, got %s", botID, botUser.ID)
	}

	return nil
}

// Helper methods

// getDecryptedBotToken gets and decrypts the bot token from integration
func (d *DiscordIntegrationServiceImpl) getDecryptedBotToken(ctx context.Context, integration *models.ProjectIntegration) (string, error) {
	encryptedToken, ok := integration.Credentials["bot_token"].(string)
	if !ok {
		return "", fmt.Errorf("bot token not found in credentials")
	}

	return d.encryptSvc.Decrypt(ctx, encryptedToken)
}

// testCredentials tests if credentials are valid and returns bot info
func (d *DiscordIntegrationServiceImpl) testCredentials(ctx context.Context, botToken string) (bool, *DiscordBotInfo, []string) {
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return false, nil, nil
	}
	defer session.Close()

	botUser, err := session.User("@me")
	if err != nil {
		return false, nil, nil
	}

	botInfo := &DiscordBotInfo{
		ID:            botUser.ID,
		Username:      botUser.Username,
		Discriminator: botUser.Discriminator,
		Avatar:        botUser.Avatar,
		Bot:           botUser.Bot,
		Verified:      botUser.Verified,
	}

	// Discord bot permissions are guild-specific
	// Return common required permissions
	permissions := []string{
		"VIEW_CHANNEL",
		"READ_MESSAGE_HISTORY",
		"READ_MESSAGES",
	}

	return true, botInfo, permissions
}

// validateConfiguration validates integration configuration
func (d *DiscordIntegrationServiceImpl) validateConfiguration(config map[string]interface{}) error {
	// Validate thread_depth if provided
	if threadDepth, ok := config["thread_depth"]; ok {
		if depth, ok := threadDepth.(float64); ok {
			if depth < 1 || depth > 100 {
				return fmt.Errorf("thread_depth must be between 1 and 100")
			}
		}
	}

	return nil
}

// generateDiscordID generates a unique ID for Discord integration
func generateDiscordID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:22]
}

// generateDataSourceDiscordID generates a data source ID for Discord
func generateDataSourceDiscordID(integrationID, channelID string) string {
	return fmt.Sprintf("%s-%s", integrationID, channelID)
}

// Discord API helper methods

// getServersFromDiscord gets all available servers (guilds) from Discord
func (d *DiscordIntegrationServiceImpl) getServersFromDiscord(ctx context.Context, botToken string) ([]DiscordServerInfo, error) {
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}
	defer session.Close()

	guilds, err := session.UserGuilds(100, "", "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to get guilds: %w", err)
	}

	var servers []DiscordServerInfo
	for _, guild := range guilds {
		// Get full guild info
		fullGuild, err := session.Guild(guild.ID)
		if err != nil {
			d.logger.Error("Failed to get full guild info", err, map[string]interface{}{
				"guild_id": guild.ID,
			})
			continue
		}

		// Get guild channels to count them
		channels, err := session.GuildChannels(guild.ID)
		channelCount := 0
		if err == nil {
			// Count only text channels
			for _, ch := range channels {
				if ch.Type == discordgo.ChannelTypeGuildText || ch.Type == discordgo.ChannelTypeGuildNews {
					channelCount++
				}
			}
		}

		servers = append(servers, DiscordServerInfo{
			ID:           fullGuild.ID,
			Name:         fullGuild.Name,
			Icon:         fullGuild.Icon,
			OwnerID:      fullGuild.OwnerID,
			MemberCount:  fullGuild.MemberCount,
			ChannelCount: channelCount,
			Description:  fullGuild.Description,
			JoinedAt:     fullGuild.JoinedAt,
			Selected:     false,
		})
	}

	return servers, nil
}

// getChannelsFromDiscord gets all available channels for a guild from Discord
func (d *DiscordIntegrationServiceImpl) getChannelsFromDiscord(ctx context.Context, botToken, guildID string) ([]DiscordChannelInfo, error) {
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}
	defer session.Close()

	channels, err := session.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild channels: %w", err)
	}

	// Build category map for category names
	categoryMap := make(map[string]string)
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildCategory {
			categoryMap[ch.ID] = ch.Name
		}
	}

	var channelInfos []DiscordChannelInfo
	for _, ch := range channels {
		// Only include text channels and announcement channels
		if ch.Type != discordgo.ChannelTypeGuildText && ch.Type != discordgo.ChannelTypeGuildNews {
			continue
		}

		categoryName := ""
		if ch.ParentID != "" {
			categoryName = categoryMap[ch.ParentID]
		}

		channelInfos = append(channelInfos, DiscordChannelInfo{
			ID:           ch.ID,
			Name:         ch.Name,
			Type:         int(ch.Type),
			Position:     ch.Position,
			Topic:        ch.Topic,
			NSFW:         ch.NSFW,
			ParentID:     ch.ParentID,
			CategoryName: categoryName,
			Selected:     false,
		})
	}

	return channelInfos, nil
}
