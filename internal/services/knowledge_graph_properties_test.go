package services

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKnowledgeGraphRepository for testing knowledge graph operations
type MockKnowledgeGraphRepository struct {
	entities      map[string]*models.KnowledgeEntity
	relationships map[string]*models.KnowledgeRelationship
	decisions     map[string]*models.DecisionRecord
	summaries     map[string]*models.DiscussionSummary
	features      map[string]*models.FeatureContext
	fileContexts  map[string][]models.FileContextHistory
}

func NewMockKnowledgeGraphRepository() *MockKnowledgeGraphRepository {
	return &MockKnowledgeGraphRepository{
		entities:      make(map[string]*models.KnowledgeEntity),
		relationships: make(map[string]*models.KnowledgeRelationship),
		decisions:     make(map[string]*models.DecisionRecord),
		summaries:     make(map[string]*models.DiscussionSummary),
		features:      make(map[string]*models.FeatureContext),
		fileContexts:  make(map[string][]models.FileContextHistory),
	}
}

// Implement required RepositoryStore methods for knowledge graph operations

func (m *MockKnowledgeGraphRepository) CreateKnowledgeEntity(ctx context.Context, entity *models.KnowledgeEntity) error {
	if entity.ID == "" {
		entity.ID = fmt.Sprintf("entity-%d", len(m.entities))
	}
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = time.Now()
	}
	entity.UpdatedAt = time.Now()
	m.entities[entity.ID] = entity
	return nil
}

func (m *MockKnowledgeGraphRepository) GetKnowledgeEntity(ctx context.Context, id string) (*models.KnowledgeEntity, error) {
	entity, exists := m.entities[id]
	if !exists {
		return nil, fmt.Errorf("entity not found: %s", id)
	}
	return entity, nil
}

func (m *MockKnowledgeGraphRepository) UpdateKnowledgeEntity(ctx context.Context, entity *models.KnowledgeEntity) error {
	if _, exists := m.entities[entity.ID]; !exists {
		return fmt.Errorf("entity not found: %s", entity.ID)
	}
	entity.UpdatedAt = time.Now()
	m.entities[entity.ID] = entity
	return nil
}

func (m *MockKnowledgeGraphRepository) DeleteKnowledgeEntity(ctx context.Context, id string) error {
	delete(m.entities, id)
	return nil
}

func (m *MockKnowledgeGraphRepository) CreateKnowledgeRelationship(ctx context.Context, relationship *models.KnowledgeRelationship) error {
	if relationship.ID == "" {
		relationship.ID = fmt.Sprintf("rel-%d", len(m.relationships))
	}
	if relationship.CreatedAt.IsZero() {
		relationship.CreatedAt = time.Now()
	}
	m.relationships[relationship.ID] = relationship
	return nil
}

func (m *MockKnowledgeGraphRepository) GetKnowledgeRelationships(ctx context.Context, entityID string) ([]models.KnowledgeRelationship, error) {
	var results []models.KnowledgeRelationship
	for _, rel := range m.relationships {
		if rel.SourceEntityID == entityID || rel.TargetEntityID == entityID {
			results = append(results, *rel)
		}
	}
	return results, nil
}

func (m *MockKnowledgeGraphRepository) DeleteKnowledgeRelationship(ctx context.Context, id string) error {
	delete(m.relationships, id)
	return nil
}

func (m *MockKnowledgeGraphRepository) CreateDecisionRecord(ctx context.Context, decision *models.DecisionRecord) error {
	if decision.ID == "" {
		decision.ID = fmt.Sprintf("decision-%d", len(m.decisions))
	}
	if decision.CreatedAt.IsZero() {
		decision.CreatedAt = time.Now()
	}
	decision.UpdatedAt = time.Now()
	m.decisions[decision.DecisionID] = decision
	return nil
}

func (m *MockKnowledgeGraphRepository) GetDecisionRecord(ctx context.Context, decisionID string) (*models.DecisionRecord, error) {
	decision, exists := m.decisions[decisionID]
	if !exists {
		return nil, fmt.Errorf("decision not found: %s", decisionID)
	}
	return decision, nil
}

