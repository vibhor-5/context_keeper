package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// **Property 17: MCP Tool Implementation Completeness**
// **Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.5**
//
// This property verifies that all MCP tools:
// 1. Handle valid inputs correctly and return structured responses
// 2. Validate required parameters and return appropriate errors
// 3. Return responses in the correct MCP format
// 4. Apply response optimization when content is large
// 5. Maintain consistent behavior across different input types
func TestProperty_MCPToolImplementationCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup test environment
		server := setupTestMCPServer(t)
		ctx := context.Background()

		// Generate test data
		toolName := rapid.SampledFrom([]string{
			"search_project_knowledge",
			"get_context_for_file", 
			"get_decision_history",
			"list_recent_architecture_discussions",
			"explain_why_code_exists",
		}).Draw(t, "tool_name")

		// Generate appropriate arguments for each tool
		var arguments map[string]interface{}
		var expectedRequiredParams []string

		switch toolName {
		case "search_project_knowledge":
			query := rapid.StringMatching(`[a-zA-Z0-9\s]{1,100}`).Draw(t, "query")
			arguments = map[string]interface{}{
				"query": query,
				"limit": rapid.IntRange(1, 50).Draw(t, "limit"),
			}
			expectedRequiredParams = []string{"query"}

		case "get_context_for_file":
			filePath := rapid.StringMatching(`[a-zA-Z0-9/_.-]{1,200}`).Draw(t, "file_path")
			arguments = map[string]interface{}{
				"file_path": filePath,
				"include_history": rapid.Bool().Draw(t, "include_history"),
			}
			expectedRequiredParams = []string{"file_path"}

		case "get_decision_history":
			target := rapid.StringMatching(`[a-zA-Z0-9/_.-]{1,100}`).Draw(t, "target")
			arguments = map[string]interface{}{
				"target": target,
				"limit": rapid.IntRange(1, 100).Draw(t, "limit"),
			}
			expectedRequiredParams = []string{"target"}

		case "list_recent_architecture_discussions":
			arguments = map[string]interface{}{
				"limit": rapid.IntRange(1, 50).Draw(t, "limit"),
				"days_back": rapid.IntRange(1, 365).Draw(t, "days_back"),
			}
			expectedRequiredParams = []string{} // No required params

		case "explain_why_code_exists":
			filePath := rapid.StringMatching(`[a-zA-Z0-9/_.-]{1,200}`).Draw(t, "file_path")
			arguments = map[string]interface{}{
				"file_path": filePath,
			}
			expectedRequiredParams = []string{"file_path"}
		}

		// Test 1: Valid inputs should return successful responses
		result, err := server.callTool(ctx, toolName, arguments)
		
		// Should not return Go errors (business logic errors are in result.IsError)
		assert.NoError(t, err, "Tool execution should not return Go errors")
		assert.NotNil(t, result, "Result should not be nil")
		
		// Response should have proper structure
		assert.NotNil(t, result.Content, "Content should not be nil")
		assert.Greater(t, len(result.Content), 0, "Content should not be empty")
		
		// First content item should be text
		assert.Equal(t, "text", result.Content[0].Type, "First content should be text type")
		assert.NotEmpty(t, result.Content[0].Text, "Text content should not be empty")

		// Test 2: Missing required parameters should return errors
		for _, requiredParam := range expectedRequiredParams {
			invalidArgs := make(map[string]interface{})
			for k, v := range arguments {
				if k != requiredParam {
					invalidArgs[k] = v
				}
			}
			
			invalidResult, err := server.callTool(ctx, toolName, invalidArgs)
			assert.NoError(t, err, "Should not return Go error for missing params")
			assert.True(t, invalidResult.IsError, "Should return business error for missing required param: %s", requiredParam)
			assert.Contains(t, strings.ToLower(invalidResult.Content[0].Text), "error", "Error message should contain 'error'")
			assert.Contains(t, strings.ToLower(invalidResult.Content[0].Text), strings.ToLower(requiredParam), "Error should mention missing parameter")
		}

		// Test 3: Response optimization should be applied for large content
		// This is tested indirectly by checking that responses are reasonable in size
		if !result.IsError {
			contentLength := len(result.Content[0].Text)
			// Responses should be optimized to reasonable sizes (not exceeding ~50KB)
			assert.Less(t, contentLength, 50000, "Response should be optimized for large content")
		}

		// Test 4: Response format consistency
		if !result.IsError {
			text := result.Content[0].Text
			
			// All successful responses should start with a header
			assert.True(t, strings.HasPrefix(text, "#"), "Response should start with markdown header")
			
			// Should contain structured information
			switch toolName {
			case "search_project_knowledge":
				assert.Contains(t, text, "Search Results", "Search results should contain results header")
				assert.Contains(t, text, "Found", "Should indicate number of results found")
				
			case "get_context_for_file":
				assert.Contains(t, text, "Context for File", "Should contain file context header")
				
			case "get_decision_history":
				assert.Contains(t, text, "Decision History", "Should contain decision history header")
				
			case "list_recent_architecture_discussions":
				assert.Contains(t, text, "Architecture Discussions", "Should contain discussions header")
				
			case "explain_why_code_exists":
				assert.Contains(t, text, "Why", "Should contain explanation header")
				assert.Contains(t, text, "Exists", "Should contain 'exists' in header")
			}
		}
	})
}

