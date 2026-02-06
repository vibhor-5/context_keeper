package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// KnowledgeGraphServiceImpl handles knowledge graph operations and entity management
type KnowledgeGraphServiceImpl struct {
	repository    RepositoryStore
	permissionSvc PermissionService
	processor     *ContextProcessor
	logger        Logger
}

// NewKnowledgeGraphService creates a new knowledge graph service
func NewKnowledgeGraphService(repository RepositoryStore, permissionSvc PermissionService, processor *ContextProcessor, logger Logger) *KnowledgeGraphServiceImpl {
	return &KnowledgeGraphServiceImpl{
		repository:    repository,
		permissionSvc: permissionSvc,
		processor:     processor,
		logger:        logger,
	}
}

// ProcessAndStoreEvents processes events and stores the resulting knowledge entities
func (kg *KnowledgeGraphServiceImpl) ProcessAndStoreEvents(ctx context.Context, events []NormalizedEvent) (*ProcessingResult, error) {
	kg.logger.Info("Processing and storing events in knowledge graph", map[string]interface{}{
		"event_count": len(events),
	})

	// Process events using the context processor
	result, err := kg.processor.ProcessEvents(ctx, events)
	if err != nil {
		return nil, fmt.Errorf("failed to process events: %w", err)
	}

	// Store all processed entities in the knowledge graph
	if err := kg.storeProcessingResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to store processing result: %w", err)
	}

	kg.logger.Info("Successfully processed and stored events", map[string]interface{}{
		"decisions":     len(result.DecisionRecords),
		"summaries":     len(result.DiscussionSummaries),
		"features":      len(result.FeatureContexts),
		"file_contexts": len(result.FileContexts),
		"relationships": len(result.Relationships),
		"errors":        len(result.Errors),
	})

	return result, nil
}

// storeProcessingResult stores all entities and relationships from processing result
func (kg *KnowledgeGraphServiceImpl) storeProcessingResult(ctx context.Context, result *ProcessingResult) error {
	// Store decision records
	for _, decision := range result.DecisionRecords {
		if err := kg.storeDecisionRecord(ctx, &decision); err != nil {
			kg.logger.Error("Failed to store decision record", err, map[string]interface{}{
				"decision_id": decision.ID,
			})
			continue
		}
	}

	// Store discussion summaries
	for _, summary := range result.DiscussionSummaries {
		if err := kg.storeDiscussionSummary(ctx, &summary); err != nil {
			kg.logger.Error("Failed to store discussion summary", err, map[string]interface{}{
				"summary_id": summary.ID,
			})
			continue
		}
	}

	// Store feature contexts
	for _, feature := range result.FeatureContexts {
		if err := kg.storeFeatureContext(ctx, &feature); err != nil {
			kg.logger.Error("Failed to store feature context", err, map[string]interface{}{
				"feature_id": feature.ID,
			})
			continue
		}
	}

	// Store file contexts
	for _, fileContext := range result.FileContexts {
		if err := kg.storeFileContextHistory(ctx, &fileContext); err != nil {
			kg.logger.Error("Failed to store file context", err, map[string]interface{}{
				"context_id": fileContext.ID,
			})
			continue
		}
	}

	// Store relationships
	for _, relationship := range result.Relationships {
		if err := kg.storeRelationship(ctx, &relationship); err != nil {
			kg.logger.Error("Failed to store relationship", err, map[string]interface{}{
				"relationship_id": relationship.ID,
			})
			continue
		}
	}

	return nil
}

