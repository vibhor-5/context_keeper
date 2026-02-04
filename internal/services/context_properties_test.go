package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// Property 7: AI Context Payload Filtering
// For any context query, the system should send exactly the most recent 10 pull requests,
// 10 issues, and 20 commits to the AI service, without performing semantic relevance filtering
// **Validates: Requirements 6.1, 6.2, 6.3, 6.7, 10.4, 10.5, 10.6**
func TestAIContextPayloadFilteringProperty(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random test scenario
			scenario := generateContextFilteringScenario(rand.New(rand.NewSource(int64(i))))

			// Test the property
			if !validateContextFilteringProperty(t, scenario) {
				t.Errorf("AI context payload filtering property violated for scenario: %+v", scenario)
			}
		})
	}
}

// ContextFilteringScenario represents a test scenario for context filtering
type ContextFilteringScenario struct {
	NumPRs     int
	NumIssues  int
	NumCommits int
	Query      string
	Mode       string
}

// generateContextFilteringScenario creates a random context filtering test scenario
func generateContextFilteringScenario(r *rand.Rand) ContextFilteringScenario {
	// Generate random numbers of items (some below, at, and above limits)
	numPRs := r.Intn(50) + 1      // 1-50 PRs
	numIssues := r.Intn(50) + 1   // 1-50 issues
	numCommits := r.Intn(100) + 1 // 1-100 commits

	// Random query and mode
	queries := []string{
		"Implement user authentication",
		"Fix database connection issue",
		"Add API endpoint for user management",
		"Optimize query performance",
		"Update documentation",
	}
	query := queries[r.Intn(len(queries))]

	modes := []string{"restore", "clarify", "query"}
	mode := modes[r.Intn(len(modes))]

	return ContextFilteringScenario{
		NumPRs:     numPRs,
		NumIssues:  numIssues,
		NumCommits: numCommits,
		Query:      query,
		Mode:       mode,
	}
}

// validateContextFilteringProperty validates the context filtering property
func validateContextFilteringProperty(t *testing.T, scenario ContextFilteringScenario) bool {
	var receivedRequest models.FilteredRepoData

	// Create a mock AI service server that captures the request
	mockAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and parse the request body
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedRequest)

		// Return mock response
		response := models.ContextResponse{
			ClarifiedGoal: "Test response",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockAIServer.Close()

	mockRepo := NewMockRepositoryStore()
	contextService := NewContextService(mockRepo, mockAIServer.URL)

	// Set up test repository
	repoID := int64(1)
	testRepo := &models.Repository{
		ID:       repoID,
		Name:     "test-repo",
		FullName: "testuser/test-repo",
		Owner:    "testuser",
	}
	mockRepo.repos[repoID] = testRepo

	// Generate test data
	mockRepo.prs = generateTestPRs(scenario.NumPRs)
	mockRepo.issues = generateTestIssues(scenario.NumIssues)
	mockRepo.commits = generateTestCommits(scenario.NumCommits)

	ctx := context.Background()
	_, err := contextService.ProcessQuery(ctx, repoID, scenario.Query, scenario.Mode)
	if err != nil {
		t.Logf("ProcessQuery failed: %v", err)
		return false
	}

	// Validate Property 7: Exact filtering limits
	expectedPRs := min(scenario.NumPRs, 10)
	expectedIssues := min(scenario.NumIssues, 10)
	expectedCommits := min(scenario.NumCommits, 20)

	if len(receivedRequest.Context.PullRequests) != expectedPRs {
		t.Logf("Expected %d PRs in AI request, got %d", expectedPRs, len(receivedRequest.Context.PullRequests))
		return false
	}

	if len(receivedRequest.Context.Issues) != expectedIssues {
		t.Logf("Expected %d issues in AI request, got %d", expectedIssues, len(receivedRequest.Context.Issues))
		return false
	}

	if len(receivedRequest.Context.Commits) != expectedCommits {
		t.Logf("Expected %d commits in AI request, got %d", expectedCommits, len(receivedRequest.Context.Commits))
		return false
	}

	// Validate that query is passed through unchanged (no semantic filtering)
	if receivedRequest.Query != scenario.Query {
		t.Logf("Expected query %q, got %q", scenario.Query, receivedRequest.Query)
		return false
	}

	// Validate that repository name is included
	if receivedRequest.Repo != "testuser/test-repo" {
		t.Logf("Expected repo 'testuser/test-repo', got %q", receivedRequest.Repo)
		return false
	}

	// Validate that all data is sent without semantic relevance filtering
	// (i.e., the most recent items are sent, not filtered by relevance)
	if scenario.NumPRs > 0 && len(receivedRequest.Context.PullRequests) > 0 {
		// First PR should be the most recent (ID 1 in our test data)
		if receivedRequest.Context.PullRequests[0].ID != 1 {
			t.Logf("Expected most recent PR (ID 1) first, got ID %d", receivedRequest.Context.PullRequests[0].ID)
			return false
		}
	}

	return true
}

