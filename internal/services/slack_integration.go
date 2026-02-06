package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/config"
	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/slack-go/slack"
)

// SlackIntegrationService handles Slack integration operations
type SlackIntegrationService interface {
	// OAuth installation flow
	ProcessOAuthInstallation(ctx context.Context, req *SlackOAuthInstallationRequest, userID string) (*models.ProjectIntegration, error)
	
	// Workspace and channel management
	GetAvailableChannels(ctx context.Context, projectID, integrationID string) ([]SlackChannelInfo, error)
	SelectChannels(ctx context.Context, req *ChannelSelectionRequest, userID string) ([]models.ProjectDataSource, error)
	
	// Configuration management
	UpdateConfiguration(ctx context.Context, req *IntegrationConfigurationRequest, userID string) (*models.ProjectIntegration, error)
	
	// Status and health
	GetIntegrationStatus(ctx context.Context, projectID, integrationID string) (*SlackIntegrationStatus, error)
	
	// Integration lifecycle
	DeleteIntegration(ctx context.Context, projectID, integrationID, userID string) error
	
	// Credential management
	RefreshCredentials(ctx context.Context, integrationID string) error
	ValidateCredentials(ctx context.Context, integrationID string) error
	ValidateWorkspaceConnection(ctx context.Context, integrationID string) error
}

// SlackIntegrationServiceImpl implements SlackIntegrationService
type SlackIntegrationServiceImpl struct {
	config     *config.Config
	httpClient *http.Client
	store      RepositoryStore
	encryptSvc EncryptionService
	logger     Logger
}

// NewSlackIntegrationService creates a new Slack integration service
func NewSlackIntegrationService(
	cfg *config.Config,
	store RepositoryStore,
	encryptSvc EncryptionService,
	logger Logger,
) SlackIntegrationService {
	return &SlackIntegrationServiceImpl{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		store:      store,
		encryptSvc: encryptSvc,
		logger:     logger,
	}
}

// Request/Response types
type SlackOAuthInstallationRequest struct {
	ProjectID string `json:"project_id"`
	Code      string `json:"code"`
	State     string `json:"state"`
}

type ChannelSelectionRequest struct {
	ProjectID     string   `json:"project_id"`
	IntegrationID string   `json:"integration_id"`
	ChannelIDs    []string `json:"channel_ids"`
	IncludeDMs    bool     `json:"include_dms"`
}

type SlackChannelInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IsChannel   bool      `json:"is_channel"`
	IsGroup     bool      `json:"is_group"`
	IsIM        bool      `json:"is_im"`
	IsMember    bool      `json:"is_member"`
	IsPrivate   bool      `json:"is_private"`
	IsArchived  bool      `json:"is_archived"`
	Topic       string    `json:"topic"`
	Purpose     string    `json:"purpose"`
	NumMembers  int       `json:"num_members"`
	Created     time.Time `json:"created"`
	Selected    bool      `json:"selected"`
}

type SlackIntegrationStatus struct {
	IntegrationID    string                 `json:"integration_id"`
	Status           string                 `json:"status"`
	LastSyncAt       *time.Time             `json:"last_sync_at"`
	LastSyncStatus   *string                `json:"last_sync_status"`
	ErrorMessage     *string                `json:"error_message"`
	CredentialsValid bool                   `json:"credentials_valid"`
	WorkspaceInfo    *SlackWorkspaceInfo    `json:"workspace_info"`
	BotInfo          *SlackBotInfo          `json:"bot_info"`
	Permissions      []string               `json:"permissions"`
	ChannelCount     int                    `json:"channel_count"`
	Configuration    map[string]interface{} `json:"configuration"`
}

type SlackWorkspaceInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
	URL    string `json:"url"`
}

type SlackBotInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	AppID   string `json:"app_id"`
	UserID  string `json:"user_id"`
}