// storeDecisionRecord stores a decision record and its knowledge entity
func (kg *KnowledgeGraphServiceImpl) storeDecisionRecord(ctx context.Context, decision *DecisionRecord) error {
	// Create knowledge entity
	entity := &models.KnowledgeEntity{
		EntityType:     "decision",
		EntityID:       decision.ID,
		Title:          decision.Title,
		Content:        decision.Decision,
		PlatformSource: &decision.PlatformSource,
		SourceEventIDs: models.StringList(decision.SourceEventIDs),
		Participants:   models.StringList(decision.Participants),
		CreatedAt:      decision.CreatedAt,
		Metadata: map[string]interface{}{
			"rationale":     decision.Rationale,
			"alternatives":  decision.Alternatives,
			"consequences":  decision.Consequences,
			"status":        decision.Status,
			"project_id":    decision.ProjectID, // Add project scoping
		},
	}

	if err := kg.repository.CreateKnowledgeEntity(ctx, entity); err != nil {
		return fmt.Errorf("failed to create knowledge entity for decision: %w", err)
	}

	// Create decision record
	decisionModel := &models.DecisionRecord{
		EntityID:       entity.ID,
		DecisionID:     decision.ID,
		Title:          decision.Title,
		Decision:       decision.Decision,
		Rationale:      &decision.Rationale,
		Alternatives:   models.StringList(decision.Alternatives),
		Consequences:   models.StringList(decision.Consequences),
		Status:         decision.Status,
		PlatformSource: decision.PlatformSource,
		SourceEventIDs: models.StringList(decision.SourceEventIDs),
		Participants:   models.StringList(decision.Participants),
		CreatedAt:      decision.CreatedAt,
	}

	return kg.repository.CreateDecisionRecord(ctx, decisionModel)
}

// storeDiscussionSummary stores a discussion summary and its knowledge entity
func (kg *KnowledgeGraphServiceImpl) storeDiscussionSummary(ctx context.Context, summary *DiscussionSummary) error {
	// Create knowledge entity
	entity := &models.KnowledgeEntity{
		EntityType:     "discussion",
		EntityID:       summary.ID,
		Title:          fmt.Sprintf("Discussion: %s", summary.ThreadID),
		Content:        summary.Summary,
		PlatformSource: &summary.Platform,
		Participants:   models.StringList(summary.Participants),
		CreatedAt:      summary.CreatedAt,
		Metadata: map[string]interface{}{
			"thread_id":          summary.ThreadID,
			"key_points":         summary.KeyPoints,
			"action_items":       summary.ActionItems,
			"file_references":    summary.FileReferences,
			"feature_references": summary.FeatureReferences,
			"project_id":         summary.ProjectID, // Add project scoping
		},
	}

	if err := kg.repository.CreateKnowledgeEntity(ctx, entity); err != nil {
		return fmt.Errorf("failed to create knowledge entity for discussion: %w", err)
	}

	// Create discussion summary
	summaryModel := &models.DiscussionSummary{
		EntityID:          entity.ID,
		SummaryID:         summary.ID,
		ThreadID:          &summary.ThreadID,
		Platform:          summary.Platform,
		Participants:      models.StringList(summary.Participants),
		Summary:           summary.Summary,
		KeyPoints:         models.StringList(summary.KeyPoints),
		ActionItems:       models.StringList(summary.ActionItems),
		FileReferences:    models.StringList(summary.FileReferences),
		FeatureReferences: models.StringList(summary.FeatureReferences),
		CreatedAt:         summary.CreatedAt,
	}

	return kg.repository.CreateDiscussionSummary(ctx, summaryModel)
}

