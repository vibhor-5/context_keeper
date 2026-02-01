package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Repository represents a GitHub repository
type Repository struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	FullName  string    `json:"full_name"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ID          string `json:"id"`
	Login       string `json:"login"`
	Email       string `json:"email"`
	GitHubToken string `json:"-"` // Never serialize the token
}

// AuthResponse represents the response from GitHub OAuth
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