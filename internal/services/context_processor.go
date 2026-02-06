package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ContextProcessor transforms raw platform events into structured knowledge
type ContextProcessor struct {
	aiService    AIService
	logger       Logger
	batchSize    int
	maxRetries   int
	retryDelay   time.Duration
}

// NewContextProcessor creates a new context processor
func NewContextProcessor(aiService AIService, logger Logger) *ContextProcessor {
	return &ContextProcessor{
		aiService:  aiService,
		logger:     logger,
		batchSize:  100,  // Default batch size
		maxRetries: 3,    // Default max retries
		retryDelay: time.Second * 2, // Default retry delay
	}
}

// NewContextProcessorWithConfig creates a new context processor with custom configuration
func NewContextProcessorWithConfig(aiService AIService, logger Logger, batchSize, maxRetries int, retryDelay time.Duration) *ContextProcessor {
	return &ContextProcessor{
		aiService:  aiService,
		logger:     logger,
		batchSize:  batchSize,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// ProcessingResult contains the structured knowledge extracted from events
type ProcessingResult struct {
	DecisionRecords     []DecisionRecord     `json:"decision_records"`
	DiscussionSummaries []DiscussionSummary  `json:"discussion_summaries"`
	FeatureContexts     []FeatureContext     `json:"feature_contexts"`
	FileContexts        []FileContextHistory `json:"file_contexts"`
	Relationships       []Relationship       `json:"relationships"`
	ProcessedEvents     int                  `json:"processed_events"`
	Errors              []ProcessingError    `json:"errors"`
}

// DecisionRecord represents an engineering decision extracted from discussions
type DecisionRecord struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Decision       string    `json:"decision"`
	Rationale      string    `json:"rationale"`
	Alternatives   []string  `json:"alternatives"`
	Consequences   []string  `json:"consequences"`
	Status         string    `json:"status"` // active, superseded, deprecated
	PlatformSource string    `json:"platform_source"`
	SourceEventIDs []string  `json:"source_event_ids"`
	Participants   []string  `json:"participants"`
	ProjectID      string    `json:"project_id"` // Project scoping
	CreatedAt      time.Time `json:"created_at"`
}

// DiscussionSummary represents a summarized conversation thread
type DiscussionSummary struct {
	ID                string    `json:"id"`
	ThreadID          string    `json:"thread_id"`
	Platform          string    `json:"platform"`
	Participants      []string  `json:"participants"`
	Summary           string    `json:"summary"`
	KeyPoints         []string  `json:"key_points"`
	ActionItems       []string  `json:"action_items"`
	FileReferences    []string  `json:"file_references"`
	FeatureReferences []string  `json:"feature_references"`
	ProjectID         string    `json:"project_id"` // Project scoping
	CreatedAt         time.Time `json:"created_at"`
}

// FeatureContext represents the development history of a feature
type FeatureContext struct {
	ID            string    `json:"id"`
	FeatureName   string    `json:"feature_name"`
	Description   string    `json:"description"`
	Status        string    `json:"status"` // planned, in_progress, completed, deprecated
	Contributors  []string  `json:"contributors"`
	RelatedFiles  []string  `json:"related_files"`
	Discussions   []string  `json:"discussions"`
	Decisions     []string  `json:"decisions"`
	ProjectID     string    `json:"project_id"` // Project scoping
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// FileContextHistory represents the change and discussion history of a file
type FileContextHistory struct {
	ID                string                 `json:"id"`
	FilePath          string                 `json:"file_path"`
	ChangeReason      string                 `json:"change_reason"`
	DiscussionContext string                 `json:"discussion_context"`
	RelatedDecisions  []string               `json:"related_decisions"`
	Contributors      []string               `json:"contributors"`
	PlatformSources   map[string]interface{} `json:"platform_sources"`
	ProjectID         string                 `json:"project_id"` // Project scoping
	CreatedAt         time.Time              `json:"created_at"`
}

// Relationship represents a connection between entities
type Relationship struct {
	ID           string                 `json:"id"`
	SourceType   string                 `json:"source_type"`
	SourceID     string                 `json:"source_id"`
	TargetType   string                 `json:"target_type"`
	TargetID     string                 `json:"target_id"`
	Type         string                 `json:"type"` // relates_to, introduced_by, modified_by, discussed_in
	Strength     float64                `json:"strength"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ProcessingError represents an error that occurred during processing
type ProcessingError struct {
	EventID   string    `json:"event_id"`
	Platform  string    `json:"platform"`
	Error     string    `json:"error"`
	Retryable bool      `json:"retryable"`
	Timestamp time.Time `json:"timestamp"`
}

// AIService interface for AI-powered context extraction
type AIService interface {
	ExtractDecisions(ctx context.Context, events []NormalizedEvent) ([]DecisionRecord, error)
	SummarizeDiscussion(ctx context.Context, events []NormalizedEvent) (*DiscussionSummary, error)
	IdentifyFeatures(ctx context.Context, events []NormalizedEvent) ([]FeatureContext, error)
	AnalyzeFileContext(ctx context.Context, events []NormalizedEvent) ([]FileContextHistory, error)
}

// Logger interface for structured logging
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
}

// ProcessEvents transforms raw platform events into structured knowledge
func (cp *ContextProcessor) ProcessEvents(ctx context.Context, events []NormalizedEvent) (*ProcessingResult, error) {
	if len(events) == 0 {
		return &ProcessingResult{
			DecisionRecords:     []DecisionRecord{},
			DiscussionSummaries: []DiscussionSummary{},
			FeatureContexts:     []FeatureContext{},
			FileContexts:        []FileContextHistory{},
			Relationships:       []Relationship{},
			ProcessedEvents:     0,
			Errors:              []ProcessingError{},
		}, nil
	}

	cp.logger.Info("Starting context processing", map[string]interface{}{
		"event_count": len(events),
		"batch_size":  cp.batchSize,
	})

	// Process events in batches for better resource management
	return cp.processBatches(ctx, events)
}

// processBatches processes events in configurable batch sizes
func (cp *ContextProcessor) processBatches(ctx context.Context, events []NormalizedEvent) (*ProcessingResult, error) {
	totalResult := &ProcessingResult{
		DecisionRecords:     []DecisionRecord{},
		DiscussionSummaries: []DiscussionSummary{},
		FeatureContexts:     []FeatureContext{},
		FileContexts:        []FileContextHistory{},
		Relationships:       []Relationship{},
		ProcessedEvents:     0,
		Errors:              []ProcessingError{},
	}

	// Process events in batches
	for i := 0; i < len(events); i += cp.batchSize {
		end := i + cp.batchSize
		if end > len(events) {
			end = len(events)
		}

		batch := events[i:end]
		cp.logger.Debug("Processing batch", map[string]interface{}{
			"batch_start": i,
			"batch_end":   end,
			"batch_size":  len(batch),
		})

		// Process batch with retry logic
		batchResult, err := cp.processBatchWithRetry(ctx, batch)
		if err != nil {
			cp.logger.Error("Failed to process batch after retries", err, map[string]interface{}{
				"batch_start": i,
				"batch_end":   end,
			})
			
			// Add batch-level error but continue processing other batches
			cp.addBatchProcessingError(totalResult, batch, "batch_processing_failed", err)
			continue
		}

		// Merge batch results
		cp.mergeBatchResults(totalResult, batchResult)
	}

	// Extract relationships from all processed entities
	relationships := cp.extractRelationships(totalResult)
	totalResult.Relationships = relationships

	cp.logger.Info("Context processing completed", map[string]interface{}{
		"decisions":     len(totalResult.DecisionRecords),
		"summaries":     len(totalResult.DiscussionSummaries),
		"features":      len(totalResult.FeatureContexts),
		"file_contexts": len(totalResult.FileContexts),
		"relationships": len(totalResult.Relationships),
		"errors":        len(totalResult.Errors),
		"total_events":  totalResult.ProcessedEvents,
	})

	return totalResult, nil
}

// processBatchWithRetry processes a batch with retry logic
func (cp *ContextProcessor) processBatchWithRetry(ctx context.Context, events []NormalizedEvent) (*ProcessingResult, error) {
	var lastErr error
	
	for attempt := 0; attempt <= cp.maxRetries; attempt++ {
		if attempt > 0 {
			cp.logger.Debug("Retrying batch processing", map[string]interface{}{
				"attempt":    attempt,
				"batch_size": len(events),
			})
			
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(cp.retryDelay * time.Duration(attempt)):
				// Continue with retry
			}
		}

		result, err := cp.processSingleBatch(ctx, events)
		if err == nil {
			return result, nil
		}

		lastErr = err
		
		// Check if error is retryable
		if !cp.isRetryableError(err) {
			cp.logger.Debug("Non-retryable error encountered", map[string]interface{}{
				"error": err.Error(),
			})
			break
		}
	}

	return nil, fmt.Errorf("batch processing failed after %d attempts: %w", cp.maxRetries, lastErr)
}

// processSingleBatch processes a single batch of events
func (cp *ContextProcessor) processSingleBatch(ctx context.Context, events []NormalizedEvent) (*ProcessingResult, error) {
	result := &ProcessingResult{
		DecisionRecords:     []DecisionRecord{},
		DiscussionSummaries: []DiscussionSummary{},
		FeatureContexts:     []FeatureContext{},
		FileContexts:        []FileContextHistory{},
		Relationships:       []Relationship{},
		ProcessedEvents:     len(events),
		Errors:              []ProcessingError{},
	}

	// Group related events (threads, file references, etc.)
	eventGroups := cp.groupRelatedEvents(events)

	// Process each group with individual error handling
	for groupIndex, group := range eventGroups {
		if err := cp.processEventGroupWithErrorHandling(ctx, group, result, groupIndex); err != nil {
			cp.logger.Error("Failed to process event group", err, map[string]interface{}{
				"group_index": groupIndex,
				"group_size":  len(group),
			})
			// Continue processing other groups - errors are already added to result
		}
	}

	return result, nil
}

// processEventGroupWithErrorHandling processes a group with comprehensive error handling
func (cp *ContextProcessor) processEventGroupWithErrorHandling(ctx context.Context, events []NormalizedEvent, result *ProcessingResult, groupIndex int) error {
	var processingErrors []error

	// Extract decisions with error handling
	if decisions, err := cp.extractDecisionsWithErrorHandling(ctx, events); err != nil {
		processingErrors = append(processingErrors, fmt.Errorf("decision extraction failed: %w", err))
		cp.addProcessingError(result, events, "decision_extraction", err)
	} else {
		result.DecisionRecords = append(result.DecisionRecords, decisions...)
	}

	// Generate discussion summaries with error handling
	if len(events) > 1 {
		if summary, err := cp.generateSummaryWithErrorHandling(ctx, events); err != nil {
			processingErrors = append(processingErrors, fmt.Errorf("summary generation failed: %w", err))
			cp.addProcessingError(result, events, "summary_generation", err)
		} else if summary != nil {
			result.DiscussionSummaries = append(result.DiscussionSummaries, *summary)
		}
	}

	// Build feature contexts with error handling
	if features, err := cp.buildFeatureContextsWithErrorHandling(ctx, events); err != nil {
		processingErrors = append(processingErrors, fmt.Errorf("feature extraction failed: %w", err))
		cp.addProcessingError(result, events, "feature_extraction", err)
	} else {
		result.FeatureContexts = append(result.FeatureContexts, features...)
	}

	// Track file contexts with error handling
	if fileContexts, err := cp.buildFileContextsWithErrorHandling(ctx, events); err != nil {
		processingErrors = append(processingErrors, fmt.Errorf("file context extraction failed: %w", err))
		cp.addProcessingError(result, events, "file_context_extraction", err)
	} else {
		result.FileContexts = append(result.FileContexts, fileContexts...)
	}

	// Return error only if all processing steps failed
	if len(processingErrors) == 4 {
		return fmt.Errorf("all processing steps failed for group %d", groupIndex)
	}

	return nil
}

// Error handling wrapper methods
func (cp *ContextProcessor) extractDecisionsWithErrorHandling(ctx context.Context, events []NormalizedEvent) ([]DecisionRecord, error) {
	defer func() {
		if r := recover(); r != nil {
			cp.logger.Error("Panic in decision extraction", fmt.Errorf("panic: %v", r), map[string]interface{}{
				"event_count": len(events),
			})
		}
	}()

	return cp.extractDecisions(ctx, events)
}

func (cp *ContextProcessor) generateSummaryWithErrorHandling(ctx context.Context, events []NormalizedEvent) (*DiscussionSummary, error) {
	defer func() {
		if r := recover(); r != nil {
			cp.logger.Error("Panic in summary generation", fmt.Errorf("panic: %v", r), map[string]interface{}{
				"event_count": len(events),
			})
		}
	}()

	return cp.generateSummary(ctx, events)
}

func (cp *ContextProcessor) buildFeatureContextsWithErrorHandling(ctx context.Context, events []NormalizedEvent) ([]FeatureContext, error) {
	defer func() {
		if r := recover(); r != nil {
			cp.logger.Error("Panic in feature context building", fmt.Errorf("panic: %v", r), map[string]interface{}{
				"event_count": len(events),
			})
		}
	}()

	return cp.buildFeatureContexts(ctx, events)
}

func (cp *ContextProcessor) buildFileContextsWithErrorHandling(ctx context.Context, events []NormalizedEvent) ([]FileContextHistory, error) {
	defer func() {
		if r := recover(); r != nil {
			cp.logger.Error("Panic in file context building", fmt.Errorf("panic: %v", r), map[string]interface{}{
				"event_count": len(events),
			})
		}
	}()

	return cp.buildFileContexts(ctx, events)
}

// mergeBatchResults merges results from a batch into the total result
func (cp *ContextProcessor) mergeBatchResults(total, batch *ProcessingResult) {
	total.DecisionRecords = append(total.DecisionRecords, batch.DecisionRecords...)
	total.DiscussionSummaries = append(total.DiscussionSummaries, batch.DiscussionSummaries...)
	total.FeatureContexts = append(total.FeatureContexts, batch.FeatureContexts...)
	total.FileContexts = append(total.FileContexts, batch.FileContexts...)
	total.Relationships = append(total.Relationships, batch.Relationships...)
	total.ProcessedEvents += batch.ProcessedEvents
	total.Errors = append(total.Errors, batch.Errors...)
}

// isRetryableError determines if an error is retryable
func (cp *ContextProcessor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())
	
	// Network-related errors are typically retryable
	retryablePatterns := []string{
		"timeout",
		"connection",
		"network",
		"temporary",
		"rate limit",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	// Non-retryable errors
	nonRetryablePatterns := []string{
		"invalid",
		"unauthorized",
		"forbidden",
		"not found",
		"bad request",
		"parse error",
		"syntax error",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errorStr, pattern) {
			return false
		}
	}

	// Default to retryable for unknown errors
	return true
}

// addBatchProcessingError adds a batch-level processing error
func (cp *ContextProcessor) addBatchProcessingError(result *ProcessingResult, events []NormalizedEvent, errorType string, err error) {
	for _, event := range events {
		processingError := ProcessingError{
			EventID:   event.PlatformID,
			Platform:  event.Platform,
			Error:     fmt.Sprintf("batch_%s: %v", errorType, err),
			Retryable: cp.isRetryableError(err),
			Timestamp: time.Now(),
		}
		result.Errors = append(result.Errors, processingError)
	}
}

// groupRelatedEvents groups events by thread, file references, or temporal proximity
func (cp *ContextProcessor) groupRelatedEvents(events []NormalizedEvent) [][]NormalizedEvent {
	var groups [][]NormalizedEvent
	threadGroups := make(map[string][]NormalizedEvent)
	ungroupedEvents := []NormalizedEvent{}

	// Group by thread ID first
	for _, event := range events {
		if event.ThreadID != nil && *event.ThreadID != "" {
			threadID := *event.ThreadID
			threadGroups[threadID] = append(threadGroups[threadID], event)
		} else {
			ungroupedEvents = append(ungroupedEvents, event)
		}
	}

	// Add thread groups
	for _, group := range threadGroups {
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}

	// Group ungrouped events by file references and temporal proximity
	fileGroups := cp.groupByFileReferences(ungroupedEvents)
	groups = append(groups, fileGroups...)

	return groups
}

// groupByFileReferences groups events that reference the same files
func (cp *ContextProcessor) groupByFileReferences(events []NormalizedEvent) [][]NormalizedEvent {
	var groups [][]NormalizedEvent
	fileGroups := make(map[string][]NormalizedEvent)

	for _, event := range events {
		if len(event.FileRefs) > 0 {
			// Use the first file reference as the group key
			fileKey := event.FileRefs[0]
			fileGroups[fileKey] = append(fileGroups[fileKey], event)
		} else {
			// Single event group for events without file references
			groups = append(groups, []NormalizedEvent{event})
		}
	}

	// Add file groups
	for _, group := range fileGroups {
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}

	return groups
}


// extractDecisions extracts decision records from event groups
func (cp *ContextProcessor) extractDecisions(ctx context.Context, events []NormalizedEvent) ([]DecisionRecord, error) {
	// Look for decision-like patterns in the content
	var decisions []DecisionRecord

	for _, event := range events {
		if cp.containsDecisionKeywords(event.Content) {
			decision := DecisionRecord{
				ID:             fmt.Sprintf("decision-%s-%d", event.Platform, event.Timestamp.Unix()),
				Title:          cp.extractDecisionTitle(event.Content),
				Decision:       cp.extractDecisionText(event.Content),
				Rationale:      cp.extractRationale(event.Content),
				Status:         "active",
				PlatformSource: event.Platform,
				SourceEventIDs: []string{event.PlatformID},
				Participants:   []string{event.Author},
				CreatedAt:      event.Timestamp,
			}

			// Extract alternatives and consequences if present
			decision.Alternatives = cp.extractAlternatives(event.Content)
			decision.Consequences = cp.extractConsequences(event.Content)

			decisions = append(decisions, decision)
		}
	}

	return decisions, nil
}

// generateSummary creates a discussion summary for a group of events
func (cp *ContextProcessor) generateSummary(ctx context.Context, events []NormalizedEvent) (*DiscussionSummary, error) {
	if len(events) == 0 {
		return nil, nil
	}

	// Get unique participants
	participantMap := make(map[string]bool)
	var fileRefs []string
	var featureRefs []string
	platform := events[0].Platform
	threadID := ""

	if events[0].ThreadID != nil {
		threadID = *events[0].ThreadID
	}

	for _, event := range events {
		participantMap[event.Author] = true
		fileRefs = append(fileRefs, event.FileRefs...)
		featureRefs = append(featureRefs, event.FeatureRefs...)
	}

	participants := make([]string, 0, len(participantMap))
	for participant := range participantMap {
		participants = append(participants, participant)
	}

	// Generate summary content
	summary := cp.generateSummaryText(events)
	keyPoints := cp.extractKeyPoints(events)
	actionItems := cp.extractActionItems(events)

	return &DiscussionSummary{
		ID:                fmt.Sprintf("summary-%s-%d", platform, events[0].Timestamp.Unix()),
		ThreadID:          threadID,
		Platform:          platform,
		Participants:      participants,
		Summary:           summary,
		KeyPoints:         keyPoints,
		ActionItems:       actionItems,
		FileReferences:    cp.deduplicateStrings(fileRefs),
		FeatureReferences: cp.deduplicateStrings(featureRefs),
		CreatedAt:         time.Now(),
	}, nil
}

// buildFeatureContexts identifies and builds feature contexts from events
func (cp *ContextProcessor) buildFeatureContexts(ctx context.Context, events []NormalizedEvent) ([]FeatureContext, error) {
	var features []FeatureContext
	featureMap := make(map[string]*FeatureContext)

	for _, event := range events {
		// Extract feature names from content and references
		featureNames := cp.extractFeatureNames(event.Content)
		featureNames = append(featureNames, event.FeatureRefs...)

		for _, featureName := range featureNames {
			if featureName == "" {
				continue
			}

			if feature, exists := featureMap[featureName]; exists {
				// Update existing feature
				feature.Contributors = cp.addUniqueString(feature.Contributors, event.Author)
				feature.RelatedFiles = append(feature.RelatedFiles, event.FileRefs...)
				feature.UpdatedAt = event.Timestamp
			} else {
				// Create new feature
				feature := &FeatureContext{
					ID:           fmt.Sprintf("feature-%s-%d", strings.ReplaceAll(featureName, " ", "-"), event.Timestamp.Unix()),
					FeatureName:  featureName,
					Description:  cp.extractFeatureDescription(event.Content, featureName),
					Status:       cp.inferFeatureStatus(event.Content),
					Contributors: []string{event.Author},
					RelatedFiles: event.FileRefs,
					Discussions:  []string{event.PlatformID},
					CreatedAt:    event.Timestamp,
					UpdatedAt:    event.Timestamp,
				}
				featureMap[featureName] = feature
			}
		}
	}

	// Convert map to slice
	for _, feature := range featureMap {
		feature.RelatedFiles = cp.deduplicateStrings(feature.RelatedFiles)
		features = append(features, *feature)
	}

	return features, nil
}

// buildFileContexts creates file context history from events
func (cp *ContextProcessor) buildFileContexts(ctx context.Context, events []NormalizedEvent) ([]FileContextHistory, error) {
	var fileContexts []FileContextHistory
	fileMap := make(map[string]*FileContextHistory)

	for _, event := range events {
		for _, filePath := range event.FileRefs {
			if filePath == "" {
				continue
			}

			if fileContext, exists := fileMap[filePath]; exists {
				// Update existing file context
				fileContext.Contributors = cp.addUniqueString(fileContext.Contributors, event.Author)
				fileContext.DiscussionContext += "\n" + event.Content
			} else {
				// Create new file context
				platformSources := map[string]interface{}{
					event.Platform: []string{event.PlatformID},
				}

				fileContext := &FileContextHistory{
					ID:                fmt.Sprintf("file-%s-%d", strings.ReplaceAll(filePath, "/", "-"), event.Timestamp.Unix()),
					FilePath:          filePath,
					ChangeReason:      cp.extractChangeReason(event.Content),
					DiscussionContext: event.Content,
					Contributors:      []string{event.Author},
					PlatformSources:   platformSources,
					CreatedAt:         event.Timestamp,
				}
				fileMap[filePath] = fileContext
			}
		}
	}

	// Convert map to slice
	for _, fileContext := range fileMap {
		fileContexts = append(fileContexts, *fileContext)
	}

	return fileContexts, nil
}

// extractRelationships identifies relationships between entities
func (cp *ContextProcessor) extractRelationships(result *ProcessingResult) []Relationship {
	var relationships []Relationship

	// Create relationships between decisions and files
	for _, decision := range result.DecisionRecords {
		for _, fileContext := range result.FileContexts {
			if cp.areRelated(decision.Decision, fileContext.FilePath) {
				relationship := Relationship{
					ID:         fmt.Sprintf("rel-%s-%s", decision.ID, fileContext.ID),
					SourceType: "decision",
					SourceID:   decision.ID,
					TargetType: "file",
					TargetID:   fileContext.ID,
					Type:       "relates_to",
					Strength:   cp.calculateRelationshipStrength(decision.Decision, fileContext.FilePath, fileContext.DiscussionContext),
					Metadata: map[string]interface{}{
						"decision_title": decision.Title,
						"file_path":      fileContext.FilePath,
					},
					CreatedAt:  time.Now(),
				}
				relationships = append(relationships, relationship)
			}
		}
	}

	// Create relationships between features and files
	for _, feature := range result.FeatureContexts {
		for _, filePath := range feature.RelatedFiles {
			for _, fileContext := range result.FileContexts {
				if fileContext.FilePath == filePath {
					relationship := Relationship{
						ID:         fmt.Sprintf("rel-%s-%s", feature.ID, fileContext.ID),
						SourceType: "feature",
						SourceID:   feature.ID,
						TargetType: "file",
						TargetID:   fileContext.ID,
						Type:       "modified_by",
						Strength:   0.9,
						Metadata: map[string]interface{}{
							"feature_name": feature.FeatureName,
							"file_path":    fileContext.FilePath,
						},
						CreatedAt:  time.Now(),
					}
					relationships = append(relationships, relationship)
				}
			}
		}
	}

	// Create relationships between decisions and features
	for _, decision := range result.DecisionRecords {
		for _, feature := range result.FeatureContexts {
			if cp.areDecisionAndFeatureRelated(decision, feature) {
				relationship := Relationship{
					ID:         fmt.Sprintf("rel-%s-%s", decision.ID, feature.ID),
					SourceType: "decision",
					SourceID:   decision.ID,
					TargetType: "feature",
					TargetID:   feature.ID,
					Type:       "introduced_by",
					Strength:   cp.calculateDecisionFeatureStrength(decision, feature),
					Metadata: map[string]interface{}{
						"decision_title": decision.Title,
						"feature_name":   feature.FeatureName,
					},
					CreatedAt:  time.Now(),
				}
				relationships = append(relationships, relationship)
			}
		}
	}

	// Create relationships between discussions and decisions
	for _, summary := range result.DiscussionSummaries {
		for _, decision := range result.DecisionRecords {
			if cp.areDiscussionAndDecisionRelated(summary, decision) {
				relationship := Relationship{
					ID:         fmt.Sprintf("rel-%s-%s", summary.ID, decision.ID),
					SourceType: "discussion",
					SourceID:   summary.ID,
					TargetType: "decision",
					TargetID:   decision.ID,
					Type:       "discussed_in",
					Strength:   0.8,
					Metadata: map[string]interface{}{
						"thread_id":      summary.ThreadID,
						"decision_title": decision.Title,
						"platform":       summary.Platform,
					},
					CreatedAt:  time.Now(),
				}
				relationships = append(relationships, relationship)
			}
		}
	}

	// Create contributor relationships
	relationships = append(relationships, cp.extractContributorRelationships(result)...)

	// Create cross-platform relationships
	relationships = append(relationships, cp.extractCrossPlatformRelationships(result)...)

	return relationships
}

// calculateRelationshipStrength calculates the strength of relationship between decision and file
func (cp *ContextProcessor) calculateRelationshipStrength(decisionText, filePath, fileContext string) float64 {
	strength := 0.0
	
	// Direct file path mention in decision
	if strings.Contains(strings.ToLower(decisionText), strings.ToLower(filePath)) {
		strength += 0.5
	}
	
	// File extension relevance
	if strings.Contains(decisionText, cp.extractFileExtension(filePath)) {
		strength += 0.2
	}
	
	// Common keywords between decision and file context
	decisionWords := cp.extractKeywords(decisionText)
	contextWords := cp.extractKeywords(fileContext)
	commonWords := cp.findCommonWords(decisionWords, contextWords)
	
	if len(commonWords) > 0 {
		strength += float64(len(commonWords)) * 0.1
	}
	
	// Cap at 1.0
	if strength > 1.0 {
		strength = 1.0
	}
	
	return strength
}

// areDecisionAndFeatureRelated checks if a decision and feature are related
func (cp *ContextProcessor) areDecisionAndFeatureRelated(decision DecisionRecord, feature FeatureContext) bool {
	// Check if feature name is mentioned in decision
	if strings.Contains(strings.ToLower(decision.Decision), strings.ToLower(feature.FeatureName)) {
		return true
	}
	
	// Check if decision title contains feature keywords
	if strings.Contains(strings.ToLower(decision.Title), strings.ToLower(feature.FeatureName)) {
		return true
	}
	
	// Check for common participants
	for _, participant := range decision.Participants {
		for _, contributor := range feature.Contributors {
			if participant == contributor {
				return true
			}
		}
	}
	
	return false
}

// calculateDecisionFeatureStrength calculates relationship strength between decision and feature
func (cp *ContextProcessor) calculateDecisionFeatureStrength(decision DecisionRecord, feature FeatureContext) float64 {
	strength := 0.0
	
	// Direct feature name mention
	if strings.Contains(strings.ToLower(decision.Decision), strings.ToLower(feature.FeatureName)) {
		strength += 0.6
	}
	
	// Common participants
	commonParticipants := 0
	for _, participant := range decision.Participants {
		for _, contributor := range feature.Contributors {
			if participant == contributor {
				commonParticipants++
			}
		}
	}
	
	if commonParticipants > 0 {
		strength += float64(commonParticipants) * 0.2
	}
	
	// Temporal proximity (decisions made around feature development time)
	timeDiff := decision.CreatedAt.Sub(feature.CreatedAt).Hours()
	if timeDiff < 24*7 { // Within a week
		strength += 0.2
	}
	
	// Cap at 1.0
	if strength > 1.0 {
		strength = 1.0
	}
	
	return strength
}

// areDiscussionAndDecisionRelated checks if a discussion and decision are related
func (cp *ContextProcessor) areDiscussionAndDecisionRelated(discussion DiscussionSummary, decision DecisionRecord) bool {
	// Check if decision keywords appear in discussion
	decisionKeywords := []string{"decision", "decided", "agreed", "consensus", "resolved"}
	
	for _, keyword := range decisionKeywords {
		if strings.Contains(strings.ToLower(discussion.Summary), keyword) {
			return true
		}
	}
	
	// Check for common participants
	for _, participant := range discussion.Participants {
		for _, decisionParticipant := range decision.Participants {
			if participant == decisionParticipant {
				return true
			}
		}
	}
	
	// Check if discussion mentions decision topic
	if strings.Contains(strings.ToLower(discussion.Summary), strings.ToLower(decision.Title)) {
		return true
	}
	
	return false
}

// extractContributorRelationships creates relationships between contributors and entities
func (cp *ContextProcessor) extractContributorRelationships(result *ProcessingResult) []Relationship {
	var relationships []Relationship
	
	// Track contributor activity across entities
	contributorActivity := make(map[string][]string)
	
	// Collect contributor activities
	for _, decision := range result.DecisionRecords {
		for _, participant := range decision.Participants {
			contributorActivity[participant] = append(contributorActivity[participant], "decision:"+decision.ID)
		}
	}
	
	for _, feature := range result.FeatureContexts {
		for _, contributor := range feature.Contributors {
			contributorActivity[contributor] = append(contributorActivity[contributor], "feature:"+feature.ID)
		}
	}
	
	for _, fileContext := range result.FileContexts {
		for _, contributor := range fileContext.Contributors {
			contributorActivity[contributor] = append(contributorActivity[contributor], "file:"+fileContext.ID)
		}
	}
	
	// Create relationships between entities that share contributors
	for contributor, activities := range contributorActivity {
		if len(activities) > 1 {
			// Create relationships between entities this contributor worked on
			for i, activity1 := range activities {
				for j, activity2 := range activities {
					if i < j { // Avoid duplicates
						parts1 := strings.Split(activity1, ":")
						parts2 := strings.Split(activity2, ":")
						
						if len(parts1) == 2 && len(parts2) == 2 {
							relationship := Relationship{
								ID:         fmt.Sprintf("contrib-%s-%s-%s", contributor, parts1[1], parts2[1]),
								SourceType: parts1[0],
								SourceID:   parts1[1],
								TargetType: parts2[0],
								TargetID:   parts2[1],
								Type:       "contributed_by",
								Strength:   0.7,
								Metadata: map[string]interface{}{
									"contributor": contributor,
								},
								CreatedAt:  time.Now(),
							}
							relationships = append(relationships, relationship)
						}
					}
				}
			}
		}
	}
	
	return relationships
}

// extractCrossPlatformRelationships identifies relationships across different platforms
func (cp *ContextProcessor) extractCrossPlatformRelationships(result *ProcessingResult) []Relationship {
	var relationships []Relationship
	
	// Group entities by platform
	platformEntities := make(map[string][]string)
	
	for _, decision := range result.DecisionRecords {
		platformEntities[decision.PlatformSource] = append(platformEntities[decision.PlatformSource], "decision:"+decision.ID)
	}
	
	for _, summary := range result.DiscussionSummaries {
		platformEntities[summary.Platform] = append(platformEntities[summary.Platform], "discussion:"+summary.ID)
	}
	
	// Find cross-platform relationships based on file references
	for _, fileContext := range result.FileContexts {
		platforms := make([]string, 0)
		for platform := range fileContext.PlatformSources {
			platforms = append(platforms, platform)
		}
		
		// If file is discussed on multiple platforms, create cross-platform relationships
		if len(platforms) > 1 {
			for i, platform1 := range platforms {
				for j, platform2 := range platforms {
					if i < j {
						relationship := Relationship{
							ID:         fmt.Sprintf("cross-%s-%s-%s", fileContext.ID, platform1, platform2),
							SourceType: "file",
							SourceID:   fileContext.ID,
							TargetType: "cross_platform",
							TargetID:   fmt.Sprintf("%s-%s", platform1, platform2),
							Type:       "discussed_across",
							Strength:   0.8,
							Metadata: map[string]interface{}{
								"file_path":  fileContext.FilePath,
								"platforms":  []string{platform1, platform2},
							},
							CreatedAt:  time.Now(),
						}
						relationships = append(relationships, relationship)
					}
				}
			}
		}
	}
	
	return relationships
}

// Helper methods for relationship identification

func (cp *ContextProcessor) extractFileExtension(filePath string) string {
	parts := strings.Split(filePath, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func (cp *ContextProcessor) extractKeywords(text string) []string {
	// Simple keyword extraction - split by spaces and filter
	words := strings.Fields(strings.ToLower(text))
	var keywords []string
	
	// Filter out common words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
	}
	
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}
	
	return keywords
}

func (cp *ContextProcessor) findCommonWords(words1, words2 []string) []string {
	wordSet := make(map[string]bool)
	var common []string
	
	// Create set from first slice
	for _, word := range words1 {
		wordSet[word] = true
	}
	
	// Find common words
	for _, word := range words2 {
		if wordSet[word] {
			common = append(common, word)
			delete(wordSet, word) // Avoid duplicates
		}
	}
	
	return common
}

// Helper methods for text analysis and extraction

func (cp *ContextProcessor) containsDecisionKeywords(content string) bool {
	decisionKeywords := []string{
		"decided", "decision", "we should", "let's use", "going with",
		"agreed", "consensus", "resolved", "conclusion", "final decision",
	}
	
	lowerContent := strings.ToLower(content)
	for _, keyword := range decisionKeywords {
		if strings.Contains(lowerContent, keyword) {
			return true
		}
	}
	return false
}

func (cp *ContextProcessor) extractDecisionTitle(content string) string {
	// Extract first sentence or first 100 characters as title
	sentences := strings.Split(content, ".")
	if len(sentences) > 0 && len(sentences[0]) > 0 {
		title := strings.TrimSpace(sentences[0])
		if len(title) > 100 {
			title = title[:97] + "..."
		}
		return title
	}
	return "Decision"
}

func (cp *ContextProcessor) extractDecisionText(content string) string {
	// For now, return the full content. Could be enhanced with NLP
	return content
}

func (cp *ContextProcessor) extractRationale(content string) string {
	// Look for rationale keywords
	rationaleKeywords := []string{"because", "since", "due to", "reason", "rationale"}
	
	lowerContent := strings.ToLower(content)
	for _, keyword := range rationaleKeywords {
		if idx := strings.Index(lowerContent, keyword); idx != -1 {
			// Extract text after the keyword
			remaining := content[idx:]
			sentences := strings.Split(remaining, ".")
			if len(sentences) > 0 {
				return strings.TrimSpace(sentences[0])
			}
		}
	}
	return ""
}

func (cp *ContextProcessor) extractAlternatives(content string) []string {
	// Look for alternative patterns
	alternativePatterns := []regexp.Regexp{
		*regexp.MustCompile(`(?i)alternative[s]?:?\s*(.+)`),
		*regexp.MustCompile(`(?i)other option[s]?:?\s*(.+)`),
		*regexp.MustCompile(`(?i)could also:?\s*(.+)`),
	}
	
	var alternatives []string
	for _, pattern := range alternativePatterns {
		matches := pattern.FindStringSubmatch(content)
		if len(matches) > 1 {
			alternatives = append(alternatives, strings.TrimSpace(matches[1]))
		}
	}
	return alternatives
}

func (cp *ContextProcessor) extractConsequences(content string) []string {
	// Look for consequence patterns
	consequencePatterns := []regexp.Regexp{
		*regexp.MustCompile(`(?i)consequence[s]?:?\s*(.+)`),
		*regexp.MustCompile(`(?i)impact:?\s*(.+)`),
		*regexp.MustCompile(`(?i)this means:?\s*(.+)`),
	}
	
	var consequences []string
	for _, pattern := range consequencePatterns {
		matches := pattern.FindStringSubmatch(content)
		if len(matches) > 1 {
			consequences = append(consequences, strings.TrimSpace(matches[1]))
		}
	}
	return consequences
}

func (cp *ContextProcessor) generateSummaryText(events []NormalizedEvent) string {
	if len(events) == 0 {
		return ""
	}
	
	// Simple summary: combine first sentences of each event
	var summaryParts []string
	for i, event := range events {
		if i >= 3 { // Limit to first 3 events for summary
			break
		}
		sentences := strings.Split(event.Content, ".")
		if len(sentences) > 0 && len(sentences[0]) > 0 {
			summaryParts = append(summaryParts, strings.TrimSpace(sentences[0]))
		}
	}
	
	summary := strings.Join(summaryParts, ". ")
	if len(summary) > 500 {
		summary = summary[:497] + "..."
	}
	return summary
}

func (cp *ContextProcessor) extractKeyPoints(events []NormalizedEvent) []string {
	var keyPoints []string
	
	for _, event := range events {
		// Look for bullet points or numbered lists
		lines := strings.Split(event.Content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || 
			   regexp.MustCompile(`^\d+\.`).MatchString(line) {
				keyPoints = append(keyPoints, line)
			}
		}
	}
	
	return keyPoints
}

func (cp *ContextProcessor) extractActionItems(events []NormalizedEvent) []string {
	var actionItems []string
	
	actionKeywords := []string{"todo", "action", "need to", "should", "must", "will"}
	
	for _, event := range events {
		lowerContent := strings.ToLower(event.Content)
		for _, keyword := range actionKeywords {
			if strings.Contains(lowerContent, keyword) {
				// Extract the sentence containing the action keyword
				sentences := strings.Split(event.Content, ".")
				for _, sentence := range sentences {
					if strings.Contains(strings.ToLower(sentence), keyword) {
						actionItems = append(actionItems, strings.TrimSpace(sentence))
						break
					}
				}
			}
		}
	}
	
	return cp.deduplicateStrings(actionItems)
}

func (cp *ContextProcessor) extractFeatureNames(content string) []string {
	// Look for feature patterns
	featurePatterns := []regexp.Regexp{
		*regexp.MustCompile(`(?i)feature[:\s]+([a-zA-Z0-9\s\-_]+)`),
		*regexp.MustCompile(`(?i)implement[:\s]+([a-zA-Z0-9\s\-_]+)`),
		*regexp.MustCompile(`(?i)add[:\s]+([a-zA-Z0-9\s\-_]+)`),
	}
	
	var features []string
	for _, pattern := range featurePatterns {
		matches := pattern.FindStringSubmatch(content)
		if len(matches) > 1 {
			featureName := strings.TrimSpace(matches[1])
			if len(featureName) > 0 && len(featureName) < 100 {
				features = append(features, featureName)
			}
		}
	}
	return features
}

func (cp *ContextProcessor) extractFeatureDescription(content, featureName string) string {
	// Extract context around the feature name
	idx := strings.Index(strings.ToLower(content), strings.ToLower(featureName))
	if idx == -1 {
		return ""
	}
	
	// Get surrounding context
	start := max(0, idx-50)
	end := minInt(len(content), idx+len(featureName)+100)
	
	return strings.TrimSpace(content[start:end])
}

func (cp *ContextProcessor) inferFeatureStatus(content string) string {
	lowerContent := strings.ToLower(content)
	
	if strings.Contains(lowerContent, "completed") || strings.Contains(lowerContent, "done") {
		return "completed"
	}
	if strings.Contains(lowerContent, "working on") || strings.Contains(lowerContent, "in progress") {
		return "in_progress"
	}
	if strings.Contains(lowerContent, "planning") || strings.Contains(lowerContent, "will") {
		return "planned"
	}
	
	return "in_progress" // Default status
}

func (cp *ContextProcessor) extractChangeReason(content string) string {
	// Look for change reason patterns
	reasonPatterns := []regexp.Regexp{
		*regexp.MustCompile(`(?i)changed because:?\s*(.+)`),
		*regexp.MustCompile(`(?i)updated to:?\s*(.+)`),
		*regexp.MustCompile(`(?i)modified for:?\s*(.+)`),
	}
	
	for _, pattern := range reasonPatterns {
		matches := pattern.FindStringSubmatch(content)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	// Default: use first sentence
	sentences := strings.Split(content, ".")
	if len(sentences) > 0 && len(sentences[0]) > 0 {
		return strings.TrimSpace(sentences[0])
	}
	
	return "File modification discussed"
}

func (cp *ContextProcessor) areRelated(text1, text2 string) bool {
	// Simple relatedness check - could be enhanced with semantic similarity
	return strings.Contains(strings.ToLower(text1), strings.ToLower(text2)) ||
		   strings.Contains(strings.ToLower(text2), strings.ToLower(text1))
}

func (cp *ContextProcessor) addProcessingError(result *ProcessingResult, events []NormalizedEvent, errorType string, err error) {
	for _, event := range events {
		processingError := ProcessingError{
			EventID:   event.PlatformID,
			Platform:  event.Platform,
			Error:     fmt.Sprintf("%s: %v", errorType, err),
			Retryable: true,
			Timestamp: time.Now(),
		}
		result.Errors = append(result.Errors, processingError)
	}
}

// Utility functions

func (cp *ContextProcessor) deduplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

func (cp *ContextProcessor) addUniqueString(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}