// storeFeatureContext stores a feature context and its knowledge entity
func (kg *KnowledgeGraphServiceImpl) storeFeatureContext(ctx context.Context, feature *FeatureContext) error {
	// Create knowledge entity
	entity := &models.KnowledgeEntity{
		EntityType:   "feature",
		EntityID:     feature.ID,
		Title:        fmt.Sprintf("Feature: %s", feature.FeatureName),
		Content:      feature.Description,
		Participants: models.StringList(feature.Contributors),
		CreatedAt:    feature.CreatedAt,
		Metadata: map[string]interface{}{
			"feature_name":  feature.FeatureName,
			"status":        feature.Status,
			"related_files": feature.RelatedFiles,
			"discussions":   feature.Discussions,
			"decisions":     feature.Decisions,
			"project_id":    feature.ProjectID, // Add project scoping
		},
	}

	if err := kg.repository.CreateKnowledgeEntity(ctx, entity); err != nil {
		return fmt.Errorf("failed to create knowledge entity for feature: %w", err)
	}

	// Create feature context
	featureModel := &models.FeatureContext{
		EntityID:     entity.ID,
		FeatureID:    feature.ID,
		FeatureName:  feature.FeatureName,
		Description:  &feature.Description,
		Status:       feature.Status,
		Contributors: models.StringList(feature.Contributors),
		RelatedFiles: models.StringList(feature.RelatedFiles),
		Discussions:  models.StringList(feature.Discussions),
		Decisions:    models.StringList(feature.Decisions),
		CreatedAt:    feature.CreatedAt,
		UpdatedAt:    feature.UpdatedAt,
	}

	return kg.repository.CreateFeatureContext(ctx, featureModel)
}

// storeFileContextHistory stores a file context and its knowledge entity
func (kg *KnowledgeGraphServiceImpl) storeFileContextHistory(ctx context.Context, fileContext *FileContextHistory) error {
	// Create knowledge entity
	entity := &models.KnowledgeEntity{
		EntityType:   "file_context",
		EntityID:     fileContext.ID,
		Title:        fmt.Sprintf("File: %s", fileContext.FilePath),
		Content:      fileContext.DiscussionContext,
		Participants: models.StringList(fileContext.Contributors),
		CreatedAt:    fileContext.CreatedAt,
		Metadata: map[string]interface{}{
			"file_path":          fileContext.FilePath,
			"change_reason":      fileContext.ChangeReason,
			"related_decisions":  fileContext.RelatedDecisions,
			"platform_sources":   fileContext.PlatformSources,
			"project_id":         fileContext.ProjectID, // Add project scoping
		},
	}

	if err := kg.repository.CreateKnowledgeEntity(ctx, entity); err != nil {
		return fmt.Errorf("failed to create knowledge entity for file context: %w", err)
	}

	// Create file context history
	fileContextModel := &models.FileContextHistory{
		EntityID:          entity.ID,
		ContextID:         fileContext.ID,
		FilePath:          fileContext.FilePath,
		ChangeReason:      &fileContext.ChangeReason,
		DiscussionContext: fileContext.DiscussionContext,
		RelatedDecisions:  models.StringList(fileContext.RelatedDecisions),
		Contributors:      models.StringList(fileContext.Contributors),
		PlatformSources:   fileContext.PlatformSources,
		CreatedAt:         fileContext.CreatedAt,
	}

	return kg.repository.CreateFileContextHistory(ctx, fileContextModel)
}

// storeRelationship stores a relationship between knowledge entities
func (kg *KnowledgeGraphServiceImpl) storeRelationship(ctx context.Context, relationship *Relationship) error {
	// Find source and target entities by their original IDs
	sourceEntity, err := kg.findEntityByOriginalID(ctx, relationship.SourceType, relationship.SourceID)
	if err != nil {
		kg.logger.Error("Failed to find source entity for relationship", err, map[string]interface{}{
			"source_type": relationship.SourceType,
			"source_id":   relationship.SourceID,
		})
		return nil // Skip this relationship rather than failing
	}

	targetEntity, err := kg.findEntityByOriginalID(ctx, relationship.TargetType, relationship.TargetID)
	if err != nil {
		kg.logger.Error("Failed to find target entity for relationship", err, map[string]interface{}{
			"target_type": relationship.TargetType,
			"target_id":   relationship.TargetID,
		})
		return nil // Skip this relationship rather than failing
	}

	// Create knowledge relationship
	relationshipModel := &models.KnowledgeRelationship{
		SourceEntityID:   sourceEntity.ID,
		TargetEntityID:   targetEntity.ID,
		RelationshipType: relationship.Type,
		Strength:         relationship.Strength,
		Metadata:         relationship.Metadata,
		CreatedAt:        relationship.CreatedAt,
	}

	return kg.repository.CreateKnowledgeRelationship(ctx, relationshipModel)
}

