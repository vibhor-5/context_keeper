package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// ContextServiceImpl implements the ContextService interface
type ContextServiceImpl struct {
	repo           RepositoryStore
	permissionSvc  PermissionService
	aiBaseURL      string
	client         *http.Client
}

// NewContextService creates a new context service instance
func NewContextService(repo RepositoryStore, permissionSvc PermissionService, aiBaseURL string) ContextService {
	return &ContextServiceImpl{
		repo:          repo,
		permissionSvc: permissionSvc,
		aiBaseURL:     aiBaseURL,
		client: &http.Client{
			Timeout: 30 * time.Second, // 30-second timeout as per requirements
		},
	}
}

// ProcessQuery processes a context query by filtering repository data and sending it to the AI service
func (c *ContextServiceImpl) ProcessQuery(ctx context.Context, repoID int64, query, mode string) (*models.ContextResponse, error) {
	// Filter repository data for AI service
	repoData, err := c.FilterRepoData(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to filter repository data: %w", err)
	}

	// Get repository information for context
	repo, err := c.repo.GetRepoByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Prepare request payload for AI service
	aiRequest := models.FilteredRepoData{
		Repo:    repo.FullName,
		Query:   query,
		Context: *repoData,
	}

	// Send request to AI service
	response, err := c.callAIService(ctx, aiRequest, mode)
	if err != nil {
		return nil, fmt.Errorf("AI service call failed: %w", err)
	}

	return response, nil
}

// FilterRepoData filters repository data to the most recent items for AI service
func (c *ContextServiceImpl) FilterRepoData(ctx context.Context, repoID int64) (*models.RepoContext, error) {
	// Get the most recent 10 pull requests
	prs, err := c.repo.GetRecentPRs(ctx, repoID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent pull requests: %w", err)
	}

	// Get the most recent 10 issues
	issues, err := c.repo.GetRecentIssues(ctx, repoID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent issues: %w", err)
	}

	// Get the most recent 20 commits
	commits, err := c.repo.GetRecentCommits(ctx, repoID, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}

	return &models.RepoContext{
		PullRequests: prs,
		Issues:       issues,
		Commits:      commits,
	}, nil
}

// callAIService makes an HTTP request to the AI service with timeout handling
func (c *ContextServiceImpl) callAIService(ctx context.Context, request models.FilteredRepoData, mode string) (*models.ContextResponse, error) {
	// Prepare the request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Determine the AI service endpoint based on mode
	endpoint := c.getAIEndpoint(mode)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Make the request with timeout
	resp, err := c.client.Do(req)
	if err != nil {
		// Check if this is a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("AI service timeout: request exceeded 30 seconds")
		}
		return nil, fmt.Errorf("AI service request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var aiResponse models.ContextResponse
	err = json.Unmarshal(responseBody, &aiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &aiResponse, nil
}

// ProcessQueryByProject processes a context query for all repositories in a project
func (c *ContextServiceImpl) ProcessQueryByProject(ctx context.Context, projectID string, query, mode string) (*models.ContextResponse, error) {
	// Filter project data for AI service
	projectData, err := c.FilterProjectData(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to filter project data: %w", err)
	}

	// Get project information for context
	project, err := c.repo.GetProjectWorkspace(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project workspace: %w", err)
	}

	// Prepare request payload for AI service
	aiRequest := models.FilteredRepoData{
		Repo:    fmt.Sprintf("Project: %s", project.Name),
		Query:   query,
		Context: *projectData,
	}

	// Send request to AI service
	response, err := c.callAIService(ctx, aiRequest, mode)
	if err != nil {
		return nil, fmt.Errorf("AI service call failed: %w", err)
	}

	return response, nil
}

// FilterProjectData filters project data to the most recent items for AI service
func (c *ContextServiceImpl) FilterProjectData(ctx context.Context, projectID string) (*models.RepoContext, error) {
	// Get the most recent 10 pull requests across all project repositories
	prs, err := c.repo.GetRecentPRsByProject(ctx, projectID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent pull requests: %w", err)
	}

	// Get the most recent 10 issues across all project repositories
	issues, err := c.repo.GetRecentIssuesByProject(ctx, projectID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent issues: %w", err)
	}

	// Get the most recent 20 commits across all project repositories
	commits, err := c.repo.GetRecentCommitsByProject(ctx, projectID, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}

	return &models.RepoContext{
		PullRequests: prs,
		Issues:       issues,
		Commits:      commits,
	}, nil
}

// getAIEndpoint returns the appropriate AI service endpoint based on mode
func (c *ContextServiceImpl) getAIEndpoint(mode string) string {
	switch mode {
	case "restore":
		return c.aiBaseURL + "/context/restore"
	case "clarify":
		return c.aiBaseURL + "/context/clarify"
	default:
		return c.aiBaseURL + "/context/query"
	}
}

// AIServiceError represents an error from the AI service
type AIServiceError struct {
	StatusCode int
	Message    string
	IsTimeout  bool
}

func (e *AIServiceError) Error() string {
	if e.IsTimeout {
		return fmt.Sprintf("AI service timeout: %s", e.Message)
	}
	return fmt.Sprintf("AI service error (status %d): %s", e.StatusCode, e.Message)
}

// NewAIServiceError creates a new AI service error
func NewAIServiceError(statusCode int, message string, isTimeout bool) *AIServiceError {
	return &AIServiceError{
		StatusCode: statusCode,
		Message:    message,
		IsTimeout:  isTimeout,
	}
}
