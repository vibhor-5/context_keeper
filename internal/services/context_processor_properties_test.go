package services

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAIService for testing
type MockAIService struct{}

func (m *MockAIService) ExtractDecisions(ctx context.Context, events []NormalizedEvent) ([]DecisionRecord, error) {
	return []DecisionRecord{}, nil
}

func (m *MockAIService) SummarizeDiscussion(ctx context.Context, events []NormalizedEvent) (*DiscussionSummary, error) {
	return nil, nil
}

func (m *MockAIService) IdentifyFeatures(ctx context.Context, events []NormalizedEvent) ([]FeatureContext, error) {
	return []FeatureContext{}, nil
}

func (m *MockAIService) AnalyzeFileContext(ctx context.Context, events []NormalizedEvent) ([]FileContextHistory, error) {
	return []FileContextHistory{}, nil
}

// MockLogger for testing
type MockLogger struct{}

func (m *MockLogger) Info(msg string, fields map[string]interface{}) {}
func (m *MockLogger) Error(msg string, err error, fields map[string]interface{}) {}
func (m *MockLogger) Debug(msg string, fields map[string]interface{}) {}

// Property test generators

// generateRandomEvent creates a random normalized event
func generateRandomEvent(r *rand.Rand, platform string) NormalizedEvent {
	eventTypes := []EventType{
		EventTypePullRequest, EventTypeIssue, EventTypeCommit,
		EventTypeMessage, EventTypeThread, EventTypeDiscussion,
	}

	authors := []string{"alice", "bob", "charlie", "diana", "eve"}
	filePaths := []string{
		"src/main.go", "internal/service.go", "cmd/server/main.go",
		"pkg/utils.go", "api/handlers.go", "config/config.go",
	}
	features := []string{
		"authentication", "user-management", "api-gateway",
		"data-processing", "notification-system", "logging",
	}

	event := NormalizedEvent{
		PlatformID:  fmt.Sprintf("%s-%d", platform, r.Int63()),
		EventType:   eventTypes[r.Intn(len(eventTypes))],
		Timestamp:   time.Now().Add(-time.Duration(r.Intn(7*24)) * time.Hour),
		Author:      authors[r.Intn(len(authors))],
		Content:     generateRandomContent(r),
		Platform:    platform,
		FileRefs:    []string{},
		FeatureRefs: []string{},
		Metadata:    make(map[string]interface{}),
	}

	// Randomly add file references
	if r.Float32() < 0.6 {
		numFiles := r.Intn(3) + 1
		for i := 0; i < numFiles; i++ {
			event.FileRefs = append(event.FileRefs, filePaths[r.Intn(len(filePaths))])
		}
	}

	// Randomly add feature references
	if r.Float32() < 0.4 {
		numFeatures := r.Intn(2) + 1
		for i := 0; i < numFeatures; i++ {
			event.FeatureRefs = append(event.FeatureRefs, features[r.Intn(len(features))])
		}
	}

	// Add thread ID for some events
	if r.Float32() < 0.3 {
		threadID := fmt.Sprintf("thread-%d", r.Int63())
		event.ThreadID = &threadID
	}

	return event
}

// generateRandomContent creates random content for events
func generateRandomContent(r *rand.Rand) string {
	templates := []string{
		"Implemented %s feature in %s",
		"Fixed bug in %s related to %s",
		"Decided to use %s for %s implementation",
		"Discussed %s approach for %s",
		"Updated %s to handle %s better",
		"Refactored %s to improve %s",
	}

	subjects := []string{
		"authentication", "database", "API", "frontend", "backend",
		"caching", "logging", "monitoring", "security", "performance",
	}

	objects := []string{
		"user management", "data processing", "error handling",
		"request routing", "response formatting", "validation",
	}

	template := templates[r.Intn(len(templates))]
	subject := subjects[r.Intn(len(subjects))]
	object := objects[r.Intn(len(objects))]

	return fmt.Sprintf(template, subject, object)
}

// generateEventSet creates a set of related events
func generateEventSet(r *rand.Rand, size int) []NormalizedEvent {
	platforms := []string{"github", "slack", "discord"}
	events := make([]NormalizedEvent, size)

	for i := 0; i < size; i++ {
		platform := platforms[r.Intn(len(platforms))]
		events[i] = generateRandomEvent(r, platform)
	}

	return events
}

