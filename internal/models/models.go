package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Repository represents a GitHub repository
type Repository struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	FullName  string     `json:"full_name"`
	Owner     string     `json:"owner"`
	ProjectID *string    `json:"project_id,omitempty"` // Multi-tenant project scoping
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID           int64      `json:"id"`
	RepoID       int64      `json:"repo_id"`
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	Author       string     `json:"author"`
	State        string     `json:"state"`
	CreatedAt    time.Time  `json:"created_at"`
	MergedAt     *time.Time `json:"merged_at"`
	FilesChanged StringList `json:"files_changed"`
	Labels       StringList `json:"labels"`
}

// Issue represents a GitHub issue
type Issue struct {
	ID        int64      `json:"id"`
	RepoID    int64      `json:"repo_id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Author    string     `json:"author"`
	State     string     `json:"state"`
	CreatedAt time.Time  `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	Labels    StringList `json:"labels"`
}

// Commit represents a GitHub commit
type Commit struct {
	SHA          string     `json:"sha"`
	RepoID       int64      `json:"repo_id"`
	Message      string     `json:"message"`
	Author       string     `json:"author"`
	CreatedAt    time.Time  `json:"created_at"`
	FilesChanged StringList `json:"files_changed"`
}

// IngestionJob represents a background ingestion job
type IngestionJob struct {
	ID         int64      `json:"id"`
	RepoID     int64      `json:"repo_id"`
	Status     JobStatus  `json:"status"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	ErrorMsg   *string    `json:"error_message"`
}

// JobStatus represents the status of an ingestion job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusPartial   JobStatus = "partial"
	JobStatusFailed    JobStatus = "failed"
)

// StringList is a custom type for handling JSONB string arrays
type StringList []string

// Value implements the driver.Valuer interface for database storage
func (sl StringList) Value() (driver.Value, error) {
	if sl == nil {
		return nil, nil
	}
	return json.Marshal(sl)
}

// Scan implements the sql.Scanner interface for database retrieval
func (sl *StringList) Scan(value interface{}) error {
	if value == nil {
		*sl = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into StringList", value)
	}

	return json.Unmarshal(bytes, sl)
}

// User represents an authenticated user
type User struct {
	ID                        string     `json:"id"`
	Email                     string     `json:"email"`
	PasswordHash              *string    `json:"-"` // Never serialize the password hash
	FirstName                 *string    `json:"first_name,omitempty"`
	LastName                  *string    `json:"last_name,omitempty"`
	AvatarURL                 *string    `json:"avatar_url,omitempty"`
	EmailVerified             bool       `json:"email_verified"`
	EmailVerificationToken    *string    `json:"-"` // Never serialize tokens
	EmailVerificationExpiresAt *time.Time `json:"-"`
	PasswordResetToken        *string    `json:"-"`
	PasswordResetExpiresAt    *time.Time `json:"-"`
	LastLoginAt               *time.Time `json:"last_login_at,omitempty"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
	
	// Legacy fields for backward compatibility
	Login       string `json:"login,omitempty"` // Populated from OAuth accounts
	GitHubToken string `json:"-"` // Never serialize the token, populated from OAuth accounts
}

