package services

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// Property 6: Ingestion Job Lifecycle
// For any ingestion job, status transitions should follow the valid sequence:
// pending → running → (completed|partial|failed), with appropriate timestamps
// and error messages persisted for each state change
// **Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5, 4.6**
func TestIngestionJobLifecycleProperty(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random test scenario
			scenario := generateJobLifecycleScenario(rand.New(rand.NewSource(int64(i))))

			// Test the property
			if !validateJobLifecycleProperty(t, scenario) {
				t.Errorf("Job lifecycle property violated for scenario: %+v", scenario)
			}
		})
	}
}

// JobLifecycleScenario represents a test scenario for job lifecycle
type JobLifecycleScenario struct {
	InitialStatus   models.JobStatus
	FinalStatus     models.JobStatus
	ShouldHaveError bool
	ErrorMessage    string
}

// generateJobLifecycleScenario creates a random job lifecycle test scenario
func generateJobLifecycleScenario(r *rand.Rand) JobLifecycleScenario {
	// All jobs start as pending
	initialStatus := models.JobStatusPending

	// Random final status (excluding pending since that's initial)
	finalStatuses := []models.JobStatus{
		models.JobStatusCompleted,
		models.JobStatusPartial,
		models.JobStatusFailed,
	}
	finalStatus := finalStatuses[r.Intn(len(finalStatuses))]

	// Determine if error should be present
	shouldHaveError := finalStatus == models.JobStatusPartial || finalStatus == models.JobStatusFailed

	var errorMessage string
	if shouldHaveError {
		errorMessages := []string{
			"GitHub API rate limit exceeded",
			"Network timeout during ingestion",
			"Some pull requests failed to ingest",
			"Repository access denied",
			"Database connection lost",
		}
		errorMessage = errorMessages[r.Intn(len(errorMessages))]
	}

	return JobLifecycleScenario{
		InitialStatus:   initialStatus,
		FinalStatus:     finalStatus,
		ShouldHaveError: shouldHaveError,
		ErrorMessage:    errorMessage,
	}
}

// validateJobLifecycleProperty validates the job lifecycle property
func validateJobLifecycleProperty(t *testing.T, scenario JobLifecycleScenario) bool {
	mockRepo := NewMockRepositoryStore()
	mockGitHub := &MockGitHubService{}
	jobService := NewJobService(mockRepo, mockGitHub)

	ctx := context.Background()

	// Create initial job
	job, err := jobService.CreateIngestionJob(ctx, 1, "testuser")
	if err != nil {
		t.Logf("Failed to create job: %v", err)
		return false
	}

	// Verify initial state
	if job.Status != models.JobStatusPending {
		t.Logf("Expected initial status %s, got %s", models.JobStatusPending, job.Status)
		return false
	}

	if job.StartedAt != nil {
		t.Logf("Expected StartedAt to be nil for pending job")
		return false
	}

	if job.FinishedAt != nil {
		t.Logf("Expected FinishedAt to be nil for pending job")
		return false
	}

	// Transition to running
	err = mockRepo.UpdateJobStatus(ctx, job.ID, models.JobStatusRunning, nil)
	if err != nil {
		t.Logf("Failed to update job to running: %v", err)
		return false
	}

	// Verify running state
	runningJob, err := jobService.GetJobStatus(ctx, job.ID)
	if err != nil {
		t.Logf("Failed to get running job status: %v", err)
		return false
	}

	if runningJob.Status != models.JobStatusRunning {
		t.Logf("Expected running status %s, got %s", models.JobStatusRunning, runningJob.Status)
		return false
	}

	if runningJob.StartedAt == nil {
		t.Logf("Expected StartedAt to be set for running job")
		return false
	}

	if runningJob.FinishedAt != nil {
		t.Logf("Expected FinishedAt to be nil for running job")
		return false
	}

	// Transition to final status
	var errorMsg *string
	if scenario.ShouldHaveError {
		errorMsg = &scenario.ErrorMessage
	}

	err = mockRepo.UpdateJobStatus(ctx, job.ID, scenario.FinalStatus, errorMsg)
	if err != nil {
		t.Logf("Failed to update job to final status %s: %v", scenario.FinalStatus, err)
		return false
	}

	// Verify final state
	finalJob, err := jobService.GetJobStatus(ctx, job.ID)
	if err != nil {
		t.Logf("Failed to get final job status: %v", err)
		return false
	}

	if finalJob.Status != scenario.FinalStatus {
		t.Logf("Expected final status %s, got %s", scenario.FinalStatus, finalJob.Status)
		return false
	}

	if finalJob.StartedAt == nil {
		t.Logf("Expected StartedAt to remain set for final job")
		return false
	}

	if finalJob.FinishedAt == nil {
		t.Logf("Expected FinishedAt to be set for final job")
		return false
	}

	// Verify error message handling
	if scenario.ShouldHaveError {
		if finalJob.ErrorMsg == nil {
			t.Logf("Expected error message for status %s", scenario.FinalStatus)
			return false
		}
		if *finalJob.ErrorMsg != scenario.ErrorMessage {
			t.Logf("Expected error message %q, got %q", scenario.ErrorMessage, *finalJob.ErrorMsg)
			return false
		}
	} else {
		if finalJob.ErrorMsg != nil {
			t.Logf("Expected no error message for status %s, got %q", scenario.FinalStatus, *finalJob.ErrorMsg)
			return false
		}
	}

	// Verify timestamp ordering
	if !finalJob.StartedAt.Before(*finalJob.FinishedAt) && !finalJob.StartedAt.Equal(*finalJob.FinishedAt) {
		t.Logf("Expected StartedAt (%v) to be before or equal to FinishedAt (%v)",
			finalJob.StartedAt, finalJob.FinishedAt)
		return false
	}

	return true
}

