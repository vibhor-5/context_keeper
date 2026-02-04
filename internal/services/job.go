package services

import (
	"context"
	"fmt"
	"log"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// JobServiceImpl implements the JobService interface
type JobServiceImpl struct {
	repo   RepositoryStore
	github GitHubService
}

// NewJobService creates a new job service instance
func NewJobService(repo RepositoryStore, github GitHubService) JobService {
	return &JobServiceImpl{
		repo:   repo,
		github: github,
	}
}

// CreateIngestionJob creates a new ingestion job with pending status
func (j *JobServiceImpl) CreateIngestionJob(ctx context.Context, repoID int64, userID string) (*models.IngestionJob, error) {
	job := &models.IngestionJob{
		RepoID: repoID,
		Status: models.JobStatusPending,
	}

	err := j.repo.CreateJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("failed to create ingestion job: %w", err)
	}

	return job, nil
}

// GetJobStatus retrieves the current status of an ingestion job
func (j *JobServiceImpl) GetJobStatus(ctx context.Context, jobID int64) (*models.IngestionJob, error) {
	job, err := j.repo.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job status: %w", err)
	}

	return job, nil
}

// ProcessJob processes an ingestion job in the background using goroutines
func (j *JobServiceImpl) ProcessJob(ctx context.Context, job *models.IngestionJob, githubToken string) error {
	// Start processing in a goroutine
	go func() {
		// Create a new context for the background job to avoid cancellation
		jobCtx := context.Background()

		// Update job status to running
		err := j.repo.UpdateJobStatus(jobCtx, job.ID, models.JobStatusRunning, nil)
		if err != nil {
			log.Printf("Failed to update job %d status to running: %v", job.ID, err)
			return
		}

		// Get repository information
		repo, err := j.repo.GetRepoByID(jobCtx, job.RepoID)
		if err != nil {
			j.handleJobError(jobCtx, job.ID, fmt.Errorf("failed to get repository: %w", err))
			return
		}

		// Process the ingestion
		err = j.ingestRepositoryData(jobCtx, repo, githubToken)
		if err != nil {
			j.handleJobError(jobCtx, job.ID, err)
			return
		}

		// Mark job as completed
		err = j.repo.UpdateJobStatus(jobCtx, job.ID, models.JobStatusCompleted, nil)
		if err != nil {
			log.Printf("Failed to update job %d status to completed: %v", job.ID, err)
		}
	}()

	return nil
}

// ingestRepositoryData performs the actual data ingestion
func (j *JobServiceImpl) ingestRepositoryData(ctx context.Context, repo *models.Repository, githubToken string) error {
	var hasErrors bool
	var errorMessages []string

	// Extract owner and repo name from full_name (format: "owner/repo")
	owner := repo.Owner
	repoName := repo.Name

	// Ingest pull requests (limit 50)
	prs, err := j.github.GetPullRequests(ctx, githubToken, owner, repoName, 50)
	if err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, fmt.Sprintf("failed to fetch pull requests: %v", err))
		log.Printf("Error fetching PRs for repo %d: %v", repo.ID, err)
	} else {
		for _, pr := range prs {
			pr.RepoID = repo.ID
			err := j.repo.CreatePullRequest(ctx, &pr)
			if err != nil {
				hasErrors = true
				errorMessages = append(errorMessages, fmt.Sprintf("failed to store PR %d: %v", pr.Number, err))
				log.Printf("Error storing PR %d for repo %d: %v", pr.Number, repo.ID, err)
			}
		}
	}

	// Ingest issues (limit 50)
	issues, err := j.github.GetIssues(ctx, githubToken, owner, repoName, 50)
	if err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, fmt.Sprintf("failed to fetch issues: %v", err))
		log.Printf("Error fetching issues for repo %d: %v", repo.ID, err)
	} else {
		for _, issue := range issues {
			issue.RepoID = repo.ID
			err := j.repo.CreateIssue(ctx, &issue)
			if err != nil {
				hasErrors = true
				errorMessages = append(errorMessages, fmt.Sprintf("failed to store issue %d: %v", issue.ID, err))
				log.Printf("Error storing issue %d for repo %d: %v", issue.ID, repo.ID, err)
			}
		}
	}

	// Ingest commits (limit 100)
	commits, err := j.github.GetCommits(ctx, githubToken, owner, repoName, 100)
	if err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, fmt.Sprintf("failed to fetch commits: %v", err))
		log.Printf("Error fetching commits for repo %d: %v", repo.ID, err)
	} else {
		for _, commit := range commits {
			commit.RepoID = repo.ID
			err := j.repo.CreateCommit(ctx, &commit)
			if err != nil {
				hasErrors = true
				errorMessages = append(errorMessages, fmt.Sprintf("failed to store commit %s: %v", commit.SHA, err))
				log.Printf("Error storing commit %s for repo %d: %v", commit.SHA, repo.ID, err)
			}
		}
	}

	// If there were errors but some data was processed, mark as partial
	if hasErrors {
		return fmt.Errorf("partial ingestion completed with errors: %v", errorMessages)
	}

	return nil
}

// handleJobError handles job processing errors and updates job status
func (j *JobServiceImpl) handleJobError(ctx context.Context, jobID int64, err error) {
	errorMsg := err.Error()

	// Determine if this is a partial or complete failure
	status := models.JobStatusFailed
	if contains(errorMsg, "partial") {
		status = models.JobStatusPartial
	}

	updateErr := j.repo.UpdateJobStatus(ctx, jobID, status, &errorMsg)
	if updateErr != nil {
		log.Printf("Failed to update job %d error status: %v", jobID, updateErr)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr))))
}

// containsAt checks if string contains substring at any position
func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