// UserOAuthAccount represents an OAuth account linked to a user
type UserOAuthAccount struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	Provider         string     `json:"provider"` // github, google
	ProviderUserID   string     `json:"provider_user_id"`
	ProviderUsername *string    `json:"provider_username,omitempty"`
	AccessToken      string     `json:"-"` // Never serialize tokens
	RefreshToken     *string    `json:"-"`
	TokenExpiresAt   *time.Time `json:"token_expires_at,omitempty"`
	Scope            *string    `json:"scope,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ProjectWorkspace represents a multi-tenant project workspace
type ProjectWorkspace struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	OwnerID     string                 `json:"owner_id"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ProjectMember represents a user's membership in a project workspace
type ProjectMember struct {
	ID          string                 `json:"id"`
	ProjectID   string                 `json:"project_id"`
	UserID      string                 `json:"user_id"`
	Role        string                 `json:"role"` // owner, admin, member, viewer
	Permissions map[string]interface{} `json:"permissions"`
	InvitedBy   *string                `json:"invited_by,omitempty"`
	InvitedAt   *time.Time             `json:"invited_at,omitempty"`
	JoinedAt    *time.Time             `json:"joined_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// UserRole represents user roles in a project
type UserRole string

const (
	RoleOwner  UserRole = "owner"
	RoleAdmin  UserRole = "admin"
	RoleMember UserRole = "member"
	RoleViewer UserRole = "viewer"
)

// AuthProvider represents OAuth providers
type AuthProvider string

const (
	ProviderGitHub AuthProvider = "github"
	ProviderGoogle AuthProvider = "google"
)

// EmailPasswordAuthRequest represents email/password authentication request
type EmailPasswordAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest represents user registration request
type RegisterRequest struct {
	Email     string  `json:"email"`
	Password  string  `json:"password"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

// PasswordResetRequest represents password reset request
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetConfirmRequest represents password reset confirmation
type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// EmailVerificationRequest represents email verification request
type EmailVerificationRequest struct {
	Token string `json:"token"`
}

// OAuthAuthRequest represents OAuth authentication request
type OAuthAuthRequest struct {
	Code     string `json:"code"`
	Provider string `json:"provider"`
}

// AuthResponse represents the response from authentication
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// ContextQuery represents a request for context restoration or requirement clarification
type ContextQuery struct {
	RepoID int64  `json:"repo_id"`
	Query  string `json:"query"`
	Mode   string `json:"mode"` // "restore" or "clarify"
}

// ContextResponse represents the response from the AI service
type ContextResponse struct {
	ClarifiedGoal string                 `json:"clarified_goal,omitempty"`
	Tasks         []Task                 `json:"tasks,omitempty"`
	Questions     []string               `json:"questions,omitempty"`
	PRScaffold    *PRScaffold            `json:"pr_scaffold,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// Task represents a clarified task from the AI service
type Task struct {
	Title      string `json:"title"`
	Acceptance string `json:"acceptance"`
}

// PRScaffold represents a suggested PR structure
type PRScaffold struct {
	Branch string `json:"branch"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// FilteredRepoData represents repository data filtered for AI service
type FilteredRepoData struct {
	Repo         string        `json:"repo"`
	Query        string        `json:"query"`
	Context      RepoContext   `json:"context"`
}

// RepoContext holds the filtered repository context
type RepoContext struct {
	PullRequests []PullRequest `json:"pull_requests"`
	Issues       []Issue       `json:"issues"`
	Commits      []Commit      `json:"commits"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Knowledge Graph Models

// KnowledgeEntity represents a knowledge entity in the graph
type KnowledgeEntity struct {
	ID              string                 `json:"id"`
	EntityType      string                 `json:"entity_type"`
	EntityID        string                 `json:"entity_id"`
	Title           string                 `json:"title"`
	Content         string                 `json:"content"`
	Metadata        map[string]interface{} `json:"metadata"`
	PlatformSource  *string                `json:"platform_source"`
	SourceEventIDs  StringList             `json:"source_event_ids"`
	Participants    StringList             `json:"participants"`
	Embedding       []float32              `json:"embedding,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// KnowledgeRelationship represents a relationship between knowledge entities
type KnowledgeRelationship struct {
	ID               string                 `json:"id"`
	SourceEntityID   string                 `json:"source_entity_id"`
	TargetEntityID   string                 `json:"target_entity_id"`
	RelationshipType string                 `json:"relationship_type"`
	Strength         float64                `json:"strength"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
}

// DecisionRecord represents an engineering decision
type DecisionRecord struct {
	ID             string     `json:"id"`
	EntityID       string     `json:"entity_id"`
	DecisionID     string     `json:"decision_id"`
	Title          string     `json:"title"`
	Decision       string     `json:"decision"`
	Rationale      *string    `json:"rationale"`
	Alternatives   StringList `json:"alternatives"`
	Consequences   StringList `json:"consequences"`
	Status         string     `json:"status"`
	PlatformSource string     `json:"platform_source"`
	SourceEventIDs StringList `json:"source_event_ids"`
	Participants   StringList `json:"participants"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// DiscussionSummary represents a summarized conversation thread
type DiscussionSummary struct {
	ID                string     `json:"id"`
	EntityID          string     `json:"entity_id"`
	SummaryID         string     `json:"summary_id"`
	ThreadID          *string    `json:"thread_id"`
	Platform          string     `json:"platform"`
	Participants      StringList `json:"participants"`
	Summary           string     `json:"summary"`
	KeyPoints         StringList `json:"key_points"`
	ActionItems       StringList `json:"action_items"`
	FileReferences    StringList `json:"file_references"`
	FeatureReferences StringList `json:"feature_references"`
	CreatedAt         time.Time  `json:"created_at"`
}

// FeatureContext represents the development history of a feature
type FeatureContext struct {
	ID           string     `json:"id"`
	EntityID     string     `json:"entity_id"`
	FeatureID    string     `json:"feature_id"`
	FeatureName  string     `json:"feature_name"`
	Description  *string    `json:"description"`
	Status       string     `json:"status"`
	Contributors StringList `json:"contributors"`
	RelatedFiles StringList `json:"related_files"`
	Discussions  StringList `json:"discussions"`
	Decisions    StringList `json:"decisions"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// FileContextHistory represents the change and discussion history of a file
type FileContextHistory struct {
	ID                string                 `json:"id"`
	EntityID          string                 `json:"entity_id"`
	ContextID         string                 `json:"context_id"`
	FilePath          string                 `json:"file_path"`
	ChangeReason      *string                `json:"change_reason"`
	DiscussionContext string                 `json:"discussion_context"`
	RelatedDecisions  StringList             `json:"related_decisions"`
	Contributors      StringList             `json:"contributors"`
	PlatformSources   map[string]interface{} `json:"platform_sources"`
	CreatedAt         time.Time              `json:"created_at"`
}

// SearchResult represents a semantic search result
type SearchResult struct {
	Entity     KnowledgeEntity `json:"entity"`
	Similarity float64         `json:"similarity"`
	Rank       int             `json:"rank"`
}

// GraphTraversalResult represents a graph traversal result
type GraphTraversalResult struct {
	Path         []KnowledgeEntity      `json:"path"`
	Relationships []KnowledgeRelationship `json:"relationships"`
	Depth        int                    `json:"depth"`
	TotalStrength float64               `json:"total_strength"`
}

// KnowledgeGraphQuery represents a query for the knowledge graph
type KnowledgeGraphQuery struct {
	Query          string            `json:"query"`
	EntityTypes    []string          `json:"entity_types,omitempty"`
	Platforms      []string          `json:"platforms,omitempty"`
	Participants   []string          `json:"participants,omitempty"`
	DateRange      *DateRange        `json:"date_range,omitempty"`
	Limit          int               `json:"limit,omitempty"`
	IncludeContent bool              `json:"include_content,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DateRange represents a date range filter
type DateRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// ProjectIntegration represents a platform integration for a project
type ProjectIntegration struct {
	ID                string                 `json:"id"`
	ProjectID         string                 `json:"project_id"`
	Platform          string                 `json:"platform"` // github, slack, discord
	IntegrationType   string                 `json:"integration_type"` // oauth, bot, webhook
	Status            string                 `json:"status"` // active, inactive, error, pending
	Configuration     map[string]interface{} `json:"configuration"`
	Credentials       map[string]interface{} `json:"credentials"` // Encrypted storage
	LastSyncAt        *time.Time             `json:"last_sync_at,omitempty"`
	LastSyncStatus    *string                `json:"last_sync_status,omitempty"`
	ErrorMessage      *string                `json:"error_message,omitempty"`
	SyncCheckpoint    map[string]interface{} `json:"sync_checkpoint"` // Platform-specific sync state
	CreatedBy         string                 `json:"created_by"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// ProjectDataSource represents a data source within a project integration
type ProjectDataSource struct {
	ID              string                 `json:"id"`
	ProjectID       string                 `json:"project_id"`
	IntegrationID   string                 `json:"integration_id"`
	SourceType      string                 `json:"source_type"` // repository, channel, server
	SourceID        string                 `json:"source_id"` // Platform-specific ID
	SourceName      string                 `json:"source_name"`
	Configuration   map[string]interface{} `json:"configuration"`
	IsActive        bool                   `json:"is_active"`
	LastIngestionAt *time.Time             `json:"last_ingestion_at,omitempty"`
	IngestionStatus *string                `json:"ingestion_status,omitempty"`
	ErrorMessage    *string                `json:"error_message,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// IntegrationStatus represents the status of an integration
type IntegrationStatus string

const (
	IntegrationStatusActive   IntegrationStatus = "active"
	IntegrationStatusInactive IntegrationStatus = "inactive"
	IntegrationStatusError    IntegrationStatus = "error"
	IntegrationStatusPending  IntegrationStatus = "pending"
)

// IntegrationType represents the type of integration
type IntegrationType string

const (
	IntegrationTypeOAuth   IntegrationType = "oauth"
	IntegrationTypeBot     IntegrationType = "bot"
	IntegrationTypeWebhook IntegrationType = "webhook"
)

// Platform represents supported platforms
type Platform string

const (
	PlatformGitHub  Platform = "github"
	PlatformSlack   Platform = "slack"
	PlatformDiscord Platform = "discord"
)

// SourceType represents the type of data source
type SourceType string

const (
	SourceTypeRepository SourceType = "repository"
	SourceTypeChannel    SourceType = "channel"
	SourceTypeServer     SourceType = "server"
)

// JSONBMap is a custom type for handling JSONB map storage
type JSONBMap map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (jm JSONBMap) Value() (driver.Value, error) {
	if jm == nil {
		return nil, nil
	}
	return json.Marshal(jm)
}

// Scan implements the sql.Scanner interface for database retrieval
func (jm *JSONBMap) Scan(value interface{}) error {
	if value == nil {
		*jm = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONBMap", value)
	}

	return json.Unmarshal(bytes, jm)
}