// findEntityByOriginalID finds a knowledge entity by its original processing ID
func (kg *KnowledgeGraphServiceImpl) findEntityByOriginalID(ctx context.Context, entityType, originalID string) (*models.KnowledgeEntity, error) {
	// This is a simplified lookup - in production you might want to cache this
	query := &models.KnowledgeGraphQuery{
		Query:       originalID,
		EntityTypes: []string{entityType},
		Limit:       1,
	}

	results, err := kg.repository.SearchKnowledgeEntities(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("entity not found: type=%s, id=%s", entityType, originalID)
	}

	return &results[0].Entity, nil
}

// SearchKnowledge performs semantic search across the knowledge graph
func (kg *KnowledgeGraphServiceImpl) SearchKnowledge(ctx context.Context, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error) {
	kg.logger.Info("Performing knowledge graph search", map[string]interface{}{
		"query":        query.Query,
		"entity_types": query.EntityTypes,
		"platforms":    query.Platforms,
		"limit":        query.Limit,
	})

	results, err := kg.repository.SearchKnowledgeEntities(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge entities: %w", err)
	}

	kg.logger.Info("Knowledge graph search completed", map[string]interface{}{
		"results_count": len(results),
	})

	return results, nil
}

// GetContextForFile retrieves all context information for a specific file
func (kg *KnowledgeGraphServiceImpl) GetContextForFile(ctx context.Context, filePath string) (*FileContextResponse, error) {
	kg.logger.Info("Getting context for file", map[string]interface{}{
		"file_path": filePath,
	})

	// Get file context history
	fileContexts, err := kg.repository.GetFileContextHistory(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file context history: %w", err)
	}

	// Search for related entities
	query := &models.KnowledgeGraphQuery{
		Query:          filePath,
		IncludeContent: true,
		Limit:          20,
	}

	relatedEntities, err := kg.repository.SearchKnowledgeEntities(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search related entities: %w", err)
	}

	// Get decisions that mention this file
	decisionQuery := &models.KnowledgeGraphQuery{
		Query:       filePath,
		EntityTypes: []string{"decision"},
		Limit:       10,
	}

	relatedDecisions, err := kg.repository.SearchKnowledgeEntities(ctx, decisionQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search related decisions: %w", err)
	}

	return &FileContextResponse{
		FilePath:         filePath,
		FileContexts:     fileContexts,
		RelatedEntities:  relatedEntities,
		RelatedDecisions: relatedDecisions,
	}, nil
}

// GetDecisionHistory retrieves decision history for a feature or file
func (kg *KnowledgeGraphServiceImpl) GetDecisionHistory(ctx context.Context, target string) (*DecisionHistoryResponse, error) {
	kg.logger.Info("Getting decision history", map[string]interface{}{
		"target": target,
	})

	// Search for decisions related to the target
	query := &models.KnowledgeGraphQuery{
		Query:          target,
		EntityTypes:    []string{"decision"},
		IncludeContent: true,
		Limit:          50,
	}

	decisionResults, err := kg.repository.SearchKnowledgeEntities(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search decisions: %w", err)
	}

	// Get detailed decision records
	var decisions []models.DecisionRecord
	for _, result := range decisionResults {
		decision, err := kg.repository.GetDecisionRecord(ctx, result.Entity.EntityID)
		if err != nil {
			kg.logger.Error("Failed to get decision record", err, map[string]interface{}{
				"decision_id": result.Entity.EntityID,
			})
			continue
		}
		decisions = append(decisions, *decision)
	}

	return &DecisionHistoryResponse{
		Target:    target,
		Decisions: decisions,
	}, nil
}