// Test helper to set up MCP server with mock dependencies
func setupTestMCPServer(t *rapid.T) *MCPServer {
	// Create mock logger
	logger := &SimpleLogger{}
	
	// Create mock knowledge graph service
	mockKG := &MockKnowledgeGraphService{}
	
	// Create response optimizer
	optimizer := NewResponseOptimizer(1000, logger) // Small token limit for testing
	
	// Create MCP server
	server := &MCPServer{
		knowledgeGraph:    mockKG,
		responseOptimizer: optimizer,
		logger:           logger,
	}
	
	return server
}

// Mock knowledge graph service for testing
type MockKnowledgeGraphService struct{}

// Implement the interface that KnowledgeGraphService provides
func (m *MockKnowledgeGraphService) SearchKnowledge(ctx context.Context, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error) {
	// Return mock search results
	results := []models.SearchResult{
		{
			Entity: models.KnowledgeEntity{
				ID:             "1",
				Title:          "Mock Search Result",
				EntityType:     "discussion",
				Content:        "This is mock content for testing search functionality.",
				PlatformSource: stringPtr("github"),
				Participants:   []string{"user1", "user2"},
				CreatedAt:      time.Now(),
			},
			Similarity: 0.85,
		},
	}
	
	// Limit results based on query
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	
	return results, nil
}

func (m *MockKnowledgeGraphService) GetContextForFile(ctx context.Context, filePath string) (*FileContextResponse, error) {
	return &FileContextResponse{
		FileContexts: []models.FileContextHistory{
			{
				ID:                "1",
				FilePath:          filePath,
				DiscussionContext: "Mock discussion context for file",
				ChangeReason:      stringPtr("Added new feature"),
				Contributors:      []string{"developer1"},
				RelatedDecisions:  []string{"decision1"},
				CreatedAt:         time.Now(),
			},
		},
		RelatedEntities: []models.SearchResult{
			{
				Entity: models.KnowledgeEntity{
					ID:         "2",
					Title:      "Related Entity",
					EntityType: "feature",
					Content:    "Related feature content",
					CreatedAt:  time.Now(),
				},
				Similarity: 0.75,
			},
		},
		RelatedDecisions: []models.SearchResult{
			{
				Entity: models.KnowledgeEntity{
					ID:         "3",
					Title:      "Related Decision",
					EntityType: "decision",
					Content:    "Decision content",
					CreatedAt:  time.Now(),
				},
				Similarity: 0.80,
			},
		},
	}, nil
}

func (m *MockKnowledgeGraphService) GetDecisionHistory(ctx context.Context, target string) (*DecisionHistoryResponse, error) {
	return &DecisionHistoryResponse{
		Decisions: []models.DecisionRecord{
			{
				ID:             "1",
				Title:          "Mock Decision",
				Decision:       "We decided to use this approach",
				Rationale:      stringPtr("It provides better performance"),
				Status:         "accepted",
				PlatformSource: "github",
				Alternatives:   []string{"Alternative 1", "Alternative 2"},
				Consequences:   []string{"Consequence 1", "Consequence 2"},
				Participants:   []string{"architect1", "developer1"},
				CreatedAt:      time.Now(),
			},
		},
	}, nil
}

func (m *MockKnowledgeGraphService) GetRecentArchitectureDiscussions(ctx context.Context, limit int) ([]models.DiscussionSummary, error) {
	discussions := []models.DiscussionSummary{
		{
			ID:             "1",
			Platform:       "slack",
			ThreadID:       stringPtr("thread123"),
			Summary:        "Discussion about system architecture",
			KeyPoints:      []string{"Point 1", "Point 2"},
			ActionItems:    []string{"Action 1", "Action 2"},
			Participants:   []string{"architect1", "developer1"},
			FileReferences: []string{"file1.go", "file2.go"},
			CreatedAt:      time.Now(),
		},
	}
	
	if limit > 0 && len(discussions) > limit {
		discussions = discussions[:limit]
	}
	
	return discussions, nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}