type SlackOAuthResponse struct {
	OK          bool   `json:"ok"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	BotUserID   string `json:"bot_user_id"`
	AppID       string `json:"app_id"`
	Team        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	Enterprise interface{} `json:"enterprise"`
	AuthedUser struct {
		ID          string `json:"id"`
		Scope       string `json:"scope"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	} `json:"authed_user"`
	IncomingWebhook struct {
		Channel          string `json:"channel"`
		ChannelID        string `json:"channel_id"`
		ConfigurationURL string `json:"configuration_url"`
		URL              string `json:"url"`
	} `json:"incoming_webhook"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// ProcessOAuthInstallation processes a Slack OAuth installation
func (s *SlackIntegrationServiceImpl) ProcessOAuthInstallation(ctx context.Context, req *SlackOAuthInstallationRequest, userID string) (*models.ProjectIntegration, error) {
	// Validate state parameter for CSRF protection
	if req.State == "" {
		return nil, fmt.Errorf("state parameter required for security")
	}

	// Check if integration already exists for this project
	existing, err := s.store.GetProjectIntegrationByPlatform(ctx, req.ProjectID, string(models.PlatformSlack))
	if err == nil && existing != nil {
		return nil, fmt.Errorf("Slack integration already exists for project")
	}

	// Exchange code for access token
	tokenResponse, err := s.exchangeCodeForToken(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Validate the token and get workspace info
	client := slack.New(tokenResponse.AccessToken)
	authTest, err := client.AuthTestContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate Slack token: %w", err)
	}

	// Encrypt credentials
	encryptedBotToken, err := s.encryptSvc.Encrypt(ctx, tokenResponse.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt bot token: %w", err)
	}

	var encryptedUserToken *string
	if tokenResponse.AuthedUser.AccessToken != "" {
		encrypted, err := s.encryptSvc.Encrypt(ctx, tokenResponse.AuthedUser.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt user token: %w", err)
		}
		encryptedUserToken = &encrypted
	}

	// Create integration record
	integration := &models.ProjectIntegration{
		ID:              generateSlackID(),
		ProjectID:       req.ProjectID,
		Platform:        string(models.PlatformSlack),
		IntegrationType: string(models.IntegrationTypeOAuth),
		Status:          string(models.IntegrationStatusActive),
		Configuration: map[string]interface{}{
			"workspace_id":   tokenResponse.Team.ID,
			"workspace_name": tokenResponse.Team.Name,
			"bot_user_id":    tokenResponse.BotUserID,
			"app_id":         tokenResponse.AppID,
			"scopes":         tokenResponse.Scope,
			"user_id":        authTest.UserID,
			"user_name":      authTest.User,
			"team_id":        authTest.TeamID,
			"team_name":      authTest.Team,
		},
		Credentials: map[string]interface{}{
			"bot_token":  encryptedBotToken,
			"user_token": encryptedUserToken,
			"token_type": tokenResponse.TokenType,
			"scope":      tokenResponse.Scope,
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
	if err := s.store.CreateProjectIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}

	s.logger.Info("Slack OAuth integration created", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": integration.ID,
		"workspace_id":   tokenResponse.Team.ID,
		"workspace_name": tokenResponse.Team.Name,
		"user_id":        userID,
	})

	return integration, nil
}

// GetAvailableChannels gets available Slack channels for selection
func (s *SlackIntegrationServiceImpl) GetAvailableChannels(ctx context.Context, projectID, integrationID string) ([]SlackChannelInfo, error) {
	// Get integration
	integration, err := s.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get bot token
	botToken, err := s.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot token: %w", err)
	}

	// Get channels from Slack
	channels, err := s.getChannelsFromSlack(ctx, botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get channels from Slack: %w", err)
	}

	// Get already selected channels
	selectedChannels, err := s.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get selected channels: %w", err)
	}

	// Mark selected channels
	selectedMap := make(map[string]bool)
	for _, ch := range selectedChannels {
		selectedMap[ch.SourceID] = true
	}

	for i := range channels {
		channels[i].Selected = selectedMap[channels[i].ID]
	}

	return channels, nil
}

// SelectChannels selects channels for the integration
func (s *SlackIntegrationServiceImpl) SelectChannels(ctx context.Context, req *ChannelSelectionRequest, userID string) ([]models.ProjectDataSource, error) {
	// Get integration
	integration, err := s.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get bot token
	botToken, err := s.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot token: %w", err)
	}

	// Create Slack client
	client := slack.New(botToken)

	// Validate channel IDs and get channel details
	var dataSources []models.ProjectDataSource
	for _, channelID := range req.ChannelIDs {
		// Get channel details from Slack
		channelInfo, err := s.getChannelDetails(ctx, client, channelID)
		if err != nil {
			return nil, fmt.Errorf("failed to get channel details for %s: %w", channelID, err)
		}

		// Check if data source already exists
		existing, _ := s.store.GetProjectDataSource(ctx, generateDataSourceSlackID(req.IntegrationID, channelID))
		if existing != nil {
			continue // Skip if already exists
		}

		// Create data source
		dataSource := models.ProjectDataSource{
			ID:            generateDataSourceSlackID(req.IntegrationID, channelID),
			ProjectID:     req.ProjectID,
			IntegrationID: req.IntegrationID,
			SourceType:    string(models.SourceTypeChannel),
			SourceID:      channelID,
			SourceName:    channelInfo.Name,
			Configuration: map[string]interface{}{
				"channel_id":   channelID,
				"channel_name": channelInfo.Name,
				"is_private":   channelInfo.IsPrivate,
				"is_archived":  channelInfo.IsArchived,
				"topic":        channelInfo.Topic,
				"purpose":      channelInfo.Purpose,
				"num_members":  channelInfo.NumMembers,
			},
			IsActive:        true,
			LastIngestionAt: nil,
			IngestionStatus: nil,
			ErrorMessage:    nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := s.store.CreateProjectDataSource(ctx, &dataSource); err != nil {
			return nil, fmt.Errorf("failed to create data source for channel %s: %w", channelID, err)
		}

		dataSources = append(dataSources, dataSource)
	}

	// Update configuration with include_dms setting
	if integration.Configuration == nil {
		integration.Configuration = make(map[string]interface{})
	}
	integration.Configuration["include_dms"] = req.IncludeDMs

	updates := map[string]interface{}{
		"configuration": integration.Configuration,
		"updated_at":    time.Now(),
	}

	if err := s.store.UpdateProjectIntegration(ctx, req.IntegrationID, updates); err != nil {
		s.logger.Error("Failed to update integration configuration", err, map[string]interface{}{
			"integration_id": req.IntegrationID,
		})
	}

	s.logger.Info("Channels selected for Slack integration", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": req.IntegrationID,
		"channel_ids":    req.ChannelIDs,
		"include_dms":    req.IncludeDMs,
		"user_id":        userID,
	})

	return dataSources, nil
}

// UpdateConfiguration updates integration configuration
func (s *SlackIntegrationServiceImpl) UpdateConfiguration(ctx context.Context, req *IntegrationConfigurationRequest, userID string) (*models.ProjectIntegration, error) {
	// Get integration
	integration, err := s.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Validate configuration
	if err := s.validateConfiguration(req.Configuration); err != nil {
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

	if err := s.store.UpdateProjectIntegration(ctx, req.IntegrationID, updates); err != nil {
		return nil, fmt.Errorf("failed to update integration: %w", err)
	}

	// Get updated integration
	updatedIntegration, err := s.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated integration: %w", err)
	}

	s.logger.Info("Slack integration configuration updated", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": req.IntegrationID,
		"user_id":        userID,
	})

	return updatedIntegration, nil
}

// GetIntegrationStatus gets integration status and health
func (s *SlackIntegrationServiceImpl) GetIntegrationStatus(ctx context.Context, projectID, integrationID string) (*SlackIntegrationStatus, error) {
	// Get integration
	integration, err := s.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get channel count
	dataSources, err := s.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources: %w", err)
	}

	// Check credentials validity and get workspace info
	credentialsValid := false
	var workspaceInfo *SlackWorkspaceInfo
	var botInfo *SlackBotInfo
	var permissions []string

	botToken, err := s.getDecryptedBotToken(ctx, integration)
	if err == nil {
		// Test credentials and get workspace info
		credentialsValid, workspaceInfo, botInfo, permissions = s.testCredentials(ctx, botToken)
	}

	status := &SlackIntegrationStatus{
		IntegrationID:    integrationID,
		Status:           integration.Status,
		LastSyncAt:       integration.LastSyncAt,
		LastSyncStatus:   integration.LastSyncStatus,
		ErrorMessage:     integration.ErrorMessage,
		CredentialsValid: credentialsValid,
		WorkspaceInfo:    workspaceInfo,
		BotInfo:          botInfo,
		Permissions:      permissions,
		ChannelCount:     len(dataSources),
		Configuration:    integration.Configuration,
	}

	return status, nil
}

// DeleteIntegration deletes a Slack integration
func (s *SlackIntegrationServiceImpl) DeleteIntegration(ctx context.Context, projectID, integrationID, userID string) error {
	// Get integration
	integration, err := s.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return fmt.Errorf("integration does not belong to project")
	}

	// Delete all data sources
	dataSources, err := s.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get data sources: %w", err)
	}

	for _, dataSource := range dataSources {
		if err := s.store.DeleteProjectDataSource(ctx, dataSource.ID); err != nil {
			s.logger.Error("Failed to delete data source", err, map[string]interface{}{
				"data_source_id": dataSource.ID,
			})
		}
	}

	// Delete integration
	if err := s.store.DeleteProjectIntegration(ctx, integrationID); err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	s.logger.Info("Slack integration deleted", map[string]interface{}{
		"project_id":     projectID,
		"integration_id": integrationID,
		"user_id":        userID,
	})

	return nil
}

// RefreshCredentials refreshes integration credentials
func (s *SlackIntegrationServiceImpl) RefreshCredentials(ctx context.Context, integrationID string) error {
	// Slack bot tokens don't expire, but we can validate them
	return s.ValidateCredentials(ctx, integrationID)
}

// ValidateCredentials validates integration credentials
func (s *SlackIntegrationServiceImpl) ValidateCredentials(ctx context.Context, integrationID string) error {
	// Get integration
	integration, err := s.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	// Get bot token
	botToken, err := s.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return fmt.Errorf("failed to get bot token: %w", err)
	}

	// Test credentials
	valid, _, _, _ := s.testCredentials(ctx, botToken)
	if !valid {
		return fmt.Errorf("credentials are invalid")
	}

	return nil
}

// ValidateWorkspaceConnection validates the workspace connection
func (s *SlackIntegrationServiceImpl) ValidateWorkspaceConnection(ctx context.Context, integrationID string) error {
	// Get integration
	integration, err := s.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	// Get bot token
	botToken, err := s.getDecryptedBotToken(ctx, integration)
	if err != nil {
		return fmt.Errorf("failed to get bot token: %w", err)
	}

	// Create Slack client and test connection
	client := slack.New(botToken)
	authTest, err := client.AuthTestContext(ctx)
	if err != nil {
		return fmt.Errorf("workspace connection validation failed: %w", err)
	}

	// Verify workspace ID matches
	workspaceID, ok := integration.Configuration["workspace_id"].(string)
	if ok && workspaceID != authTest.TeamID {
		return fmt.Errorf("workspace ID mismatch: expected %s, got %s", workspaceID, authTest.TeamID)
	}

	return nil
}

// Helper methods

// getDecryptedBotToken gets and decrypts the bot token from integration
func (s *SlackIntegrationServiceImpl) getDecryptedBotToken(ctx context.Context, integration *models.ProjectIntegration) (string, error) {
	encryptedToken, ok := integration.Credentials["bot_token"].(string)
	if !ok {
		return "", fmt.Errorf("bot token not found in credentials")
	}

	return s.encryptSvc.Decrypt(ctx, encryptedToken)
}

// testCredentials tests if credentials are valid and returns workspace info
func (s *SlackIntegrationServiceImpl) testCredentials(ctx context.Context, botToken string) (bool, *SlackWorkspaceInfo, *SlackBotInfo, []string) {
	client := slack.New(botToken)
	
	authTest, err := client.AuthTestContext(ctx)
	if err != nil {
		return false, nil, nil, nil
	}

	// Get team info
	teamInfo, err := client.GetTeamInfoContext(ctx)
	if err != nil {
		return true, nil, nil, nil // Valid credentials but couldn't get team info
	}

	workspaceInfo := &SlackWorkspaceInfo{
		ID:     teamInfo.ID,
		Name:   teamInfo.Name,
		Domain: teamInfo.Domain,
		URL:    fmt.Sprintf("https://%s.slack.com", teamInfo.Domain),
	}

	botInfo := &SlackBotInfo{
		ID:     authTest.BotID,
		Name:   authTest.User,
		UserID: authTest.UserID,
	}

	// Parse scopes from auth test
	var permissions []string
	if authTest.URL != "" {
		// Scopes are typically in the URL or need to be stored during OAuth
		permissions = []string{"channels:history", "groups:history", "im:history", "mpim:history"}
	}

	return true, workspaceInfo, botInfo, permissions
}

// validateConfiguration validates integration configuration
func (s *SlackIntegrationServiceImpl) validateConfiguration(config map[string]interface{}) error {
	// Validate thread_depth if provided
	if threadDepth, ok := config["thread_depth"]; ok {
		if depth, ok := threadDepth.(float64); ok {
			if depth < 1 || depth > 100 {
				return fmt.Errorf("thread_depth must be between 1 and 100")
			}
		}
	}

	// Validate include_dms if provided
	if includeDMs, ok := config["include_dms"]; ok {
		if _, ok := includeDMs.(bool); !ok {
			return fmt.Errorf("include_dms must be a boolean")
		}
	}

	return nil
}

// generateSlackID generates a unique ID for Slack integration
func generateSlackID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:22]
}

// generateDataSourceSlackID generates a data source ID for Slack
func generateDataSourceSlackID(integrationID, channelID string) string {
	return fmt.Sprintf("%s-%s", integrationID, channelID)
}

// Slack API helper methods

// exchangeCodeForToken exchanges OAuth code for access token
func (s *SlackIntegrationServiceImpl) exchangeCodeForToken(ctx context.Context, code string) (*SlackOAuthResponse, error) {
	data := url.Values{}
	data.Set("client_id", s.config.SlackOAuth.ClientID)
	data.Set("client_secret", s.config.SlackOAuth.ClientSecret)
	data.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/oauth.v2.access", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Slack OAuth token exchange failed with status: %d", resp.StatusCode)
	}

	var tokenResp SlackOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if !tokenResp.OK {
		return nil, fmt.Errorf("Slack OAuth error: %s", tokenResp.Error)
	}

	return &tokenResp, nil
}

// getChannelsFromSlack gets all available channels from Slack
func (s *SlackIntegrationServiceImpl) getChannelsFromSlack(ctx context.Context, botToken string) ([]SlackChannelInfo, error) {
	client := slack.New(botToken)

	var allChannels []SlackChannelInfo

	// Get public channels
	publicChannels, _, err := client.GetConversationsContext(ctx, &slack.GetConversationsParameters{
		Types: []string{"public_channel"},
		Limit: 1000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get public channels: %w", err)
	}

	for _, ch := range publicChannels {
		allChannels = append(allChannels, s.convertChannelToInfo(ch))
	}

	// Get private channels (groups)
	privateChannels, _, err := client.GetConversationsContext(ctx, &slack.GetConversationsParameters{
		Types: []string{"private_channel"},
		Limit: 1000,
	})
	if err == nil { // Don't fail if we can't get private channels
		for _, ch := range privateChannels {
			allChannels = append(allChannels, s.convertChannelToInfo(ch))
		}
	}

	return allChannels, nil
}

// getChannelDetails gets details for a specific channel
func (s *SlackIntegrationServiceImpl) getChannelDetails(ctx context.Context, client *slack.Client, channelID string) (*SlackChannelInfo, error) {
	channel, err := client.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
		ChannelID: channelID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info: %w", err)
	}

	info := s.convertChannelToInfo(*channel)
	return &info, nil
}

// convertChannelToInfo converts Slack channel to SlackChannelInfo
func (s *SlackIntegrationServiceImpl) convertChannelToInfo(ch slack.Channel) SlackChannelInfo {
	return SlackChannelInfo{
		ID:         ch.ID,
		Name:       ch.Name,
		IsChannel:  ch.IsChannel,
		IsGroup:    ch.IsGroup,
		IsIM:       ch.IsIM,
		IsMember:   ch.IsMember,
		IsPrivate:  ch.IsPrivate,
		IsArchived: ch.IsArchived,
		Topic:      ch.Topic.Value,
		Purpose:    ch.Purpose.Value,
		NumMembers: ch.NumMembers,
		Created:    time.Unix(int64(ch.Created), 0),
		Selected:   false,
	}
}
