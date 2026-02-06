package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/config"
	"github.com/DevAnuragT/context_keeper/internal/models"
)

// GitHubIntegrationService handles GitHub integration operations
type GitHubIntegrationService interface {
	// App installation flow
	ProcessAppInstallation(ctx context.Context, req *GitHubInstallationRequest, userID string) (*models.ProjectIntegration, error)
	
	// OAuth installation flow
	ProcessOAuthInstallation(ctx context.Context, req *GitHubOAuthInstallationRequest, userID string) (*models.ProjectIntegration, error)
	
	// Repository management
	GetAvailableRepositories(ctx context.Context, projectID, integrationID string) ([]GitHubRepositoryInfo, error)
	SelectRepositories(ctx context.Context, req *RepositorySelectionRequest, userID string) ([]models.ProjectDataSource, error)
	
	// Configuration management
	UpdateConfiguration(ctx context.Context, req *IntegrationConfigurationRequest, userID string) (*models.ProjectIntegration, error)
	
	// Status and health
	GetIntegrationStatus(ctx context.Context, projectID, integrationID string) (*GitHubIntegrationStatus, error)
	
	// Integration lifecycle
	DeleteIntegration(ctx context.Context, projectID, integrationID, userID string) error
	
	// Credential management
	RefreshCredentials(ctx context.Context, integrationID string) error
	ValidateCredentials(ctx context.Context, integrationID string) error
}

// GitHubIntegrationServiceImpl implements GitHubIntegrationService
type GitHubIntegrationServiceImpl struct {
	config     *config.Config
	httpClient *http.Client
	store      RepositoryStore
	encryptSvc EncryptionService
	logger     Logger
}

