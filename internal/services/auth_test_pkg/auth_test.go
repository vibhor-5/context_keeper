package auth_test_pkg

import (
	"context"
	"testing"

	"github.com/DevAnuragT/context_keeper/internal/config"
	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/DevAnuragT/context_keeper/internal/services"
	"github.com/stretchr/testify/assert"
)

// MockMultiTenantStore for isolated testing
type MockMultiTenantStore struct {
	users         map[string]*models.User
	oauthAccounts map[string]*models.UserOAuthAccount
}

func NewMockMultiTenantStore() *MockMultiTenantStore {
	return &MockMultiTenantStore{
		users:         make(map[string]*models.User),
		oauthAccounts: make(map[string]*models.UserOAuthAccount),
	}
}

func (m *MockMultiTenantStore) CreateUser(ctx context.Context, user *models.User) error {
	user.ID = "mock-user-id"
	m.users[user.Email] = user
	return nil
}

func (m *MockMultiTenantStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if user, exists := m.users[email]; exists {
		return user, nil
	}
	return nil, assert.AnError
}

func (m *MockMultiTenantStore) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return nil, assert.AnError
}

func (m *MockMultiTenantStore) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error {
	return nil
}

func (m *MockMultiTenantStore) CreateOAuthAccount(ctx context.Context, account *models.UserOAuthAccount) error {
	account.ID = "mock-oauth-id"
	key := account.Provider + ":" + account.ProviderUserID
	m.oauthAccounts[key] = account
	return nil
}

func (m *MockMultiTenantStore) GetOAuthAccount(ctx context.Context, provider, providerUserID string) (*models.UserOAuthAccount, error) {
	key := provider + ":" + providerUserID
	if account, exists := m.oauthAccounts[key]; exists {
		return account, nil
	}
	return nil, assert.AnError
}

func (m *MockMultiTenantStore) GetOAuthAccountsByUser(ctx context.Context, userID string) ([]models.UserOAuthAccount, error) {
	var accounts []models.UserOAuthAccount
	for _, account := range m.oauthAccounts {
		if account.UserID == userID {
			accounts = append(accounts, *account)
		}
	}
	return accounts, nil
}

func (m *MockMultiTenantStore) UpdateOAuthAccount(ctx context.Context, accountID string, updates map[string]interface{}) error {
	return nil
}

// Stub implementations for other required methods
func (m *MockMultiTenantStore) DeleteUser(ctx context.Context, userID string) error { return nil }
func (m *MockMultiTenantStore) DeleteOAuthAccount(ctx context.Context, accountID string) error { return nil }
func (m *MockMultiTenantStore) CreateProjectWorkspace(ctx context.Context, workspace *models.ProjectWorkspace) error { return nil }
func (m *MockMultiTenantStore) GetProjectWorkspace(ctx context.Context, projectID string) (*models.ProjectWorkspace, error) { return nil, nil }
func (m *MockMultiTenantStore) GetProjectWorkspacesByUser(ctx context.Context, userID string) ([]models.ProjectWorkspace, error) { return nil, nil }
func (m *MockMultiTenantStore) UpdateProjectWorkspace(ctx context.Context, projectID string, updates map[string]interface{}) error { return nil }
func (m *MockMultiTenantStore) DeleteProjectWorkspace(ctx context.Context, projectID string) error { return nil }
func (m *MockMultiTenantStore) CreateProjectMember(ctx context.Context, member *models.ProjectMember) error { return nil }
func (m *MockMultiTenantStore) GetProjectMember(ctx context.Context, projectID, userID string) (*models.ProjectMember, error) { return nil, nil }
func (m *MockMultiTenantStore) GetProjectMembers(ctx context.Context, projectID string) ([]models.ProjectMember, error) { return nil, nil }
func (m *MockMultiTenantStore) GetUserProjectMemberships(ctx context.Context, userID string) ([]models.ProjectMember, error) { return nil, nil }
func (m *MockMultiTenantStore) UpdateProjectMember(ctx context.Context, memberID string, updates map[string]interface{}) error { return nil }
func (m *MockMultiTenantStore) DeleteProjectMember(ctx context.Context, memberID string) error { return nil }

