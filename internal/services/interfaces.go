package services

import (
	"context"
	"github.com/DevAnuragT/context_keeper/internal/models"
)

// AuthService handles GitHub OAuth and JWT operations
type AuthService interface {
	HandleGitHubCallback(ctx context.Context, code string) (*models.AuthResponse, error)
	ValidateJWT(token string) (*models.User, error)
	GenerateJWT(user *models.User) (string, error)
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
}

// RepositoryStore handles database operations
type RepositoryStore interface {
	// Repository operations
	CreateRepo(ctx context.Context, repo *models.Repository) error
	GetReposByUser(ctx context.Context, userID string) ([]models.Repository, error)
	GetRepoByID(ctx context.Context, repoID int64) (*models.Repository, error)
	
	// Pull request operations
	CreatePullRequest(ctx context.Context, pr *models.PullRequest) error
	GetRecentPRs(ctx context.Context, repoID int64, limit int) ([]models.PullRequest, error)
	
	// Issue operations
	CreateIssue(ctx context.Context, issue *models.Issue) error
	GetRecentIssues(ctx context.Context, repoID int64, limit int) ([]models.Issue, error)
	
	// Commit operations
	CreateCommit(ctx context.Context, commit *models.Commit) error
	GetRecentCommits(ctx context.Context, repoID int64, limit int) ([]models.Commit, error)
	
	// Job operations
	CreateJob(ctx context.Context, job *models.IngestionJob) error
	UpdateJobStatus(ctx context.Context, jobID int64, status models.JobStatus, errorMsg *string) error
	GetJobByID(ctx context.Context, jobID int64) (*models.IngestionJob, error)
	GetJobsByRepo(ctx context.Context, repoID int64) ([]models.IngestionJob, error)
}