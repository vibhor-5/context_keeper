package services

import (
	"context"
	"time"
	"github.com/DevAnuragT/context_keeper/internal/models"
)

// FileContextResponse represents the response for file context queries
type FileContextResponse struct {
	FilePath         string                    `json:"file_path"`
	FileContexts     []models.FileContextHistory `json:"file_contexts"`
	RelatedEntities  []models.SearchResult     `json:"related_entities"`
	RelatedDecisions []models.SearchResult     `json:"related_decisions"`
}

// DecisionHistoryResponse represents the response for decision history queries
type DecisionHistoryResponse struct {
	Target    string                  `json:"target"`
	Decisions []models.DecisionRecord `json:"decisions"`
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

// AuthService handles authentication operations
type AuthService interface {
	// Legacy GitHub OAuth (for backward compatibility)
	HandleGitHubCallback(ctx context.Context, code string) (*models.AuthResponse, error)
	ValidateJWT(token string) (*models.User, error)
	GenerateJWT(user *models.User) (string, error)
	
	// New multi-tenant authentication methods
	RegisterUser(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error)
	LoginWithEmailPassword(ctx context.Context, req *models.EmailPasswordAuthRequest) (*models.AuthResponse, error)
	HandleOAuthCallback(ctx context.Context, provider string, code string) (*models.AuthResponse, error)
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	VerifyEmail(ctx context.Context, token string) error
	ResendEmailVerification(ctx context.Context, email string) error
	
	// User management
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error
	
	// OAuth account management
	LinkOAuthAccount(ctx context.Context, userID string, provider string, account *models.UserOAuthAccount) error
	UnlinkOAuthAccount(ctx context.Context, userID string, provider string) error
	GetOAuthAccounts(ctx context.Context, userID string) ([]models.UserOAuthAccount, error)
}

// GitHubService handles GitHub API operations
type GitHubService interface {
	GetUserRepos(ctx context.Context, token string) ([]models.Repository, error)
	GetPullRequests(ctx context.Context, token, owner, repo string, limit int) ([]models.PullRequest, error)
	GetIssues(ctx context.Context, token, owner, repo string, limit int) ([]models.Issue, error)
	GetCommits(ctx context.Context, token, owner, repo string, limit int) ([]models.Commit, error)
	GetUserInfo(ctx context.Context, token string) (*models.User, error)
}

// JobService handles background ingestion jobs
type JobService interface {
	CreateIngestionJob(ctx context.Context, repoID int64, userID string) (*models.IngestionJob, error)
	GetJobStatus(ctx context.Context, jobID int64) (*models.IngestionJob, error)
	ProcessJob(ctx context.Context, job *models.IngestionJob, githubToken string) error
}

// ContextService handles AI service integration
type ContextService interface {
	ProcessQuery(ctx context.Context, repoID int64, query, mode string) (*models.ContextResponse, error)
	FilterRepoData(ctx context.Context, repoID int64) (*models.RepoContext, error)
	
	// Project-scoped context operations
	ProcessQueryByProject(ctx context.Context, projectID string, query, mode string) (*models.ContextResponse, error)
	FilterProjectData(ctx context.Context, projectID string) (*models.RepoContext, error)
}

// KnowledgeGraphService handles knowledge graph operations
type KnowledgeGraphService interface {
	SearchKnowledge(ctx context.Context, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error)
	GetContextForFile(ctx context.Context, filePath string) (*FileContextResponse, error)
	GetDecisionHistory(ctx context.Context, target string) (*DecisionHistoryResponse, error)
	GetRecentArchitectureDiscussions(ctx context.Context, limit int) ([]models.DiscussionSummary, error)
	
	// Project-scoped knowledge graph operations
	SearchKnowledgeByProject(ctx context.Context, projectID string, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error)
	GetContextForFileByProject(ctx context.Context, projectID, filePath string) (*FileContextResponse, error)
	GetDecisionHistoryByProject(ctx context.Context, projectID, target string) (*DecisionHistoryResponse, error)
	GetRecentArchitectureDiscussionsByProject(ctx context.Context, projectID string, limit int) ([]models.DiscussionSummary, error)
}

// PermissionService handles project-level permission checks
type PermissionService interface {
	// Project access validation
	CanAccessProject(ctx context.Context, userID, projectID string) (bool, error)
	CanReadProject(ctx context.Context, userID, projectID string) (bool, error)
	CanWriteProject(ctx context.Context, userID, projectID string) (bool, error)
	CanAdminProject(ctx context.Context, userID, projectID string) (bool, error)
	
	// Resource access validation
	CanAccessRepository(ctx context.Context, userID string, repoID int64) (bool, error)
	CanAccessKnowledgeEntity(ctx context.Context, userID, entityID string) (bool, error)
	
	// Project membership validation
	GetUserProjectRole(ctx context.Context, userID, projectID string) (models.UserRole, error)
	IsProjectMember(ctx context.Context, userID, projectID string) (bool, error)
	IsProjectOwner(ctx context.Context, userID, projectID string) (bool, error)
}

// IngestionOrchestrator manages project-scoped data ingestion across all platforms
type IngestionOrchestrator interface {
	// Start ingestion for a project
	StartProjectIngestion(ctx context.Context, projectID string) error
	
	// Start ingestion for a specific integration
	StartIntegrationIngestion(ctx context.Context, integrationID string) error
	
	// Start ingestion for a specific data source
	StartDataSourceIngestion(ctx context.Context, dataSourceID string) error
	
	// Stop ingestion for a project
	StopProjectIngestion(ctx context.Context, projectID string) error
	
	// Get ingestion health status
	GetIngestionHealth(ctx context.Context, projectID string) (*IngestionHealthStatus, error)
	GetIntegrationHealth(ctx context.Context, integrationID string) (*IntegrationHealthStatus, error)
	
	// Sync checkpoint management
	GetSyncCheckpoint(ctx context.Context, integrationID string) (map[string]interface{}, error)
	UpdateSyncCheckpoint(ctx context.Context, integrationID string, checkpoint map[string]interface{}) error
	
	// Retry management
	RetryFailedIngestion(ctx context.Context, integrationID string) error
	
	// Background orchestration
	StartOrchestrator(ctx context.Context) error
	StopOrchestrator(ctx context.Context) error
}

// IngestionHealthStatus represents the health status of project ingestion
type IngestionHealthStatus struct {
	ProjectID           string                      `json:"project_id"`
	OverallStatus       string                      `json:"overall_status"`
	ActiveIntegrations  int                         `json:"active_integrations"`
	HealthyIntegrations int                         `json:"healthy_integrations"`
	FailedIntegrations  int                         `json:"failed_integrations"`
	LastSyncAt          *time.Time                  `json:"last_sync_at"`
	Integrations        []IntegrationHealthStatus   `json:"integrations"`
}

// IntegrationHealthStatus represents the health status of an integration
type IntegrationHealthStatus struct {
	IntegrationID      string                 `json:"integration_id"`
	Platform           string                 `json:"platform"`
	Status             string                 `json:"status"`
	LastSyncAt         *time.Time             `json:"last_sync_at"`
	LastSyncStatus     *string                `json:"last_sync_status"`
	ErrorMessage       *string                `json:"error_message"`
	ErrorCount         int                    `json:"error_count"`
	DataSourceCount    int                    `json:"data_source_count"`
	ActiveDataSources  int                    `json:"active_data_sources"`
	SyncCheckpoint     map[string]interface{} `json:"sync_checkpoint"`
	NextSyncScheduled  *time.Time             `json:"next_sync_scheduled"`
}

// RepositoryStore handles database operations
type RepositoryStore interface {
	// Repository operations
	CreateRepo(ctx context.Context, repo *models.Repository) error
	GetReposByUser(ctx context.Context, userID string) ([]models.Repository, error)
	GetRepoByID(ctx context.Context, repoID int64) (*models.Repository, error)
	
	// Project-scoped repository operations
	GetReposByProject(ctx context.Context, projectID string) ([]models.Repository, error)
	GetRepoByIDAndProject(ctx context.Context, repoID int64, projectID string) (*models.Repository, error)
	
	// Pull request operations
	CreatePullRequest(ctx context.Context, pr *models.PullRequest) error
	GetRecentPRs(ctx context.Context, repoID int64, limit int) ([]models.PullRequest, error)
	
	// Project-scoped pull request operations
	GetRecentPRsByProject(ctx context.Context, projectID string, limit int) ([]models.PullRequest, error)
	
	// Issue operations
	CreateIssue(ctx context.Context, issue *models.Issue) error
	GetRecentIssues(ctx context.Context, repoID int64, limit int) ([]models.Issue, error)
	
	// Project-scoped issue operations
	GetRecentIssuesByProject(ctx context.Context, projectID string, limit int) ([]models.Issue, error)
	
	// Commit operations
	CreateCommit(ctx context.Context, commit *models.Commit) error
	GetRecentCommits(ctx context.Context, repoID int64, limit int) ([]models.Commit, error)
	
	// Project-scoped commit operations
	GetRecentCommitsByProject(ctx context.Context, projectID string, limit int) ([]models.Commit, error)
	
	// Job operations
	CreateJob(ctx context.Context, job *models.IngestionJob) error
	UpdateJobStatus(ctx context.Context, jobID int64, status models.JobStatus, errorMsg *string) error
	GetJobByID(ctx context.Context, jobID int64) (*models.IngestionJob, error)
	GetJobsByRepo(ctx context.Context, repoID int64) ([]models.IngestionJob, error)
	
	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error
	DeleteUser(ctx context.Context, userID string) error
	
	// OAuth account operations
	CreateOAuthAccount(ctx context.Context, account *models.UserOAuthAccount) error
	GetOAuthAccount(ctx context.Context, provider, providerUserID string) (*models.UserOAuthAccount, error)
	GetOAuthAccountsByUser(ctx context.Context, userID string) ([]models.UserOAuthAccount, error)
	UpdateOAuthAccount(ctx context.Context, accountID string, updates map[string]interface{}) error
	DeleteOAuthAccount(ctx context.Context, accountID string) error
	
	// Project workspace operations
	CreateProjectWorkspace(ctx context.Context, workspace *models.ProjectWorkspace) error
	GetProjectWorkspace(ctx context.Context, projectID string) (*models.ProjectWorkspace, error)
	GetProjectWorkspacesByUser(ctx context.Context, userID string) ([]models.ProjectWorkspace, error)
	UpdateProjectWorkspace(ctx context.Context, projectID string, updates map[string]interface{}) error
	DeleteProjectWorkspace(ctx context.Context, projectID string) error
	
	// Project member operations
	CreateProjectMember(ctx context.Context, member *models.ProjectMember) error
	GetProjectMember(ctx context.Context, projectID, userID string) (*models.ProjectMember, error)
	GetProjectMembers(ctx context.Context, projectID string) ([]models.ProjectMember, error)
	GetUserProjectMemberships(ctx context.Context, userID string) ([]models.ProjectMember, error)
	UpdateProjectMember(ctx context.Context, memberID string, updates map[string]interface{}) error
	DeleteProjectMember(ctx context.Context, memberID string) error
	
	// Project integration operations
	CreateProjectIntegration(ctx context.Context, integration *models.ProjectIntegration) error
	GetProjectIntegration(ctx context.Context, integrationID string) (*models.ProjectIntegration, error)
	GetProjectIntegrations(ctx context.Context, projectID string) ([]models.ProjectIntegration, error)
	GetProjectIntegrationByPlatform(ctx context.Context, projectID, platform string) (*models.ProjectIntegration, error)
	UpdateProjectIntegration(ctx context.Context, integrationID string, updates map[string]interface{}) error
	DeleteProjectIntegration(ctx context.Context, integrationID string) error
	
	// Project data source operations
	CreateProjectDataSource(ctx context.Context, dataSource *models.ProjectDataSource) error
	GetProjectDataSource(ctx context.Context, dataSourceID string) (*models.ProjectDataSource, error)
	GetProjectDataSources(ctx context.Context, projectID string) ([]models.ProjectDataSource, error)
	GetProjectDataSourcesByIntegration(ctx context.Context, integrationID string) ([]models.ProjectDataSource, error)
	UpdateProjectDataSource(ctx context.Context, dataSourceID string, updates map[string]interface{}) error
	DeleteProjectDataSource(ctx context.Context, dataSourceID string) error
	
	// Knowledge Graph operations
	CreateKnowledgeEntity(ctx context.Context, entity *models.KnowledgeEntity) error
	GetKnowledgeEntity(ctx context.Context, id string) (*models.KnowledgeEntity, error)
	UpdateKnowledgeEntity(ctx context.Context, entity *models.KnowledgeEntity) error
	DeleteKnowledgeEntity(ctx context.Context, id string) error
	
	// Project-scoped knowledge graph operations
	GetKnowledgeEntitiesByProject(ctx context.Context, projectID string, entityTypes []string, limit int) ([]models.KnowledgeEntity, error)
	GetKnowledgeEntityByIDAndProject(ctx context.Context, id, projectID string) (*models.KnowledgeEntity, error)
	
	CreateKnowledgeRelationship(ctx context.Context, relationship *models.KnowledgeRelationship) error
	GetKnowledgeRelationships(ctx context.Context, entityID string) ([]models.KnowledgeRelationship, error)
	DeleteKnowledgeRelationship(ctx context.Context, id string) error
	
	CreateDecisionRecord(ctx context.Context, decision *models.DecisionRecord) error
	GetDecisionRecord(ctx context.Context, decisionID string) (*models.DecisionRecord, error)
	UpdateDecisionRecord(ctx context.Context, decision *models.DecisionRecord) error
	
	CreateDiscussionSummary(ctx context.Context, summary *models.DiscussionSummary) error
	GetDiscussionSummary(ctx context.Context, summaryID string) (*models.DiscussionSummary, error)
	
	CreateFeatureContext(ctx context.Context, feature *models.FeatureContext) error
	GetFeatureContext(ctx context.Context, featureID string) (*models.FeatureContext, error)
	UpdateFeatureContext(ctx context.Context, feature *models.FeatureContext) error
	
	CreateFileContextHistory(ctx context.Context, fileContext *models.FileContextHistory) error
	GetFileContextHistory(ctx context.Context, filePath string) ([]models.FileContextHistory, error)
	
	// Semantic search operations
	SearchKnowledgeEntities(ctx context.Context, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error)
	SearchSimilarEntities(ctx context.Context, embedding []float32, entityTypes []string, limit int) ([]models.SearchResult, error)
	
	// Project-scoped semantic search operations
	SearchKnowledgeEntitiesByProject(ctx context.Context, projectID string, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error)
	SearchSimilarEntitiesByProject(ctx context.Context, projectID string, embedding []float32, entityTypes []string, limit int) ([]models.SearchResult, error)
	
	// Graph traversal operations
	TraverseKnowledgeGraph(ctx context.Context, startEntityID string, maxDepth int, relationshipTypes []string) (*models.GraphTraversalResult, error)
	GetRelatedEntities(ctx context.Context, entityID string, relationshipTypes []string, limit int) ([]models.KnowledgeEntity, error)
	
	// Project-scoped graph traversal operations
	TraverseKnowledgeGraphByProject(ctx context.Context, projectID, startEntityID string, maxDepth int, relationshipTypes []string) (*models.GraphTraversalResult, error)
	GetRelatedEntitiesByProject(ctx context.Context, projectID, entityID string, relationshipTypes []string, limit int) ([]models.KnowledgeEntity, error)
}