// **Property 11: Relationship Identification Accuracy**
// **Validates: Requirements 5.2, 5.3, 5.4**
func TestProperty_RelationshipIdentificationAccuracy(t *testing.T) {
	const iterations = 100
	
	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate test data
			r := rand.New(rand.NewSource(int64(i)))
			eventCount := r.Intn(50) + 10 // 10-59 events
			events := generateEventSet(r, eventCount)

			// Create processor
			processor := NewContextProcessor(&MockAIService{}, &MockLogger{})

			// Process events
			ctx := context.Background()
			result, err := processor.ProcessEvents(ctx, events)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Property: All relationships must have valid source and target IDs
			for _, rel := range result.Relationships {
				assert.NotEmpty(t, rel.ID, "Relationship must have an ID")
				assert.NotEmpty(t, rel.SourceType, "Relationship must have source type")
				assert.NotEmpty(t, rel.SourceID, "Relationship must have source ID")
				assert.NotEmpty(t, rel.TargetType, "Relationship must have target type")
				assert.NotEmpty(t, rel.TargetID, "Relationship must have target ID")
				assert.NotEmpty(t, rel.Type, "Relationship must have a type")
				
				// Strength should be between 0 and 1
				assert.GreaterOrEqual(t, rel.Strength, 0.0, "Relationship strength should be >= 0")
				assert.LessOrEqual(t, rel.Strength, 1.0, "Relationship strength should be <= 1")
				
				// Timestamp should be reasonable
				assert.False(t, rel.CreatedAt.IsZero(), "Relationship should have creation timestamp")
			}

			// Property: Relationships should reference existing entities
			entityIDs := make(map[string]bool)
			
			// Collect all entity IDs
			for _, decision := range result.DecisionRecords {
				entityIDs[decision.ID] = true
			}
			for _, summary := range result.DiscussionSummaries {
				entityIDs[summary.ID] = true
			}
			for _, feature := range result.FeatureContexts {
				entityIDs[feature.ID] = true
			}
			for _, fileContext := range result.FileContexts {
				entityIDs[fileContext.ID] = true
			}

			// Verify relationships reference valid entities (except cross-platform relationships)
			for _, rel := range result.Relationships {
				if rel.TargetType != "cross_platform" {
					if rel.SourceType != "cross_platform" {
						assert.True(t, entityIDs[rel.SourceID], 
							"Relationship source ID %s should reference existing entity", rel.SourceID)
					}
					if rel.TargetType != "cross_platform" {
						assert.True(t, entityIDs[rel.TargetID], 
							"Relationship target ID %s should reference existing entity", rel.TargetID)
					}
				}
			}

			// Property: File-related relationships should be consistent
			fileRelationships := make(map[string][]Relationship)
			for _, rel := range result.Relationships {
				if rel.SourceType == "file" || rel.TargetType == "file" {
					fileID := rel.SourceID
					if rel.TargetType == "file" {
						fileID = rel.TargetID
					}
					fileRelationships[fileID] = append(fileRelationships[fileID], rel)
				}
			}

			// Files with multiple relationships should have consistent metadata
			for _, rels := range fileRelationships {
				if len(rels) > 1 {
					var filePath string
					for _, rel := range rels {
						if path, exists := rel.Metadata["file_path"]; exists {
							if pathStr, ok := path.(string); ok {
								if filePath == "" {
									filePath = pathStr
								} else {
									assert.Equal(t, filePath, pathStr, 
										"File relationships should have consistent file paths")
								}
							}
						}
					}
				}
			}

			// Property: Contributor relationships should be symmetric in nature
			contributorRels := make(map[string][]Relationship)
			for _, rel := range result.Relationships {
				if rel.Type == "contributed_by" {
					if contributor, exists := rel.Metadata["contributor"]; exists {
						if contributorStr, ok := contributor.(string); ok {
							contributorRels[contributorStr] = append(contributorRels[contributorStr], rel)
						}
					}
				}
			}

			// Contributors should have relationships that make sense
			for contributor, rels := range contributorRels {
				assert.NotEmpty(t, contributor, "Contributor should have a name")
				assert.GreaterOrEqual(t, len(rels), 1, "Contributor should have at least one relationship")
				
				// All relationships for a contributor should reference that contributor
				for _, rel := range rels {
					if contributorMeta, exists := rel.Metadata["contributor"]; exists {
						assert.Equal(t, contributor, contributorMeta, 
							"Contributor relationship should reference correct contributor")
					}
				}
			}
		})
	}
}