// TraverseRelationships performs graph traversal to find related entities
func (kg *KnowledgeGraphServiceImpl) TraverseRelationships(ctx context.Context, startEntityID string, maxDepth int, relationshipTypes []string) (*models.GraphTraversalResult, error) {
	kg.logger.Info("Traversing knowledge graph relationships", map[string]interface{}{
		"start_entity":       startEntityID,
		"max_depth":          maxDepth,
		"relationship_types": relationshipTypes,
	})

	result, err := kg.repository.TraverseKnowledgeGraph(ctx, startEntityID, maxDepth, relationshipTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse knowledge graph: %w", err)
	}

	kg.logger.Info("Knowledge graph traversal completed", map[string]interface{}{
		"entities_found":    len(result.Path),
		"relationships":     len(result.Relationships),
		"max_depth_reached": result.Depth,
		"total_strength":    result.TotalStrength,
	})

	return result, nil
}

// GetRecentArchitectureDiscussions retrieves recent architecture-related discussions
func (kg *KnowledgeGraphServiceImpl) GetRecentArchitectureDiscussions(ctx context.Context, limit int) ([]models.DiscussionSummary, error) {
	kg.logger.Info("Getting recent architecture discussions", map[string]interface{}{
		"limit": limit,
	})

	// Search for architecture-related discussions
	architectureTerms := []string{
		"architecture", "design", "pattern", "structure", "framework",
		"system", "component", "module", "interface", "api",
	}

	query := &models.KnowledgeGraphQuery{
		Query:       strings.Join(architectureTerms, " OR "),
		EntityTypes: []string{"discussion"},
		Limit:       limit,
		DateRange: &models.DateRange{
			Start: timePtr(time.Now().AddDate(0, -3, 0)), // Last 3 months
		},
	}

	results, err := kg.repository.SearchKnowledgeEntities(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search architecture discussions: %w", err)
	}

	// Get detailed discussion summaries
	var discussions []models.DiscussionSummary
	for _, result := range results {
		discussion, err := kg.repository.GetDiscussionSummary(ctx, result.Entity.EntityID)
		if err != nil {
			kg.logger.Error("Failed to get discussion summary", err, map[string]interface{}{
				"summary_id": result.Entity.EntityID,
			})
			continue
		}
		discussions = append(discussions, *discussion)
	}

	return discussions, nil
}

// SearchKnowledgeByProject performs semantic search within a specific project
func (kg *KnowledgeGraphServiceImpl) SearchKnowledgeByProject(ctx context.Context, projectID string, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error) {
	kg.logger.Info("Performing project-scoped knowledge graph search", map[string]interface{}{
		"project_id":   projectID,
		"query":        query.Query,
		"entity_types": query.EntityTypes,
		"platforms":    query.Platforms,
		"limit":        query.Limit,
	})

	results, err := kg.repository.SearchKnowledgeEntitiesByProject(ctx, projectID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge entities by project: %w", err)
	}

	kg.logger.Info("Project-scoped knowledge graph search completed", map[string]interface{}{
		"project_id":    projectID,
		"results_count": len(results),
	})

	return results, nil
}