// Project integration operations
func (m *MockMultiTenantStore) CreateProjectIntegration(ctx context.Context, integration *models.ProjectIntegration) error { return nil }
func (m *MockMultiTenantStore) GetProjectIntegration(ctx context.Context, integrationID string) (*models.ProjectIntegration, error) { return nil, nil }
func (m *MockMultiTenantStore) GetProjectIntegrations(ctx context.Context, projectID string) ([]models.ProjectIntegration, error) { return nil, nil }
func (m *MockMultiTenantStore) GetProjectIntegrationByPlatform(ctx context.Context, projectID, platform string) (*models.ProjectIntegration, error) { return nil, nil }
func (m *MockMultiTenantStore) UpdateProjectIntegration(ctx context.Context, integrationID string, updates map[string]interface{}) error { return nil }
func (m *MockMultiTenantStore) DeleteProjectIntegration(ctx context.Context, integrationID string) error { return nil }

// Project data source operations
func (m *MockMultiTenantStore) CreateProjectDataSource(ctx context.Context, dataSource *models.ProjectDataSource) error { return nil }
func (m *MockMultiTenantStore) GetProjectDataSource(ctx context.Context, dataSourceID string) (*models.ProjectDataSource, error) { return nil, nil }
func (m *MockMultiTenantStore) GetProjectDataSources(ctx context.Context, projectID string) ([]models.ProjectDataSource, error) { return nil, nil }
func (m *MockMultiTenantStore) GetProjectDataSourcesByIntegration(ctx context.Context, integrationID string) ([]models.ProjectDataSource, error) { return nil, nil }
func (m *MockMultiTenantStore) UpdateProjectDataSource(ctx context.Context, dataSourceID string, updates map[string]interface{}) error { return nil }
func (m *MockMultiTenantStore) DeleteProjectDataSource(ctx context.Context, dataSourceID string) error { return nil }
func (m *MockMultiTenantStore) CreateRepo(ctx context.Context, repo *models.Repository) error { return nil }
func (m *MockMultiTenantStore) GetReposByUser(ctx context.Context, userID string) ([]models.Repository, error) { return nil, nil }
func (m *MockMultiTenantStore) GetRepoByID(ctx context.Context, repoID int64) (*models.Repository, error) { return nil, nil }
func (m *MockMultiTenantStore) CreatePullRequest(ctx context.Context, pr *models.PullRequest) error { return nil }
func (m *MockMultiTenantStore) GetRecentPRs(ctx context.Context, repoID int64, limit int) ([]models.PullRequest, error) { return nil, nil }
func (m *MockMultiTenantStore) CreateIssue(ctx context.Context, issue *models.Issue) error { return nil }
func (m *MockMultiTenantStore) GetRecentIssues(ctx context.Context, repoID int64, limit int) ([]models.Issue, error) { return nil, nil }
func (m *MockMultiTenantStore) CreateCommit(ctx context.Context, commit *models.Commit) error { return nil }
func (m *MockMultiTenantStore) GetRecentCommits(ctx context.Context, repoID int64, limit int) ([]models.Commit, error) { return nil, nil }
func (m *MockMultiTenantStore) CreateJob(ctx context.Context, job *models.IngestionJob) error { return nil }
func (m *MockMultiTenantStore) UpdateJobStatus(ctx context.Context, jobID int64, status models.JobStatus, errorMsg *string) error { return nil }
func (m *MockMultiTenantStore) GetJobByID(ctx context.Context, jobID int64) (*models.IngestionJob, error) { return nil, nil }
func (m *MockMultiTenantStore) GetJobsByRepo(ctx context.Context, repoID int64) ([]models.IngestionJob, error) { return nil, nil }
func (m *MockMultiTenantStore) CreateKnowledgeEntity(ctx context.Context, entity *models.KnowledgeEntity) error { return nil }
func (m *MockMultiTenantStore) GetKnowledgeEntity(ctx context.Context, id string) (*models.KnowledgeEntity, error) { return nil, nil }
func (m *MockMultiTenantStore) UpdateKnowledgeEntity(ctx context.Context, entity *models.KnowledgeEntity) error { return nil }
func (m *MockMultiTenantStore) DeleteKnowledgeEntity(ctx context.Context, id string) error { return nil }
func (m *MockMultiTenantStore) CreateKnowledgeRelationship(ctx context.Context, relationship *models.KnowledgeRelationship) error { return nil }
func (m *MockMultiTenantStore) GetKnowledgeRelationships(ctx context.Context, entityID string) ([]models.KnowledgeRelationship, error) { return nil, nil }
func (m *MockMultiTenantStore) DeleteKnowledgeRelationship(ctx context.Context, id string) error { return nil }
func (m *MockMultiTenantStore) CreateDecisionRecord(ctx context.Context, decision *models.DecisionRecord) error { return nil }
func (m *MockMultiTenantStore) GetDecisionRecord(ctx context.Context, decisionID string) (*models.DecisionRecord, error) { return nil, nil }
func (m *MockMultiTenantStore) UpdateDecisionRecord(ctx context.Context, decision *models.DecisionRecord) error { return nil }
func (m *MockMultiTenantStore) CreateDiscussionSummary(ctx context.Context, summary *models.DiscussionSummary) error { return nil }
func (m *MockMultiTenantStore) GetDiscussionSummary(ctx context.Context, summaryID string) (*models.DiscussionSummary, error) { return nil, nil }
func (m *MockMultiTenantStore) CreateFeatureContext(ctx context.Context, feature *models.FeatureContext) error { return nil }
func (m *MockMultiTenantStore) GetFeatureContext(ctx context.Context, featureID string) (*models.FeatureContext, error) { return nil, nil }
func (m *MockMultiTenantStore) UpdateFeatureContext(ctx context.Context, feature *models.FeatureContext) error { return nil }
func (m *MockMultiTenantStore) CreateFileContextHistory(ctx context.Context, fileContext *models.FileContextHistory) error { return nil }
func (m *MockMultiTenantStore) GetFileContextHistory(ctx context.Context, filePath string) ([]models.FileContextHistory, error) { return nil, nil }
func (m *MockMultiTenantStore) SearchKnowledgeEntities(ctx context.Context, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error) { return nil, nil }
func (m *MockMultiTenantStore) SearchSimilarEntities(ctx context.Context, embedding []float32, entityTypes []string, limit int) ([]models.SearchResult, error) { return nil, nil }
func (m *MockMultiTenantStore) TraverseKnowledgeGraph(ctx context.Context, startEntityID string, maxDepth int, relationshipTypes []string) (*models.GraphTraversalResult, error) { return nil, nil }
func (m *MockMultiTenantStore) GetRelatedEntities(ctx context.Context, entityID string, relationshipTypes []string, limit int) ([]models.KnowledgeEntity, error) { return nil, nil }