// Property 8: AI Service Timeout Enforcement
// For any AI service call, if the response takes longer than 30 seconds,
// the system should immediately return a timeout error without waiting further
// **Validates: Requirements 6.4, 6.5**
func TestAIServiceTimeoutEnforcementProperty(t *testing.T) {
	const iterations = 10 // Fewer iterations due to timeout testing

	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random test scenario
			scenario := generateTimeoutScenario(rand.New(rand.NewSource(int64(i))))

			// Test the property
			if !validateTimeoutProperty(t, scenario) {
				t.Errorf("AI service timeout property violated for scenario: %+v", scenario)
			}
		})
	}
}

// TimeoutScenario represents a test scenario for timeout testing
type TimeoutScenario struct {
	DelaySeconds  int
	ShouldTimeout bool
	Query         string
}

// generateTimeoutScenario creates a random timeout test scenario
func generateTimeoutScenario(r *rand.Rand) TimeoutScenario {
	// Generate delays both below and above the timeout threshold
	delaySeconds := []int{1, 2, 5, 6}[r.Intn(4)] // Either 1-2 seconds (should succeed) or 5-6 seconds (should timeout)
	shouldTimeout := delaySeconds > 3            // Timeout if delay > 3 seconds (we use 3s timeout)

	queries := []string{
		"Test timeout query",
		"Another timeout test",
		"Timeout scenario query",
	}
	query := queries[r.Intn(len(queries))]

	return TimeoutScenario{
		DelaySeconds:  delaySeconds,
		ShouldTimeout: shouldTimeout,
		Query:         query,
	}
}

// validateTimeoutProperty validates the timeout enforcement property
func validateTimeoutProperty(t *testing.T, scenario TimeoutScenario) bool {
	// Create a mock AI service server with configurable delay
	mockAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(scenario.DelaySeconds) * time.Second)

		response := models.ContextResponse{
			ClarifiedGoal: "Test response",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockAIServer.Close()

	mockRepo := NewMockRepositoryStore()
	// Create context service with shorter timeout for testing
	contextService := &ContextServiceImpl{
		repo:      mockRepo,
		aiBaseURL: mockAIServer.URL,
		client: &http.Client{
			Timeout: 3 * time.Second, // 3-second timeout for testing
		},
	}

	// Set up test repository
	repoID := int64(1)
	testRepo := &models.Repository{
		ID:       repoID,
		Name:     "test-repo",
		FullName: "testuser/test-repo",
		Owner:    "testuser",
	}
	mockRepo.repos[repoID] = testRepo

	ctx := context.Background()
	startTime := time.Now()
	_, err := contextService.ProcessQuery(ctx, repoID, scenario.Query, "query")
	duration := time.Since(startTime)

	if scenario.ShouldTimeout {
		// Should have timed out
		if err == nil {
			t.Logf("Expected timeout error for %d second delay, got nil", scenario.DelaySeconds)
			return false
		}

		// Should have failed quickly (within timeout + small buffer)
		if duration > 4*time.Second {
			t.Logf("Timeout took too long: %v (expected < 4s)", duration)
			return false
		}

		// Error should indicate timeout
		if !contains(err.Error(), "timeout") && !contains(err.Error(), "context deadline exceeded") {
			t.Logf("Expected timeout error, got: %v", err)
			return false
		}
	} else {
		// Should have succeeded
		if err != nil {
			t.Logf("Expected success for %d second delay, got error: %v", scenario.DelaySeconds, err)
			return false
		}
	}

	return true
}

// Helper functions for generating test data
func generateTestPRs(count int) []models.PullRequest {
	prs := make([]models.PullRequest, count)
	for i := 0; i < count; i++ {
		prs[i] = models.PullRequest{
			ID:        int64(i + 1),
			Number:    i + 1,
			Title:     fmt.Sprintf("PR %d", i+1),
			Author:    fmt.Sprintf("user%d", i+1),
			State:     "open",
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour), // Most recent first
		}
	}
	return prs
}

func generateTestIssues(count int) []models.Issue {
	issues := make([]models.Issue, count)
	for i := 0; i < count; i++ {
		issues[i] = models.Issue{
			ID:        int64(i + 1),
			Title:     fmt.Sprintf("Issue %d", i+1),
			Author:    fmt.Sprintf("user%d", i+1),
			State:     "open",
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour), // Most recent first
		}
	}
	return issues
}

func generateTestCommits(count int) []models.Commit {
	commits := make([]models.Commit, count)
	for i := 0; i < count; i++ {
		commits[i] = models.Commit{
			SHA:       fmt.Sprintf("sha%d", i+1),
			Message:   fmt.Sprintf("Commit %d", i+1),
			Author:    fmt.Sprintf("user%d", i+1),
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour), // Most recent first
		}
	}
	return commits
}