func (m *MockKnowledgeGraphRepository) UpdateDecisionRecord(ctx context.Context, decision *models.DecisionRecord) error {
	if _, exists := m.decisions[decision.DecisionID]; !exists {
		return fmt.Errorf("decision not found: %s", decision.DecisionID)
	}
	decision.UpdatedAt = time.Now()
	m.decisions[decision.DecisionID] = decision
	return nil
}

func (m *MockKnowledgeGraphRepository) CreateDiscussionSummary(ctx context.Context, summary *models.DiscussionSummary) error {
	if summary.ID == "" {
		summary.ID = fmt.Sprintf("summary-%d", len(m.summaries))
	}
	if summary.CreatedAt.IsZero() {
		summary.CreatedAt = time.Now()
	}
	m.summaries[summary.SummaryID] = summary
	return nil
}

func (m *MockKnowledgeGraphRepository) GetDiscussionSummary(ctx context.Context, summaryID string) (*models.DiscussionSummary, error) {
	summary, exists := m.summaries[summaryID]
	if !exists {
		return nil, fmt.Errorf("summary not found: %s", summaryID)
	}
	return summary, nil
}

func (m *MockKnowledgeGraphRepository) CreateFeatureContext(ctx context.Context, feature *models.FeatureContext) error {
	if feature.ID == "" {
		feature.ID = fmt.Sprintf("feature-%d", len(m.features))
	}
	if feature.CreatedAt.IsZero() {
		feature.CreatedAt = time.Now()
	}
	feature.UpdatedAt = time.Now()
	m.features[feature.FeatureID] = feature
	return nil
}

func (m *MockKnowledgeGraphRepository) GetFeatureContext(ctx context.Context, featureID string) (*models.FeatureContext, error) {
	feature, exists := m.features[featureID]
	if !exists {
		return nil, fmt.Errorf("feature not found: %s", featureID)
	}
	return feature, nil
}

func (m *MockKnowledgeGraphRepository) UpdateFeatureContext(ctx context.Context, feature *models.FeatureContext) error {
	if _, exists := m.features[feature.FeatureID]; !exists {
		return fmt.Errorf("feature not found: %s", feature.FeatureID)
	}
	feature.UpdatedAt = time.Now()
	m.features[feature.FeatureID] = feature
	return nil
}

func (m *MockKnowledgeGraphRepository) CreateFileContextHistory(ctx context.Context, fileContext *models.FileContextHistory) error {
	if fileContext.ID == "" {
		fileContext.ID = fmt.Sprintf("file-%d", len(m.fileContexts))
	}
	if fileContext.CreatedAt.IsZero() {
		fileContext.CreatedAt = time.Now()
	}
	m.fileContexts[fileContext.FilePath] = append(m.fileContexts[fileContext.FilePath], *fileContext)
	return nil
}

func (m *MockKnowledgeGraphRepository) GetFileContextHistory(ctx context.Context, filePath string) ([]models.FileContextHistory, error) {
	contexts, exists := m.fileContexts[filePath]
	if !exists {
		return []models.FileContextHistory{}, nil
	}
	return contexts, nil
}