// Helper function
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Test the multi-tenant authentication system
func TestMultiTenantAuthentication(t *testing.T) {
	t.Run("password service functionality", func(t *testing.T) {
		passwordSvc := services.NewPasswordService()
		
		password := "testpassword123"
		hash, err := passwordSvc.HashPassword(password)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		
		// Verify correct password
		valid, err := passwordSvc.VerifyPassword(password, hash)
		assert.NoError(t, err)
		assert.True(t, valid)
		
		// Verify wrong password
		invalid, err := passwordSvc.VerifyPassword("wrongpassword", hash)
		assert.NoError(t, err)
		assert.False(t, invalid)
		
		// Test token generation
		token, err := passwordSvc.GenerateSecureToken()
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.GreaterOrEqual(t, len(token), 40)
	})
	
	t.Run("email service functionality", func(t *testing.T) {
		emailSvc := services.NewMockEmailService()
		
		ctx := context.Background()
		err := emailSvc.SendEmailVerification(ctx, "test@example.com", "token123", "John")
		assert.NoError(t, err)
		
		assert.Len(t, emailSvc.SentEmails, 1)
		assert.Equal(t, "test@example.com", emailSvc.SentEmails[0].To)
		assert.Equal(t, "verification", emailSvc.SentEmails[0].Type)
	})
	
	t.Run("google oauth service functionality", func(t *testing.T) {
		googleOAuth := services.NewMockGoogleOAuthService()
		
		authURL := googleOAuth.GetAuthURL("test-state")
		assert.Contains(t, authURL, "test-state")
		
		ctx := context.Background()
		userInfo, err := googleOAuth.HandleGoogleCallback(ctx, "test-code")
		assert.NoError(t, err)
		assert.NotNil(t, userInfo)
		assert.Equal(t, "test@example.com", userInfo.Email)
	})
	
	t.Run("user registration and login flow", func(t *testing.T) {
		cfg := &config.Config{
			JWTSecret: "test-secret",
		}
		
		mockStore := NewMockMultiTenantStore()
		mockPasswordSvc := services.NewMockPasswordService()
		mockEmailSvc := services.NewMockEmailService()
		mockGoogleOAuth := services.NewMockGoogleOAuthService()
		
		authSvc := services.NewAuthService(cfg, mockStore, mockPasswordSvc, mockEmailSvc, mockGoogleOAuth)
		
		// Test user registration
		req := &models.RegisterRequest{
			Email:     "newuser@example.com",
			Password:  "securepassword123",
			FirstName: stringPtr("Jane"),
			LastName:  stringPtr("Doe"),
		}
		
		ctx := context.Background()
		response, err := authSvc.RegisterUser(ctx, req)
		
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Token)
		assert.Equal(t, "newuser@example.com", response.User.Email)
		assert.Equal(t, "Jane", *response.User.FirstName)
		assert.Equal(t, "Doe", *response.User.LastName)
		assert.False(t, response.User.EmailVerified)
		
		// Verify email was sent
		assert.Len(t, mockEmailSvc.SentEmails, 1)
		assert.Equal(t, "verification", mockEmailSvc.SentEmails[0].Type)
		
		// Verify user was stored
		storedUser, exists := mockStore.users["newuser@example.com"]
		assert.True(t, exists)
		assert.Equal(t, "newuser@example.com", storedUser.Email)
		
		// Test duplicate registration fails
		response2, err2 := authSvc.RegisterUser(ctx, req)
		assert.Error(t, err2)
		assert.Nil(t, response2)
		assert.Contains(t, err2.Error(), "already exists")
	})
	
	t.Run("JWT token generation and validation", func(t *testing.T) {
		cfg := &config.Config{
			JWTSecret: "test-secret-key-for-testing",
		}
		
		mockStore := NewMockMultiTenantStore()
		mockPasswordSvc := services.NewMockPasswordService()
		mockEmailSvc := services.NewMockEmailService()
		mockGoogleOAuth := services.NewMockGoogleOAuthService()
		
		authSvc := services.NewAuthService(cfg, mockStore, mockPasswordSvc, mockEmailSvc, mockGoogleOAuth)
		
		user := &models.User{
			ID:    "test-user-123",
			Email: "test@example.com",
			Login: "testuser",
		}
		
		// Generate JWT
		token, err := authSvc.GenerateJWT(user)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		
		// Validate JWT
		validatedUser, err := authSvc.ValidateJWT(token)
		assert.NoError(t, err)
		assert.NotNil(t, validatedUser)
		assert.Equal(t, user.ID, validatedUser.ID)
		assert.Equal(t, user.Email, validatedUser.Email)
		assert.Equal(t, user.Login, validatedUser.Login)
		
		// Test invalid token
		_, err = authSvc.ValidateJWT("invalid.token.here")
		assert.Error(t, err)
		
		// Test tampered token
		tamperedToken := token[:len(token)-5] + "XXXXX"
		_, err = authSvc.ValidateJWT(tamperedToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token signature")
	})
}