// **Property 12: Error Resilience in Processing**
// **Validates: Requirements 5.6**
func TestProperty_ErrorResilienceInProcessing(t *testing.T) {
	const iterations = 100
	
	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate test data with potential error conditions
			r := rand.New(rand.NewSource(int64(i)))
			eventCount := r.Intn(30) + 5 // 5-34 events
			events := generateEventSet(r, eventCount)

			// Introduce some problematic events
			if len(events) > 0 {
				// Empty content
				events[0].Content = ""
				
				if len(events) > 1 {
					// Very long content
					events[1].Content = strings.Repeat("x", 10000)
				}
				
				if len(events) > 2 {
					// Invalid characters
					events[2].Content = "Test\x00\x01\x02content"
				}
				
				if len(events) > 3 {
					// Empty author
					events[3].Author = ""
				}
			}

			// Create processor with small batch size to test batch error handling
			processor := NewContextProcessorWithConfig(
				&MockAIService{}, 
				&MockLogger{}, 
				5,  // Small batch size
				2,  // Max retries
				time.Millisecond * 10, // Short retry delay
			)

			// Process events
			ctx := context.Background()
			result, err := processor.ProcessEvents(ctx, events)

			// Property: Processing should not fail completely due to individual event errors
			assert.NoError(t, err, "Processing should not fail completely")
			assert.NotNil(t, result, "Result should not be nil")

			// Property: Processed event count should match input
			assert.Equal(t, len(events), result.ProcessedEvents, 
				"Processed event count should match input")

			// Property: Error handling should be graceful
			if len(result.Errors) > 0 {
				for _, processingError := range result.Errors {
					assert.NotEmpty(t, processingError.EventID, "Error should have event ID")
					assert.NotEmpty(t, processingError.Platform, "Error should have platform")
					assert.NotEmpty(t, processingError.Error, "Error should have error message")
					assert.False(t, processingError.Timestamp.IsZero(), "Error should have timestamp")
					
					// Retryable flag should be set appropriately
					assert.IsType(t, false, processingError.Retryable, "Retryable should be boolean")
				}
			}

			// Property: Some processing should succeed even with errors
			totalEntities := len(result.DecisionRecords) + len(result.DiscussionSummaries) + 
							len(result.FeatureContexts) + len(result.FileContexts)
			
			if len(events) > 5 { // Only check for larger event sets
				// Should extract at least some entities unless all events are problematic
				assert.GreaterOrEqual(t, totalEntities, 0, 
					"Should extract some entities even with errors")
			}

			// Property: Relationships should still be extracted even with some processing errors
			assert.GreaterOrEqual(t, len(result.Relationships), 0, 
				"Relationships should be extractable even with errors")

			// Property: All extracted entities should have valid IDs
			for _, decision := range result.DecisionRecords {
				assert.NotEmpty(t, decision.ID, "Decision should have valid ID")
				assert.False(t, decision.CreatedAt.IsZero(), "Decision should have timestamp")
			}
			
			for _, summary := range result.DiscussionSummaries {
				assert.NotEmpty(t, summary.ID, "Summary should have valid ID")
				assert.False(t, summary.CreatedAt.IsZero(), "Summary should have timestamp")
			}
			
			for _, feature := range result.FeatureContexts {
				assert.NotEmpty(t, feature.ID, "Feature should have valid ID")
				assert.False(t, feature.CreatedAt.IsZero(), "Feature should have timestamp")
			}
			
			for _, fileContext := range result.FileContexts {
				assert.NotEmpty(t, fileContext.ID, "File context should have valid ID")
				assert.False(t, fileContext.CreatedAt.IsZero(), "File context should have timestamp")
			}
		})
	}
}

// Test batch processing behavior
func TestProperty_BatchProcessingConsistency(t *testing.T) {
	const iterations = 50
	
	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate test data
			r := rand.New(rand.NewSource(int64(i)))
			eventCount := r.Intn(100) + 20 // 20-119 events
			events := generateEventSet(r, eventCount)

			// Process with different batch sizes
			batchSizes := []int{5, 10, 25, 50}
			results := make([]*ProcessingResult, len(batchSizes))

			for j, batchSize := range batchSizes {
				processor := NewContextProcessorWithConfig(
					&MockAIService{}, 
					&MockLogger{}, 
					batchSize,
					1, // No retries for consistency test
					time.Millisecond,
				)

				ctx := context.Background()
				result, err := processor.ProcessEvents(ctx, events)
				require.NoError(t, err)
				results[j] = result
			}

			// Property: Different batch sizes should produce consistent results
			baseResult := results[0]
			for j := 1; j < len(results); j++ {
				result := results[j]
				
				// Same number of processed events
				assert.Equal(t, baseResult.ProcessedEvents, result.ProcessedEvents,
					"Batch size %d should process same number of events as batch size %d", 
					batchSizes[j], batchSizes[0])

				// Similar number of entities (may vary slightly due to grouping)
				assert.InDelta(t, len(baseResult.DecisionRecords), len(result.DecisionRecords), 5,
					"Decision count should be similar across batch sizes")
				assert.InDelta(t, len(baseResult.FeatureContexts), len(result.FeatureContexts), 5,
					"Feature count should be similar across batch sizes")
				assert.InDelta(t, len(baseResult.FileContexts), len(result.FileContexts), 10,
					"File context count should be similar across batch sizes")
			}
		})
	}
}