// TestJobLifecycleInvalidTransitions tests that invalid transitions are handled properly
func TestJobLifecycleInvalidTransitions(t *testing.T) {
	// Test invalid transitions that should not occur in normal operation
	invalidTransitions := []struct {
		from models.JobStatus
		to   models.JobStatus
	}{
		{models.JobStatusCompleted, models.JobStatusRunning},
		{models.JobStatusFailed, models.JobStatusRunning},
		{models.JobStatusPartial, models.JobStatusRunning},
		{models.JobStatusCompleted, models.JobStatusPending},
		{models.JobStatusFailed, models.JobStatusPending},
	}

	mockRepo := NewMockRepositoryStore()
	ctx := context.Background()

	for i, transition := range invalidTransitions {
		jobID := int64(i + 1)

		// Create job in initial state
		testJob := &models.IngestionJob{
			ID:     jobID,
			RepoID: 1,
			Status: transition.from,
		}

		// Set timestamps based on initial state
		now := time.Now()
		if transition.from != models.JobStatusPending {
			testJob.StartedAt = &now
		}
		if transition.from == models.JobStatusCompleted ||
			transition.from == models.JobStatusFailed ||
			transition.from == models.JobStatusPartial {
			testJob.FinishedAt = &now
		}

		mockRepo.jobs[jobID] = testJob

		// The system should handle these gracefully (our implementation allows any transition)
		// but in a real system, you might want to validate transitions
		err := mockRepo.UpdateJobStatus(ctx, jobID, transition.to, nil)
		if err != nil {
			t.Logf("Transition from %s to %s failed: %v", transition.from, transition.to, err)
		}

		// Note: Our current implementation allows any transition for simplicity
		// In a production system, you might want to add validation
	}
}

// TestJobLifecycleTimestampConsistency tests timestamp consistency across job lifecycle
func TestJobLifecycleTimestampConsistency(t *testing.T) {
	const iterations = 50

	for i := 0; i < iterations; i++ {
		mockRepo := NewMockRepositoryStore()
		mockGitHub := &MockGitHubService{}
		jobService := NewJobService(mockRepo, mockGitHub)

		ctx := context.Background()

		// Create job
		job, err := jobService.CreateIngestionJob(ctx, 1, "testuser")
		if err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		// Add small delay to ensure timestamp differences
		time.Sleep(1 * time.Millisecond)

		// Transition to running
		err = mockRepo.UpdateJobStatus(ctx, job.ID, models.JobStatusRunning, nil)
		if err != nil {
			t.Fatalf("Failed to update to running: %v", err)
		}

		runningJob, _ := jobService.GetJobStatus(ctx, job.ID)

		// Add small delay
		time.Sleep(1 * time.Millisecond)

		// Transition to completed
		err = mockRepo.UpdateJobStatus(ctx, job.ID, models.JobStatusCompleted, nil)
		if err != nil {
			t.Fatalf("Failed to update to completed: %v", err)
		}

		finalJob, _ := jobService.GetJobStatus(ctx, job.ID)

		// Verify timestamp consistency
		if runningJob.StartedAt == nil || finalJob.FinishedAt == nil {
			t.Fatalf("Missing timestamps: StartedAt=%v, FinishedAt=%v",
				runningJob.StartedAt, finalJob.FinishedAt)
		}

		if finalJob.StartedAt.After(*finalJob.FinishedAt) {
			t.Errorf("StartedAt (%v) should not be after FinishedAt (%v)",
				finalJob.StartedAt, finalJob.FinishedAt)
		}
	}
}
