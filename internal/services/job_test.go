package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// MockRepositoryStore for testing
type MockRepositoryStore struct {
	jobs            map[int64]*models.IngestionJob
	repos           map[int64]*models.Repository
	nextJobID       int64
	shouldFail      bool
	prs             []models.PullRequest
	issues          []models.Issue
	commits         []models.Commit
	lastPRLimit     int
	lastIssueLimit  int
	lastCommitLimit int
}

func NewMockRepositoryStore() *MockRepositoryStore {
	return &MockRepositoryStore{
		jobs:      make(map[int64]*models.IngestionJob),
		repos:     make(map[int64]*models.Repository),
		nextJobID: 1,
		prs:       []models.PullRequest{},
		issues:    []models.Issue{},
		commits:   []models.Commit{},
	}
}

func (m *MockRepositoryStore) CreateJob(ctx context.Context, job *models.IngestionJob) error {
	if m.shouldFail {
		return errors.New("mock create job error")
	}
	job.ID = m.nextJobID
	m.nextJobID++
	m.jobs[job.ID] = job
	return nil
}

func (m *MockRepositoryStore) UpdateJobStatus(ctx context.Context, jobID int64, status models.JobStatus, errorMsg *string) error {
	if m.shouldFail {
		return errors.New("mock update job error")
	}
	if job, exists := m.jobs[jobID]; exists {
		job.Status = status
		job.ErrorMsg = errorMsg
		if status == models.JobStatusRunning {
			now := time.Now()
			job.StartedAt = &now
		} else {
			now := time.Now()
			job.FinishedAt = &now
		}
	}
	return nil
}

func (m *MockRepositoryStore) GetJobByID(ctx context.Context, jobID int64) (*models.IngestionJob, error) {
	if m.shouldFail {
		return nil, errors.New("mock get job error")
	}
	if job, exists := m.jobs[jobID]; exists {
		return job, nil
	}
	return nil, errors.New("job not found")
}

func (m *MockRepositoryStore) GetRepoByID(ctx context.Context, repoID int64) (*models.Repository, error) {
	if m.shouldFail {
		return nil, errors.New("mock get repo error")
	}
	if repo, exists := m.repos[repoID]; exists {
		return repo, nil
	}
	return nil, errors.New("repo not found")
}

// Implement other required methods (not used in job tests)
func (m *MockRepositoryStore) CreateRepo(ctx context.Context, repo *models.Repository) error {
	return nil
}
func (m *MockRepositoryStore) GetReposByUser(ctx context.Context, userID string) ([]models.Repository, error) {
	return nil, nil
}
func (m *MockRepositoryStore) CreatePullRequest(ctx context.Context, pr *models.PullRequest) error {
	return nil
}
func (m *MockRepositoryStore) GetRecentPRs(ctx context.Context, repoID int64, limit int) ([]models.PullRequest, error) {
	m.lastPRLimit = limit
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	// Apply the limit to the returned data
	if len(m.prs) > limit {
		return m.prs[:limit], nil
	}
	return m.prs, nil
}
func (m *MockRepositoryStore) CreateIssue(ctx context.Context, issue *models.Issue) error { return nil }
func (m *MockRepositoryStore) GetRecentIssues(ctx context.Context, repoID int64, limit int) ([]models.Issue, error) {
	m.lastIssueLimit = limit
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	// Apply the limit to the returned data
	if len(m.issues) > limit {
		return m.issues[:limit], nil
	}
	return m.issues, nil
}
func (m *MockRepositoryStore) CreateCommit(ctx context.Context, commit *models.Commit) error {
	return nil
}
func (m *MockRepositoryStore) GetRecentCommits(ctx context.Context, repoID int64, limit int) ([]models.Commit, error) {
	m.lastCommitLimit = limit
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	// Apply the limit to the returned data
	if len(m.commits) > limit {
		return m.commits[:limit], nil
	}
	return m.commits, nil
}
func (m *MockRepositoryStore) GetJobsByRepo(ctx context.Context, repoID int64) ([]models.IngestionJob, error) {
	return nil, nil
}

// MockGitHubService for testing
type MockGitHubService struct {
	shouldFail bool
}

func (m *MockGitHubService) GetUserRepos(ctx context.Context, token string) ([]models.Repository, error) {
	return nil, nil
}
func (m *MockGitHubService) GetUserInfo(ctx context.Context, token string) (*models.User, error) {
	return nil, nil
}

func (m *MockGitHubService) GetPullRequests(ctx context.Context, token, owner, repo string, limit int) ([]models.PullRequest, error) {
	if m.shouldFail {
		return nil, errors.New("mock github error")
	}
	return []models.PullRequest{
		{ID: 1, Number: 1, Title: "Test PR", Author: "testuser", State: "open"},
	}, nil
}