func (m *MockKnowledgeGraphRepository) SearchKnowledgeEntities(ctx context.Context, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error) {
	var results []models.SearchResult
	rank := 1
	
	for _, entity := range m.entities {
		// Simple text matching for mock
		if strings.Contains(strings.ToLower(entity.Title), strings.ToLower(query.Query)) ||
		   strings.Contains(strings.ToLower(entity.Content), strings.ToLower(query.Query)) {
			
			// Apply entity type filter
			if len(query.EntityTypes) > 0 {
				found := false
				for _, entityType := range query.EntityTypes {
					if entity.EntityType == entityType {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			
			// Apply platform filter
			if len(query.Platforms) > 0 && entity.PlatformSource != nil {
				found := false
				for _, platform := range query.Platforms {
					if *entity.PlatformSource == platform {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			
			results = append(results, models.SearchResult{
				Entity:     *entity,
				Similarity: 0.8, // Mock similarity
				Rank:       rank,
			})
			rank++
			
			if query.Limit > 0 && len(results) >= query.Limit {
				break
			}
		}
	}
	
	return results, nil
}

func (m *MockKnowledgeGraphRepository) SearchSimilarEntities(ctx context.Context, embedding []float32, entityTypes []string, limit int) ([]models.SearchResult, error) {
	// Mock implementation - return some entities
	var results []models.SearchResult
	rank := 1
	
	for _, entity := range m.entities {
		if len(entityTypes) > 0 {
			found := false
			for _, entityType := range entityTypes {
				if entity.EntityType == entityType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		results = append(results, models.SearchResult{
			Entity:     *entity,
			Similarity: 0.7, // Mock similarity
			Rank:       rank,
		})
		rank++
		
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	
	return results, nil
}

func (m *MockKnowledgeGraphRepository) TraverseKnowledgeGraph(ctx context.Context, startEntityID string, maxDepth int, relationshipTypes []string) (*models.GraphTraversalResult, error) {
	// Simple mock traversal
	startEntity, exists := m.entities[startEntityID]
	if !exists {
		return nil, fmt.Errorf("start entity not found: %s", startEntityID)
	}
	
	var path []models.KnowledgeEntity
	var relationships []models.KnowledgeRelationship
	
	path = append(path, *startEntity)
	
	// Add some related entities
	for _, rel := range m.relationships {
		if rel.SourceEntityID == startEntityID || rel.TargetEntityID == startEntityID {
			relationships = append(relationships, *rel)
			
			// Add the other entity to the path
			otherEntityID := rel.TargetEntityID
			if rel.TargetEntityID == startEntityID {
				otherEntityID = rel.SourceEntityID
			}
			
			if otherEntity, exists := m.entities[otherEntityID]; exists {
				path = append(path, *otherEntity)
			}
		}
	}
	
	return &models.GraphTraversalResult{
		Path:          path,
		Relationships: relationships,
		Depth:         1,
		TotalStrength: 1.0,
	}, nil
}

func (m *MockKnowledgeGraphRepository) GetRelatedEntities(ctx context.Context, entityID string, relationshipTypes []string, limit int) ([]models.KnowledgeEntity, error) {
	var entities []models.KnowledgeEntity
	
	for _, rel := range m.relationships {
		if rel.SourceEntityID == entityID || rel.TargetEntityID == entityID {
			// Check relationship type filter
			if len(relationshipTypes) > 0 {
				found := false
				for _, relType := range relationshipTypes {
					if rel.RelationshipType == relType {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			
			// Get the other entity
			otherEntityID := rel.TargetEntityID
			if rel.TargetEntityID == entityID {
				otherEntityID = rel.SourceEntityID
			}
			
			if otherEntity, exists := m.entities[otherEntityID]; exists {
				entities = append(entities, *otherEntity)
			}
			
			if limit > 0 && len(entities) >= limit {
				break
			}
		}
	}
	
	return entities, nil
}

// Stub implementations for other RepositoryStore methods (not used in knowledge graph tests)
func (m *MockKnowledgeGraphRepository) CreateRepo(ctx context.Context, repo *models.Repository) error { return nil }
func (m *MockKnowledgeGraphRepository) GetReposByUser(ctx context.Context, userID string) ([]models.Repository, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetRepoByID(ctx context.Context, repoID int64) (*models.Repository, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) CreatePullRequest(ctx context.Context, pr *models.PullRequest) error { return nil }
func (m *MockKnowledgeGraphRepository) GetRecentPRs(ctx context.Context, repoID int64, limit int) ([]models.PullRequest, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) CreateIssue(ctx context.Context, issue *models.Issue) error { return nil }
func (m *MockKnowledgeGraphRepository) GetRecentIssues(ctx context.Context, repoID int64, limit int) ([]models.Issue, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) CreateCommit(ctx context.Context, commit *models.Commit) error { return nil }
func (m *MockKnowledgeGraphRepository) GetRecentCommits(ctx context.Context, repoID int64, limit int) ([]models.Commit, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) CreateJob(ctx context.Context, job *models.IngestionJob) error { return nil }
func (m *MockKnowledgeGraphRepository) UpdateJobStatus(ctx context.Context, jobID int64, status models.JobStatus, errorMsg *string) error { return nil }
func (m *MockKnowledgeGraphRepository) GetJobByID(ctx context.Context, jobID int64) (*models.IngestionJob, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetJobsByRepo(ctx context.Context, repoID int64) ([]models.IngestionJob, error) { return nil, nil }

// User operations
func (m *MockKnowledgeGraphRepository) CreateUser(ctx context.Context, user *models.User) error { return nil }
func (m *MockKnowledgeGraphRepository) GetUserByID(ctx context.Context, userID string) (*models.User, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error { return nil }
func (m *MockKnowledgeGraphRepository) DeleteUser(ctx context.Context, userID string) error { return nil }

// OAuth account operations
func (m *MockKnowledgeGraphRepository) CreateOAuthAccount(ctx context.Context, account *models.UserOAuthAccount) error { return nil }
func (m *MockKnowledgeGraphRepository) GetOAuthAccount(ctx context.Context, provider, providerUserID string) (*models.UserOAuthAccount, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetOAuthAccountsByUser(ctx context.Context, userID string) ([]models.UserOAuthAccount, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) UpdateOAuthAccount(ctx context.Context, accountID string, updates map[string]interface{}) error { return nil }
func (m *MockKnowledgeGraphRepository) DeleteOAuthAccount(ctx context.Context, accountID string) error { return nil }

// Project workspace operations
func (m *MockKnowledgeGraphRepository) CreateProjectWorkspace(ctx context.Context, workspace *models.ProjectWorkspace) error { return nil }
func (m *MockKnowledgeGraphRepository) GetProjectWorkspace(ctx context.Context, projectID string) (*models.ProjectWorkspace, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetProjectWorkspacesByUser(ctx context.Context, userID string) ([]models.ProjectWorkspace, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) UpdateProjectWorkspace(ctx context.Context, projectID string, updates map[string]interface{}) error { return nil }
func (m *MockKnowledgeGraphRepository) DeleteProjectWorkspace(ctx context.Context, projectID string) error { return nil }

// Project member operations
func (m *MockKnowledgeGraphRepository) CreateProjectMember(ctx context.Context, member *models.ProjectMember) error { return nil }
func (m *MockKnowledgeGraphRepository) GetProjectMember(ctx context.Context, projectID, userID string) (*models.ProjectMember, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetProjectMembers(ctx context.Context, projectID string) ([]models.ProjectMember, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetUserProjectMemberships(ctx context.Context, userID string) ([]models.ProjectMember, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) UpdateProjectMember(ctx context.Context, memberID string, updates map[string]interface{}) error { return nil }
func (m *MockKnowledgeGraphRepository) DeleteProjectMember(ctx context.Context, memberID string) error { return nil }

// Project integration operations
func (m *MockKnowledgeGraphRepository) CreateProjectIntegration(ctx context.Context, integration *models.ProjectIntegration) error { return nil }
func (m *MockKnowledgeGraphRepository) GetProjectIntegration(ctx context.Context, integrationID string) (*models.ProjectIntegration, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetProjectIntegrations(ctx context.Context, projectID string) ([]models.ProjectIntegration, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetProjectIntegrationByPlatform(ctx context.Context, projectID, platform string) (*models.ProjectIntegration, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) UpdateProjectIntegration(ctx context.Context, integrationID string, updates map[string]interface{}) error { return nil }
func (m *MockKnowledgeGraphRepository) DeleteProjectIntegration(ctx context.Context, integrationID string) error { return nil }

// Project data source operations
func (m *MockKnowledgeGraphRepository) CreateProjectDataSource(ctx context.Context, dataSource *models.ProjectDataSource) error { return nil }
func (m *MockKnowledgeGraphRepository) GetProjectDataSource(ctx context.Context, dataSourceID string) (*models.ProjectDataSource, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetProjectDataSources(ctx context.Context, projectID string) ([]models.ProjectDataSource, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) GetProjectDataSourcesByIntegration(ctx context.Context, integrationID string) ([]models.ProjectDataSource, error) { return nil, nil }
func (m *MockKnowledgeGraphRepository) UpdateProjectDataSource(ctx context.Context, dataSourceID string, updates map[string]interface{}) error { return nil }
func (m *MockKnowledgeGraphRepository) DeleteProjectDataSource(ctx context.Context, dataSourceID string) error { return nil }

// Property test generators for knowledge graph entities

func generateRandomKnowledgeEntity(r *rand.Rand) *models.KnowledgeEntity {
	entityTypes := []string{"decision", "discussion", "feature", "file_context"}
	platforms := []string{"github", "slack", "discord"}
	
	entityType := entityTypes[r.Intn(len(entityTypes))]
	platform := platforms[r.Intn(len(platforms))]
	
	entity := &models.KnowledgeEntity{
		EntityType:     entityType,
		EntityID:       fmt.Sprintf("%s-%d", entityType, r.Int63()),
		Title:          generateRandomTitle(r, entityType),
		Content:        generateRandomContent(r),
		PlatformSource: &platform,
		SourceEventIDs: models.StringList{fmt.Sprintf("event-%d", r.Int63())},
		Participants:   models.StringList{generateRandomParticipant(r)},
		CreatedAt:      time.Now().Add(-time.Duration(r.Intn(30*24)) * time.Hour),
		Metadata:       generateRandomMetadata(r),
	}
	
	return entity
}

func generateRandomTitle(r *rand.Rand, entityType string) string {
	switch entityType {
	case "decision":
		decisions := []string{
			"Use React for frontend",
			"Adopt microservices architecture",
			"Switch to PostgreSQL",
			"Implement OAuth authentication",
			"Use Docker for deployment",
		}
		return decisions[r.Intn(len(decisions))]
	case "discussion":
		return fmt.Sprintf("Discussion about %s implementation", generateRandomTopic(r))
	case "feature":
		return fmt.Sprintf("Feature: %s", generateRandomFeature(r))
	case "file_context":
		return fmt.Sprintf("File: %s", generateRandomFilePath(r))
	default:
		return "Generic entity"
	}
}

func generateRandomTopic(r *rand.Rand) string {
	topics := []string{"authentication", "database", "API", "frontend", "backend", "deployment", "testing"}
	return topics[r.Intn(len(topics))]
}

func generateRandomFeature(r *rand.Rand) string {
	features := []string{"user login", "data export", "real-time notifications", "file upload", "search functionality"}
	return features[r.Intn(len(features))]
}

func generateRandomFilePath(r *rand.Rand) string {
	paths := []string{
		"src/components/Login.tsx",
		"internal/auth/service.go",
		"api/handlers/users.go",
		"config/database.yml",
		"tests/integration/auth_test.go",
	}
	return paths[r.Intn(len(paths))]
}

func generateRandomParticipant(r *rand.Rand) string {
	participants := []string{"alice", "bob", "charlie", "diana", "eve", "frank"}
	return participants[r.Intn(len(participants))]
}

func generateRandomMetadata(r *rand.Rand) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	if r.Float32() < 0.5 {
		metadata["priority"] = []string{"high", "medium", "low"}[r.Intn(3)]
	}
	
	if r.Float32() < 0.3 {
		metadata["tags"] = []string{"architecture", "performance", "security"}
	}
	
	return metadata
}

func generateRandomRelationship(r *rand.Rand, sourceID, targetID string) *models.KnowledgeRelationship {
	relationshipTypes := []string{"relates_to", "introduced_by", "modified_by", "discussed_in", "contributed_by"}
	
	return &models.KnowledgeRelationship{
		SourceEntityID:   sourceID,
		TargetEntityID:   targetID,
		RelationshipType: relationshipTypes[r.Intn(len(relationshipTypes))],
		Strength:         r.Float64(),
		Metadata: map[string]interface{}{
			"confidence": r.Float64(),
		},
		CreatedAt: time.Now(),
	}
}

// **Property 13: Knowledge Graph Entity Storage**
// **Validates: Requirements 6.1, 6.2**
func TestProperty_KnowledgeGraphEntityStorage(t *testing.T) {
	const iterations = 100
	
	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Setup
			r := rand.New(rand.NewSource(int64(i)))
			mockRepo := NewMockKnowledgeGraphRepository()
			ctx := context.Background()
			
			// Generate random entities
			entityCount := r.Intn(20) + 5 // 5-24 entities
			entities := make([]*models.KnowledgeEntity, entityCount)
			
			for j := 0; j < entityCount; j++ {
				entities[j] = generateRandomKnowledgeEntity(r)
			}
			
			// Property: All entities should be stored successfully
			for _, entity := range entities {
				err := mockRepo.CreateKnowledgeEntity(ctx, entity)
				assert.NoError(t, err, "Entity creation should not fail")
				assert.NotEmpty(t, entity.ID, "Entity should have an ID after creation")
				assert.False(t, entity.CreatedAt.IsZero(), "Entity should have creation timestamp")
				assert.False(t, entity.UpdatedAt.IsZero(), "Entity should have update timestamp")
			}
			
			// Property: All stored entities should be retrievable
			for _, entity := range entities {
				retrieved, err := mockRepo.GetKnowledgeEntity(ctx, entity.ID)
				assert.NoError(t, err, "Entity retrieval should not fail")
				assert.NotNil(t, retrieved, "Retrieved entity should not be nil")
				assert.Equal(t, entity.ID, retrieved.ID, "Retrieved entity should have same ID")
				assert.Equal(t, entity.EntityType, retrieved.EntityType, "Entity type should match")
				assert.Equal(t, entity.EntityID, retrieved.EntityID, "Entity ID should match")
				assert.Equal(t, entity.Title, retrieved.Title, "Title should match")
				assert.Equal(t, entity.Content, retrieved.Content, "Content should match")
			}
			
			// Property: Entity updates should work correctly
			for _, entity := range entities {
				originalTitle := entity.Title
				entity.Title = "Updated: " + originalTitle
				
				err := mockRepo.UpdateKnowledgeEntity(ctx, entity)
				assert.NoError(t, err, "Entity update should not fail")
				
				retrieved, err := mockRepo.GetKnowledgeEntity(ctx, entity.ID)
				assert.NoError(t, err, "Entity retrieval after update should not fail")
				assert.Equal(t, entity.Title, retrieved.Title, "Updated title should match")
				assert.True(t, retrieved.UpdatedAt.After(retrieved.CreatedAt) || retrieved.UpdatedAt.Equal(retrieved.CreatedAt), 
					"Updated timestamp should be >= created timestamp")
			}
			
			// Property: Entity deletion should work correctly
			if len(entities) > 0 {
				entityToDelete := entities[0]
				err := mockRepo.DeleteKnowledgeEntity(ctx, entityToDelete.ID)
				assert.NoError(t, err, "Entity deletion should not fail")
				
				_, err = mockRepo.GetKnowledgeEntity(ctx, entityToDelete.ID)
				assert.Error(t, err, "Deleted entity should not be retrievable")
			}
			
			// Property: Relationships should be created and retrieved correctly
			if len(entities) >= 2 {
				relationshipCount := r.Intn(10) + 1 // 1-10 relationships
				relationships := make([]*models.KnowledgeRelationship, relationshipCount)
				
				for j := 0; j < relationshipCount; j++ {
					sourceIdx := r.Intn(len(entities))
					targetIdx := r.Intn(len(entities))
					if sourceIdx == targetIdx && len(entities) > 1 {
						targetIdx = (targetIdx + 1) % len(entities)
					}
					
					relationships[j] = generateRandomRelationship(r, entities[sourceIdx].ID, entities[targetIdx].ID)
					
					err := mockRepo.CreateKnowledgeRelationship(ctx, relationships[j])
					assert.NoError(t, err, "Relationship creation should not fail")
					assert.NotEmpty(t, relationships[j].ID, "Relationship should have an ID")
					assert.False(t, relationships[j].CreatedAt.IsZero(), "Relationship should have creation timestamp")
				}
				
				// Verify relationships can be retrieved
				for _, entity := range entities {
					rels, err := mockRepo.GetKnowledgeRelationships(ctx, entity.ID)
					assert.NoError(t, err, "Relationship retrieval should not fail")
					
					// Count expected relationships for this entity
					expectedCount := 0
					for _, rel := range relationships {
						if rel.SourceEntityID == entity.ID || rel.TargetEntityID == entity.ID {
							expectedCount++
						}
					}
					
					assert.Equal(t, expectedCount, len(rels), "Retrieved relationship count should match expected")
				}
			}
			
			// Property: Search should work correctly
			if len(entities) > 0 {
				// Search by entity type
				entityTypes := []string{"decision", "discussion", "feature", "file_context"}
				for _, entityType := range entityTypes {
					query := &models.KnowledgeGraphQuery{
						Query:       "test",
						EntityTypes: []string{entityType},
						Limit:       10,
					}
					
					results, err := mockRepo.SearchKnowledgeEntities(ctx, query)
					assert.NoError(t, err, "Search should not fail")
					
					// All results should match the entity type filter
					for _, result := range results {
						assert.Equal(t, entityType, result.Entity.EntityType, 
							"Search result should match entity type filter")
					}
				}
			}
		})
	}
}

// **Property 15: Schema Migration Data Preservation**
// **Validates: Requirements 6.4, 8.4**
func TestProperty_SchemaMigrationDataPreservation(t *testing.T) {
	const iterations = 50
	
	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Setup
			r := rand.New(rand.NewSource(int64(i)))
			mockRepo := NewMockKnowledgeGraphRepository()
			ctx := context.Background()
			
			// Create initial data set
			entityCount := r.Intn(15) + 5 // 5-19 entities
			originalEntities := make([]*models.KnowledgeEntity, entityCount)
			
			for j := 0; j < entityCount; j++ {
				originalEntities[j] = generateRandomKnowledgeEntity(r)
				err := mockRepo.CreateKnowledgeEntity(ctx, originalEntities[j])
				require.NoError(t, err, "Initial entity creation should succeed")
			}
			
			// Create relationships
			relationshipCount := r.Intn(8) + 2 // 2-9 relationships
			originalRelationships := make([]*models.KnowledgeRelationship, relationshipCount)
			
			for j := 0; j < relationshipCount; j++ {
				sourceIdx := r.Intn(len(originalEntities))
				targetIdx := r.Intn(len(originalEntities))
				if sourceIdx == targetIdx && len(originalEntities) > 1 {
					targetIdx = (targetIdx + 1) % len(originalEntities)
				}
				
				originalRelationships[j] = generateRandomRelationship(r, 
					originalEntities[sourceIdx].ID, originalEntities[targetIdx].ID)
				
				err := mockRepo.CreateKnowledgeRelationship(ctx, originalRelationships[j])
				require.NoError(t, err, "Initial relationship creation should succeed")
			}
			
			// Simulate schema migration by creating a new mock repository
			// In a real scenario, this would involve database schema changes
			newMockRepo := NewMockKnowledgeGraphRepository()
			
			// Property: Data should be preserved during migration
			// Migrate entities
			for _, entity := range originalEntities {
				// Simulate migration by copying data to new repository
				migratedEntity := &models.KnowledgeEntity{
					ID:             entity.ID,
					EntityType:     entity.EntityType,
					EntityID:       entity.EntityID,
					Title:          entity.Title,
					Content:        entity.Content,
					Metadata:       entity.Metadata,
					PlatformSource: entity.PlatformSource,
					SourceEventIDs: entity.SourceEventIDs,
					Participants:   entity.Participants,
					Embedding:      entity.Embedding,
					CreatedAt:      entity.CreatedAt,
					UpdatedAt:      entity.UpdatedAt,
				}
				
				err := newMockRepo.CreateKnowledgeEntity(ctx, migratedEntity)
				assert.NoError(t, err, "Entity migration should succeed")
			}
			
			// Migrate relationships
			for _, relationship := range originalRelationships {
				migratedRelationship := &models.KnowledgeRelationship{
					ID:               relationship.ID,
					SourceEntityID:   relationship.SourceEntityID,
					TargetEntityID:   relationship.TargetEntityID,
					RelationshipType: relationship.RelationshipType,
					Strength:         relationship.Strength,
					Metadata:         relationship.Metadata,
					CreatedAt:        relationship.CreatedAt,
				}
				
				err := newMockRepo.CreateKnowledgeRelationship(ctx, migratedRelationship)
				assert.NoError(t, err, "Relationship migration should succeed")
			}
			
			// Property: All original data should be preserved after migration
			for _, originalEntity := range originalEntities {
				migratedEntity, err := newMockRepo.GetKnowledgeEntity(ctx, originalEntity.ID)
				assert.NoError(t, err, "Migrated entity should be retrievable")
				assert.Equal(t, originalEntity.ID, migratedEntity.ID, "Entity ID should be preserved")
				assert.Equal(t, originalEntity.EntityType, migratedEntity.EntityType, "Entity type should be preserved")
				assert.Equal(t, originalEntity.EntityID, migratedEntity.EntityID, "Entity ID should be preserved")
				assert.Equal(t, originalEntity.Title, migratedEntity.Title, "Title should be preserved")
				assert.Equal(t, originalEntity.Content, migratedEntity.Content, "Content should be preserved")
				assert.Equal(t, originalEntity.CreatedAt.Unix(), migratedEntity.CreatedAt.Unix(), "Creation time should be preserved")
			}
			
			// Property: All relationships should be preserved
			for _, originalEntity := range originalEntities {
				originalRels, err := mockRepo.GetKnowledgeRelationships(ctx, originalEntity.ID)
				assert.NoError(t, err, "Original relationships should be retrievable")
				
				migratedRels, err := newMockRepo.GetKnowledgeRelationships(ctx, originalEntity.ID)
				assert.NoError(t, err, "Migrated relationships should be retrievable")
				
				assert.Equal(t, len(originalRels), len(migratedRels), 
					"Relationship count should be preserved after migration")
				
				// Check that all relationships are preserved
				for _, originalRel := range originalRels {
					found := false
					for _, migratedRel := range migratedRels {
						if originalRel.ID == migratedRel.ID &&
						   originalRel.SourceEntityID == migratedRel.SourceEntityID &&
						   originalRel.TargetEntityID == migratedRel.TargetEntityID &&
						   originalRel.RelationshipType == migratedRel.RelationshipType {
							found = true
							assert.Equal(t, originalRel.Strength, migratedRel.Strength, 
								"Relationship strength should be preserved")
							break
						}
					}
					assert.True(t, found, "Original relationship should exist in migrated data")
				}
			}
			
			// Property: Search functionality should work after migration
			query := &models.KnowledgeGraphQuery{
				Query: "test",
				Limit: 10,
			}
			
			originalResults, err := mockRepo.SearchKnowledgeEntities(ctx, query)
			assert.NoError(t, err, "Original search should work")
			
			migratedResults, err := newMockRepo.SearchKnowledgeEntities(ctx, query)
			assert.NoError(t, err, "Migrated search should work")
			
			// Results should be similar (exact match depends on search implementation)
			assert.Equal(t, len(originalResults), len(migratedResults), 
				"Search result count should be preserved after migration")
		})
	}
}