// NewGitHubIntegrationService creates a new GitHub integration service
func NewGitHubIntegrationService(
	cfg *config.Config,
	store RepositoryStore,
	encryptSvc EncryptionService,
	logger Logger,
) GitHubIntegrationService {
	return &GitHubIntegrationServiceImpl{
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
type GitHubInstallationRequest struct {
	ProjectID      string `json:"project_id"`
	InstallationID string `json:"installation_id"`
	SetupAction    string `json:"setup_action"`
}

type GitHubOAuthInstallationRequest struct {
	ProjectID string `json:"project_id"`
	Code      string `json:"code"`
	State     string `json:"state"`
}

type RepositorySelectionRequest struct {
	ProjectID     string   `json:"project_id"`
	IntegrationID string   `json:"integration_id"`
	RepositoryIDs []string `json:"repository_ids"`
}

type IntegrationConfigurationRequest struct {
	ProjectID     string                 `json:"project_id"`
	IntegrationID string                 `json:"integration_id"`
	Configuration map[string]interface{} `json:"configuration"`
}

type GitHubRepositoryInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	Owner       struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"owner"`
	Permissions struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
	DefaultBranch string    `json:"default_branch"`
	Language      string    `json:"language"`
	UpdatedAt     time.Time `json:"updated_at"`
	Selected      bool      `json:"selected"`
}

type GitHubIntegrationStatus struct {
	IntegrationID    string                 `json:"integration_id"`
	Status           string                 `json:"status"`
	LastSyncAt       *time.Time             `json:"last_sync_at"`
	LastSyncStatus   *string                `json:"last_sync_status"`
	ErrorMessage     *string                `json:"error_message"`
	CredentialsValid bool                   `json:"credentials_valid"`
	Permissions      map[string]interface{} `json:"permissions"`
	RateLimit        *GitHubRateLimit       `json:"rate_limit"`
	RepositoryCount  int                    `json:"repository_count"`
	Configuration    map[string]interface{} `json:"configuration"`
}

type GitHubRateLimit struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"reset_at"`
}

// ProcessAppInstallation processes a GitHub App installation
func (g *GitHubIntegrationServiceImpl) ProcessAppInstallation(ctx context.Context, req *GitHubInstallationRequest, userID string) (*models.ProjectIntegration, error) {
	// Validate installation ID
	installationID, err := strconv.ParseInt(req.InstallationID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid installation ID: %w", err)
	}

	// Check if integration already exists for this project
	existing, err := g.store.GetProjectIntegrationByPlatform(ctx, req.ProjectID, string(models.PlatformGitHub))
	if err == nil && existing != nil {
		return nil, fmt.Errorf("GitHub integration already exists for project")
	}

	// Get installation details from GitHub
	installation, err := g.getInstallationDetails(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation details: %w", err)
	}

	// Generate installation access token
	accessToken, err := g.generateInstallationAccessToken(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Encrypt credentials
	encryptedToken, err := g.encryptSvc.Encrypt(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %w", err)
	}

	// Create integration record
	integration := &models.ProjectIntegration{
		ID:              generateID(),
		ProjectID:       req.ProjectID,
		Platform:        string(models.PlatformGitHub),
		IntegrationType: string(models.IntegrationTypeOAuth),
		Status:          string(models.IntegrationStatusActive),
		Configuration: map[string]interface{}{
			"installation_id":      installationID,
			"account_login":        installation.Account.Login,
			"account_type":         installation.Account.Type,
			"permissions":          installation.Permissions,
			"repository_selection": installation.RepositorySelection,
			"setup_action":         req.SetupAction,
		},
		Credentials: map[string]interface{}{
			"access_token":    encryptedToken,
			"token_type":      "installation",
			"installation_id": installationID,
			"expires_at":      time.Now().Add(1 * time.Hour),
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
	if err := g.store.CreateProjectIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}

	g.logger.Info("GitHub App integration created", map[string]interface{}{
		"project_id":      req.ProjectID,
		"integration_id":  integration.ID,
		"installation_id": installationID,
		"user_id":         userID,
	})

	return integration, nil
}
// ProcessOAuthInstallation processes a GitHub OAuth installation
func (g *GitHubIntegrationServiceImpl) ProcessOAuthInstallation(ctx context.Context, req *GitHubOAuthInstallationRequest, userID string) (*models.ProjectIntegration, error) {
	// Validate state parameter for CSRF protection
	if req.State == "" {
		return nil, fmt.Errorf("state parameter required for security")
	}

	// Check if integration already exists for this project
	existing, err := g.store.GetProjectIntegrationByPlatform(ctx, req.ProjectID, string(models.PlatformGitHub))
	if err == nil && existing != nil {
		return nil, fmt.Errorf("GitHub integration already exists for project")
	}

	// Exchange code for access token
	tokenResponse, err := g.exchangeCodeForToken(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user information
	userInfo, err := g.getGitHubUserInfo(ctx, tokenResponse.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Encrypt credentials
	encryptedToken, err := g.encryptSvc.Encrypt(ctx, tokenResponse.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %w", err)
	}

	var encryptedRefreshToken *string
	if tokenResponse.RefreshToken != "" {
		encrypted, err := g.encryptSvc.Encrypt(ctx, tokenResponse.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
		encryptedRefreshToken = &encrypted
	}

	// Create integration record
	integration := &models.ProjectIntegration{
		ID:              generateID(),
		ProjectID:       req.ProjectID,
		Platform:        string(models.PlatformGitHub),
		IntegrationType: string(models.IntegrationTypeOAuth),
		Status:          string(models.IntegrationStatusActive),
		Configuration: map[string]interface{}{
			"user_login": userInfo.Login,
			"user_id":    userInfo.ID,
			"scopes":     tokenResponse.Scope,
			"token_type": tokenResponse.TokenType,
		},
		Credentials: map[string]interface{}{
			"access_token":  encryptedToken,
			"refresh_token": encryptedRefreshToken,
			"token_type":    tokenResponse.TokenType,
			"scope":         tokenResponse.Scope,
			"expires_at":    tokenResponse.ExpiresAt,
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
	if err := g.store.CreateProjectIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}

	g.logger.Info("GitHub OAuth integration created", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": integration.ID,
		"user_login":     userInfo.Login,
		"user_id":        userID,
	})

	return integration, nil
}

// GetAvailableRepositories gets available repositories for selection
func (g *GitHubIntegrationServiceImpl) GetAvailableRepositories(ctx context.Context, projectID, integrationID string) ([]GitHubRepositoryInfo, error) {
	// Get integration
	integration, err := g.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get access token
	accessToken, err := g.getDecryptedAccessToken(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Get repositories from GitHub
	repositories, err := g.getRepositoriesFromGitHub(ctx, accessToken, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories from GitHub: %w", err)
	}

	// Get already selected repositories
	selectedRepos, err := g.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get selected repositories: %w", err)
	}

	// Mark selected repositories
	selectedMap := make(map[string]bool)
	for _, repo := range selectedRepos {
		selectedMap[repo.SourceID] = true
	}

	for i := range repositories {
		repositories[i].Selected = selectedMap[strconv.FormatInt(repositories[i].ID, 10)]
	}

	return repositories, nil
}

// SelectRepositories selects repositories for the integration
func (g *GitHubIntegrationServiceImpl) SelectRepositories(ctx context.Context, req *RepositorySelectionRequest, userID string) ([]models.ProjectDataSource, error) {
	// Get integration
	integration, err := g.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get access token
	accessToken, err := g.getDecryptedAccessToken(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Validate repository IDs and get repository details
	var dataSources []models.ProjectDataSource
	for _, repoIDStr := range req.RepositoryIDs {
		repoID, err := strconv.ParseInt(repoIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid repository ID: %s", repoIDStr)
		}

		// Get repository details from GitHub
		repoInfo, err := g.getRepositoryDetails(ctx, accessToken, repoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository details for %d: %w", repoID, err)
		}

		// Check if data source already exists
		existing, _ := g.store.GetProjectDataSource(ctx, generateDataSourceID(req.IntegrationID, repoIDStr))
		if existing != nil {
			continue // Skip if already exists
		}

		// Create data source
		dataSource := models.ProjectDataSource{
			ID:            generateDataSourceID(req.IntegrationID, repoIDStr),
			ProjectID:     req.ProjectID,
			IntegrationID: req.IntegrationID,
			SourceType:    string(models.SourceTypeRepository),
			SourceID:      repoIDStr,
			SourceName:    repoInfo.FullName,
			Configuration: map[string]interface{}{
				"repository_id":   repoID,
				"repository_name": repoInfo.Name,
				"full_name":       repoInfo.FullName,
				"owner":           repoInfo.Owner.Login,
				"private":         repoInfo.Private,
				"default_branch":  repoInfo.DefaultBranch,
				"language":        repoInfo.Language,
				"permissions":     repoInfo.Permissions,
			},
			IsActive:        true,
			LastIngestionAt: nil,
			IngestionStatus: nil,
			ErrorMessage:    nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := g.store.CreateProjectDataSource(ctx, &dataSource); err != nil {
			return nil, fmt.Errorf("failed to create data source for repository %s: %w", repoIDStr, err)
		}

		dataSources = append(dataSources, dataSource)
	}

	g.logger.Info("Repositories selected for integration", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": req.IntegrationID,
		"repository_ids": req.RepositoryIDs,
		"user_id":        userID,
	})

	return dataSources, nil
}

// UpdateConfiguration updates integration configuration
func (g *GitHubIntegrationServiceImpl) UpdateConfiguration(ctx context.Context, req *IntegrationConfigurationRequest, userID string) (*models.ProjectIntegration, error) {
	// Get integration
	integration, err := g.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Validate configuration
	if err := g.validateConfiguration(req.Configuration); err != nil {
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

	if err := g.store.UpdateProjectIntegration(ctx, req.IntegrationID, updates); err != nil {
		return nil, fmt.Errorf("failed to update integration: %w", err)
	}

	// Get updated integration
	updatedIntegration, err := g.store.GetProjectIntegration(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated integration: %w", err)
	}

	g.logger.Info("Integration configuration updated", map[string]interface{}{
		"project_id":     req.ProjectID,
		"integration_id": req.IntegrationID,
		"user_id":        userID,
	})

	return updatedIntegration, nil
}

// GetIntegrationStatus gets integration status and health
func (g *GitHubIntegrationServiceImpl) GetIntegrationStatus(ctx context.Context, projectID, integrationID string) (*GitHubIntegrationStatus, error) {
	// Get integration
	integration, err := g.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration does not belong to project")
	}

	// Get repository count
	dataSources, err := g.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources: %w", err)
	}

	// Check credentials validity
	credentialsValid := false
	var rateLimit *GitHubRateLimit
	
	accessToken, err := g.getDecryptedAccessToken(ctx, integration)
	if err == nil {
		// Test credentials and get rate limit
		credentialsValid, rateLimit = g.testCredentials(ctx, accessToken)
	}

	status := &GitHubIntegrationStatus{
		IntegrationID:    integrationID,
		Status:           integration.Status,
		LastSyncAt:       integration.LastSyncAt,
		LastSyncStatus:   integration.LastSyncStatus,
		ErrorMessage:     integration.ErrorMessage,
		CredentialsValid: credentialsValid,
		Permissions:      integration.Configuration,
		RateLimit:        rateLimit,
		RepositoryCount:  len(dataSources),
		Configuration:    integration.Configuration,
	}

	return status, nil
}

// DeleteIntegration deletes a GitHub integration
func (g *GitHubIntegrationServiceImpl) DeleteIntegration(ctx context.Context, projectID, integrationID, userID string) error {
	// Get integration
	integration, err := g.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	if integration.ProjectID != projectID {
		return fmt.Errorf("integration does not belong to project")
	}

	// Delete all data sources
	dataSources, err := g.store.GetProjectDataSourcesByIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get data sources: %w", err)
	}

	for _, dataSource := range dataSources {
		if err := g.store.DeleteProjectDataSource(ctx, dataSource.ID); err != nil {
			g.logger.Error("Failed to delete data source", err, map[string]interface{}{
				"data_source_id": dataSource.ID,
			})
		}
	}

	// Delete integration
	if err := g.store.DeleteProjectIntegration(ctx, integrationID); err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	g.logger.Info("GitHub integration deleted", map[string]interface{}{
		"project_id":     projectID,
		"integration_id": integrationID,
		"user_id":        userID,
	})

	return nil
}

// RefreshCredentials refreshes integration credentials
func (g *GitHubIntegrationServiceImpl) RefreshCredentials(ctx context.Context, integrationID string) error {
	// Get integration
	integration, err := g.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	// Handle different integration types
	switch integration.IntegrationType {
	case string(models.IntegrationTypeOAuth):
		// For GitHub Apps, generate new installation access token
		if installationID, ok := integration.Configuration["installation_id"].(float64); ok {
			accessToken, err := g.generateInstallationAccessToken(ctx, int64(installationID))
			if err != nil {
				return fmt.Errorf("failed to generate new access token: %w", err)
			}

			// Encrypt and update token
			encryptedToken, err := g.encryptSvc.Encrypt(ctx, accessToken)
			if err != nil {
				return fmt.Errorf("failed to encrypt access token: %w", err)
			}

			updates := map[string]interface{}{
				"credentials": map[string]interface{}{
					"access_token": encryptedToken,
					"token_type":   "installation",
					"expires_at":   time.Now().Add(1 * time.Hour),
				},
				"updated_at": time.Now(),
			}

			return g.store.UpdateProjectIntegration(ctx, integrationID, updates)
		}
		
		// For OAuth tokens, use refresh token if available
		if refreshToken, ok := integration.Credentials["refresh_token"].(string); ok && refreshToken != "" {
			// Decrypt refresh token
			decryptedRefreshToken, err := g.encryptSvc.Decrypt(ctx, refreshToken)
			if err != nil {
				return fmt.Errorf("failed to decrypt refresh token: %w", err)
			}

			// Refresh access token
			tokenResponse, err := g.refreshAccessToken(ctx, decryptedRefreshToken)
			if err != nil {
				return fmt.Errorf("failed to refresh access token: %w", err)
			}

			// Encrypt and update tokens
			encryptedAccessToken, err := g.encryptSvc.Encrypt(ctx, tokenResponse.AccessToken)
			if err != nil {
				return fmt.Errorf("failed to encrypt access token: %w", err)
			}

			updates := map[string]interface{}{
				"credentials": map[string]interface{}{
					"access_token": encryptedAccessToken,
					"token_type":   tokenResponse.TokenType,
					"expires_at":   tokenResponse.ExpiresAt,
				},
				"updated_at": time.Now(),
			}

			if tokenResponse.RefreshToken != "" {
				encryptedRefreshToken, err := g.encryptSvc.Encrypt(ctx, tokenResponse.RefreshToken)
				if err != nil {
					return fmt.Errorf("failed to encrypt refresh token: %w", err)
				}
				updates["credentials"].(map[string]interface{})["refresh_token"] = encryptedRefreshToken
			}

			return g.store.UpdateProjectIntegration(ctx, integrationID, updates)
		}

		return fmt.Errorf("no refresh mechanism available for this integration")

	default:
		return fmt.Errorf("unsupported integration type: %s", integration.IntegrationType)
	}
}

// ValidateCredentials validates integration credentials
func (g *GitHubIntegrationServiceImpl) ValidateCredentials(ctx context.Context, integrationID string) error {
	// Get integration
	integration, err := g.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	// Get access token
	accessToken, err := g.getDecryptedAccessToken(ctx, integration)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Test credentials
	valid, _ := g.testCredentials(ctx, accessToken)
	if !valid {
		return fmt.Errorf("credentials are invalid")
	}

	return nil
}
// Helper methods

// getDecryptedAccessToken gets and decrypts the access token from integration
func (g *GitHubIntegrationServiceImpl) getDecryptedAccessToken(ctx context.Context, integration *models.ProjectIntegration) (string, error) {
	encryptedToken, ok := integration.Credentials["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access token not found in credentials")
	}

	return g.encryptSvc.Decrypt(ctx, encryptedToken)
}

// testCredentials tests if credentials are valid and returns rate limit info
func (g *GitHubIntegrationServiceImpl) testCredentials(ctx context.Context, accessToken string) (bool, *GitHubRateLimit) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/rate_limit", nil)
	if err != nil {
		return false, nil
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var rateLimitResp struct {
		Rate struct {
			Limit     int `json:"limit"`
			Remaining int `json:"remaining"`
			Reset     int `json:"reset"`
		} `json:"rate"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rateLimitResp); err != nil {
		return true, nil // Valid credentials but couldn't parse rate limit
	}

	rateLimit := &GitHubRateLimit{
		Limit:     rateLimitResp.Rate.Limit,
		Remaining: rateLimitResp.Rate.Remaining,
		ResetAt:   time.Unix(int64(rateLimitResp.Rate.Reset), 0),
	}

	return true, rateLimit
}

// validateConfiguration validates integration configuration
func (g *GitHubIntegrationServiceImpl) validateConfiguration(config map[string]interface{}) error {
	// Add validation logic for configuration parameters
	// This is a placeholder - implement specific validation rules
	return nil
}

// generateID generates a unique ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:22]
}

// generateDataSourceID generates a data source ID
func generateDataSourceID(integrationID, sourceID string) string {
	return fmt.Sprintf("%s-%s", integrationID, sourceID)
}

// GitHub API helper methods
func (g *GitHubIntegrationServiceImpl) getInstallationDetails(ctx context.Context, installationID int64) (*GitHubInstallation, error) {
	// For now, return a mock installation - in production this would use GitHub App authentication
	return &GitHubInstallation{
		ID: installationID,
		Account: struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		}{
			Login: "example-org",
			Type:  "Organization",
		},
		Permissions: map[string]string{
			"contents": "read",
			"metadata": "read",
			"pull_requests": "read",
			"issues": "read",
		},
		RepositorySelection: "selected",
	}, nil
}

func (g *GitHubIntegrationServiceImpl) generateInstallationAccessToken(ctx context.Context, installationID int64) (string, error) {
	// For now, return a mock token - in production this would use GitHub App private key
	return "ghs_mock_installation_token_" + strconv.FormatInt(installationID, 10), nil
}

func (g *GitHubIntegrationServiceImpl) exchangeCodeForToken(ctx context.Context, code string) (*GitHubTokenResponse, error) {
	// Implementation for OAuth code exchange
	data := url.Values{}
	data.Set("client_id", g.config.GitHubOAuth.ClientID)
	data.Set("client_secret", g.config.GitHubOAuth.ClientSecret)
	data.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub OAuth token exchange failed with status: %d", resp.StatusCode)
	}

	var tokenResp GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("GitHub OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	return &tokenResp, nil
}

func (g *GitHubIntegrationServiceImpl) refreshAccessToken(ctx context.Context, refreshToken string) (*GitHubTokenResponse, error) {
	// GitHub doesn't currently support refresh tokens for OAuth apps
	return nil, fmt.Errorf("refresh tokens not supported by GitHub OAuth")
}

func (g *GitHubIntegrationServiceImpl) getGitHubUserInfo(ctx context.Context, accessToken string) (*GitHubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API request failed with status: %d", resp.StatusCode)
	}

	var githubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &GitHubUserInfo{
		ID:        fmt.Sprintf("%d", githubUser.ID),
		Login:     githubUser.Login,
		Email:     githubUser.Email,
		AvatarURL: githubUser.AvatarURL,
	}, nil
}

func (g *GitHubIntegrationServiceImpl) getRepositoriesFromGitHub(ctx context.Context, accessToken string, integration *models.ProjectIntegration) ([]GitHubRepositoryInfo, error) {
	// Get user repositories
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/repos?type=all&sort=updated&per_page=100", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API request failed with status: %d", resp.StatusCode)
	}

	var repositories []GitHubRepositoryInfo
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return nil, fmt.Errorf("failed to decode repositories response: %w", err)
	}

	return repositories, nil
}

func (g *GitHubIntegrationServiceImpl) getRepositoryDetails(ctx context.Context, accessToken string, repoID int64) (*GitHubRepositoryInfo, error) {
	// Get repository by ID
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.github.com/repositories/%d", repoID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API request failed with status: %d", resp.StatusCode)
	}

	var repository GitHubRepositoryInfo
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return nil, fmt.Errorf("failed to decode repository response: %w", err)
	}

	return &repository, nil
}

// Supporting types
type GitHubInstallation struct {
	ID                  int64 `json:"id"`
	Account             struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"account"`
	Permissions         map[string]string `json:"permissions"`
	RepositorySelection string            `json:"repository_selection"`
}

type GitHubTokenResponse struct {
	AccessToken      string     `json:"access_token"`
	TokenType        string     `json:"token_type"`
	Scope            string     `json:"scope"`
	RefreshToken     string     `json:"refresh_token,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	Error            string     `json:"error,omitempty"`
	ErrorDescription string     `json:"error_description,omitempty"`
}