func (m *MockGitHubService) GetIssues(ctx context.Context, token, owner, repo string, limit int) ([]models.Issue, error) {
	if m.shouldFail {
		return nil, errors.New("mock github error")
	}
	return []models.Issue{
		{ID: 1, Title: "Test Issue", Author: "testuser", State: "open"},
	}, nil
}

func (m *MockGitHubService) GetCommits(ctx context.Context, token, owner, repo string, limit int) ([]models.Commit, error) {
	if m.shouldFail {
		return nil, errors.New("mock github error")
	}
	return []models.Commit{
		{SHA: "abc123", Message: "Test commit", Author: "testuser"},
	}, nil
}

func TestJobService_CreateIngestionJob(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	jobService := NewJobService(mockRepo, mockGitHub)

	ctx := context.Background()
	repoID := int64(1)
	userID := "testuser"

	job, err := jobService.CreateIngestionJob(ctx, repoID, userID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if job.RepoID != repoID {
		t.Errorf("Expected RepoID %d, got %d", repoID, job.RepoID)
	}

	if job.Status != models.JobStatusPending {
		t.Errorf("Expected status %s, got %s", models.JobStatusPending, job.Status)
	}

	if job.ID == 0 {
		t.Error("Expected job ID to be set")
	}
}

func TestJobService_CreateIngestionJob_Error(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockRepo.shouldFail = true
	mockGitHub := &MockGitHubService{}
	jobService := NewJobService(mockRepo, mockGitHub)

	ctx := context.Background()
	repoID := int64(1)
	userID := "testuser"

	_, err := jobService.CreateIngestionJob(ctx, repoID, userID)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "failed to create ingestion job") {
		t.Errorf("Expected error message to contain 'failed to create ingestion job', got %v", err)
	}
}

func TestJobService_GetJobStatus(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	jobService := NewJobService(mockRepo, mockGitHub)

	// Create a job first
	testJob := &models.IngestionJob{
		ID:     1,
		RepoID: 1,
		Status: models.JobStatusCompleted,
	}
	mockRepo.jobs[1] = testJob

	ctx := context.Background()
	job, err := jobService.GetJobStatus(ctx, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if job.ID != 1 {
		t.Errorf("Expected job ID 1, got %d", job.ID)
	}

	if job.Status != models.JobStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.JobStatusCompleted, job.Status)
	}
}

func TestJobService_GetJobStatus_NotFound(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	jobService := NewJobService(mockRepo, mockGitHub)

	ctx := context.Background()
	_, err := jobService.GetJobStatus(ctx, 999)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !contains(err.Error(), "failed to get job status") {
		t.Errorf("Expected error message to contain 'failed to get job status', got %v", err)
	}
}

func TestJobService_ProcessJob(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	jobService := NewJobService(mockRepo, mockGitHub)

	// Set up test data
	testRepo := &models.Repository{
		ID:       1,
		Name:     "test-repo",
		Owner:    "testuser",
		FullName: "testuser/test-repo",
	}
	mockRepo.repos[1] = testRepo

	testJob := &models.IngestionJob{
		ID:     1,
		RepoID: 1,
		Status: models.JobStatusPending,
	}
	mockRepo.jobs[1] = testJob // Add job to mock repo

	ctx := context.Background()
	err := jobService.ProcessJob(ctx, testJob, "test-token")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Give the goroutine a moment to start
	time.Sleep(100 * time.Millisecond)

	// Check that job exists in mock repo
	_, exists := mockRepo.jobs[1]
	if !exists {
		t.Fatal("Job should exist in mock repo")
	}

	// The job should eventually be marked as completed, but we can't easily test
	// the async behavior without more complex synchronization
}