// GetContextForFileByProject retrieves all context information for a specific file within a project
func (kg *KnowledgeGraphServiceImpl) GetContextForFileByProject(ctx context.Context, projectID, filePath string) (*FileContextResponse, error) {
	kg.logger.Info("Getting context for file in project", map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	})

	// Get file context history (this method needs to be project-scoped in the repository)
	fileContexts, err := kg.repository.GetFileContextHistory(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file context history: %w", err)
	}

	// Filter file contexts to only include those from the specified project
	var projectFileContexts []models.FileContextHistory
	for _, fc := range fileContexts {
		// Check if this file context belongs to the project
		// This is a simplified check - in production you might want to store project_id directly
		if projectSources, ok := fc.PlatformSources["project_id"]; ok {
			if projectID == projectSources {
				projectFileContexts = append(projectFileContexts, fc)
			}
		}
	}

	// Search for related entities within the project
	query := &models.KnowledgeGraphQuery{
		Query:          filePath,
		IncludeContent: true,
		Limit:          20,
	}

	relatedEntities, err := kg.repository.SearchKnowledgeEntitiesByProject(ctx, projectID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search related entities: %w", err)
	}

	// Get decisions that mention this file within the project
	decisionQuery := &models.KnowledgeGraphQuery{
		Query:       filePath,
		EntityTypes: []string{"decision"},
		Limit:       10,
	}

	relatedDecisions, err := kg.repository.SearchKnowledgeEntitiesByProject(ctx, projectID, decisionQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search related decisions: %w", err)
	}

	return &FileContextResponse{
		FilePath:         filePath,
		FileContexts:     projectFileContexts,
		RelatedEntities:  relatedEntities,
		RelatedDecisions: relatedDecisions,
	}, nil
}

// GetDecisionHistoryByProject retrieves decision history for a feature or file within a project
func (kg *KnowledgeGraphServiceImpl) GetDecisionHistoryByProject(ctx context.Context, projectID, target string) (*DecisionHistoryResponse, error) {
	kg.logger.Info("Getting decision history for project", map[string]interface{}{
		"project_id": projectID,
		"target":     target,
	})

	// Search for decisions related to the target within the project
	query := &models.KnowledgeGraphQuery{
		Query:          target,
		EntityTypes:    []string{"decision"},
		IncludeContent: true,
		Limit:          50,
	}

	decisionResults, err := kg.repository.SearchKnowledgeEntitiesByProject(ctx, projectID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search decisions: %w", err)
	}

	// Get detailed decision records
	var decisions []models.DecisionRecord
	for _, result := range decisionResults {
		decision, err := kg.repository.GetDecisionRecord(ctx, result.Entity.EntityID)
		if err != nil {
			kg.logger.Error("Failed to get decision record", err, map[string]interface{}{
				"decision_id": result.Entity.EntityID,
			})
			continue
		}
		decisions = append(decisions, *decision)
	}

	return &DecisionHistoryResponse{
		Target:    target,
		Decisions: decisions,
	}, nil
}

// GetRecentArchitectureDiscussionsByProject retrieves recent architecture-related discussions within a project
func (kg *KnowledgeGraphServiceImpl) GetRecentArchitectureDiscussionsByProject(ctx context.Context, projectID string, limit int) ([]models.DiscussionSummary, error) {
	kg.logger.Info("Getting recent architecture discussions for project", map[string]interface{}{
		"project_id": projectID,
		"limit":      limit,
	})

	// Search for architecture-related discussions within the project
	architectureTerms := []string{
		"architecture", "design", "pattern", "structure", "framework",
		"system", "component", "module", "interface", "api",
	}

	query := &models.KnowledgeGraphQuery{
		Query:       strings.Join(architectureTerms, " OR "),
		EntityTypes: []string{"discussion"},
		Limit:       limit,
		DateRange: &models.DateRange{
			Start: timePtr(time.Now().AddDate(0, -3, 0)), // Last 3 months
		},
	}

	results, err := kg.repository.SearchKnowledgeEntitiesByProject(ctx, projectID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search architecture discussions: %w", err)
	}

	// Get detailed discussion summaries
	var discussions []models.DiscussionSummary
	for _, result := range results {
		discussion, err := kg.repository.GetDiscussionSummary(ctx, result.Entity.EntityID)
		if err != nil {
			kg.logger.Error("Failed to get discussion summary", err, map[string]interface{}{
				"summary_id": result.Entity.EntityID,
			})
			continue
		}
		discussions = append(discussions, *discussion)
	}

	return discussions, nil
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}