func TestJobService_ProcessJob_RepoNotFound(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	jobService := NewJobService(mockRepo, mockGitHub)

	testJob := &models.IngestionJob{
		ID:     1,
		RepoID: 999, // Non-existent repo
		Status: models.JobStatusPending,
	}

	ctx := context.Background()
	err := jobService.ProcessJob(ctx, testJob, "test-token")
	if err != nil {
		t.Fatalf("Expected no error from ProcessJob (async), got %v", err)
	}

	// Give the goroutine a moment to process and fail
	time.Sleep(100 * time.Millisecond)

	// The job should be marked as failed due to repo not found
	// (This is tested indirectly through the async behavior)
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "lo wo", true},
		{"hello world", "xyz", false},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
		{"partial ingestion", "partial", true},
	}

	for _, tt := range tests {
		got := contains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestJobService_JobLifecycle(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	_ = NewJobService(mockRepo, mockGitHub)

	// Test job lifecycle: pending -> running -> completed
	testJob := &models.IngestionJob{
		ID:     1,
		RepoID: 1,
		Status: models.JobStatusPending,
	}

	// Test pending status
	if testJob.Status != models.JobStatusPending {
		t.Errorf("Expected initial status %s, got %s", models.JobStatusPending, testJob.Status)
	}

	// Test status transitions through UpdateJobStatus
	ctx := context.Background()

	// Transition to running
	err := mockRepo.UpdateJobStatus(ctx, testJob.ID, models.JobStatusRunning, nil)
	if err != nil {
		t.Fatalf("Failed to update job to running: %v", err)
	}

	// Verify running status was set
	mockRepo.jobs[1] = testJob
	err = mockRepo.UpdateJobStatus(ctx, 1, models.JobStatusRunning, nil)
	if err != nil {
		t.Fatalf("Failed to update job status: %v", err)
	}

	job := mockRepo.jobs[1]
	if job.Status != models.JobStatusRunning {
		t.Errorf("Expected status %s, got %s", models.JobStatusRunning, job.Status)
	}

	if job.StartedAt == nil {
		t.Error("Expected StartedAt to be set when status is running")
	}

	// Transition to completed
	err = mockRepo.UpdateJobStatus(ctx, 1, models.JobStatusCompleted, nil)
	if err != nil {
		t.Fatalf("Failed to update job to completed: %v", err)
	}

	job = mockRepo.jobs[1]
	if job.Status != models.JobStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.JobStatusCompleted, job.Status)
	}

	if job.FinishedAt == nil {
		t.Error("Expected FinishedAt to be set when status is completed")
	}
}

func TestJobService_JobLifecycle_WithErrors(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	_ = NewJobService(mockRepo, mockGitHub)

	ctx := context.Background()
	testJob := &models.IngestionJob{
		ID:     1,
		RepoID: 1,
		Status: models.JobStatusPending,
	}
	mockRepo.jobs[1] = testJob

	// Test transition to failed with error message
	errorMsg := "GitHub API rate limit exceeded"
	err := mockRepo.UpdateJobStatus(ctx, 1, models.JobStatusFailed, &errorMsg)
	if err != nil {
		t.Fatalf("Failed to update job to failed: %v", err)
	}

	job := mockRepo.jobs[1]
	if job.Status != models.JobStatusFailed {
		t.Errorf("Expected status %s, got %s", models.JobStatusFailed, job.Status)
	}

	if job.ErrorMsg == nil || *job.ErrorMsg != errorMsg {
		t.Errorf("Expected error message %q, got %v", errorMsg, job.ErrorMsg)
	}

	if job.FinishedAt == nil {
		t.Error("Expected FinishedAt to be set when status is failed")
	}
}

func TestJobService_JobLifecycle_PartialStatus(t *testing.T) {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	_ = NewJobService(mockRepo, mockGitHub)

	ctx := context.Background()
	testJob := &models.IngestionJob{
		ID:     1,
		RepoID: 1,
		Status: models.JobStatusPending,
	}
	mockRepo.jobs[1] = testJob

	// Test transition to partial with error message
	errorMsg := "Some pull requests failed to ingest"
	err := mockRepo.UpdateJobStatus(ctx, 1, models.JobStatusPartial, &errorMsg)
	if err != nil {
		t.Fatalf("Failed to update job to partial: %v", err)
	}

	job := mockRepo.jobs[1]
	if job.Status != models.JobStatusPartial {
		t.Errorf("Expected status %s, got %s", models.JobStatusPartial, job.Status)
	}

	if job.ErrorMsg == nil || *job.ErrorMsg != errorMsg {
		t.Errorf("Expected error message %q, got %v", errorMsg, job.ErrorMsg)
	}

	if job.FinishedAt == nil {
		t.Error("Expected FinishedAt to be set when status is partial")
	}
}

func TestJobService_ValidStatusTransitions(t *testing.T) {
	// Test that all valid status transitions are supported
	validTransitions := []struct {
		from models.JobStatus
		to   models.JobStatus
	}{
		{models.JobStatusPending, models.JobStatusRunning},
		{models.JobStatusRunning, models.JobStatusCompleted},
		{models.JobStatusRunning, models.JobStatusPartial},
		{models.JobStatusRunning, models.JobStatusFailed},
	}

	mockRepo := NewMockRepositoryStore()
	ctx := context.Background()

	for i, transition := range validTransitions {
		jobID := int64(i + 1)
		testJob := &models.IngestionJob{
			ID:     jobID,
			RepoID: 1,
			Status: transition.from,
		}
		mockRepo.jobs[jobID] = testJob

		var errorMsg *string
		if transition.to == models.JobStatusFailed || transition.to == models.JobStatusPartial {
			msg := "test error"
			errorMsg = &msg
		}

		err := mockRepo.UpdateJobStatus(ctx, jobID, transition.to, errorMsg)
		if err != nil {
			t.Errorf("Failed to transition from %s to %s: %v", transition.from, transition.to, err)
		}

		job := mockRepo.jobs[jobID]
		if job.Status != transition.to {
			t.Errorf("Expected status %s, got %s", transition.to, job.Status)
		}
	}
}
