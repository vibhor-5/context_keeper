package services

import (
	"context"
)

// ProductionMockAIService implements the AIService interface for production use
// This is a temporary implementation until a real AI service is integrated
type ProductionMockAIService struct{}

func (m *ProductionMockAIService) ExtractDecisions(ctx context.Context, events []NormalizedEvent) ([]DecisionRecord, error) {
	// Mock implementation - return empty decisions for now
	return []DecisionRecord{}, nil
}

func (m *ProductionMockAIService) SummarizeDiscussion(ctx context.Context, events []NormalizedEvent) (*DiscussionSummary, error) {
	// Mock implementation - return empty summary for now
	return &DiscussionSummary{}, nil
}

func (m *ProductionMockAIService) IdentifyFeatures(ctx context.Context, events []NormalizedEvent) ([]FeatureContext, error) {
	// Mock implementation - return empty features for now
	return []FeatureContext{}, nil
}

func (m *ProductionMockAIService) AnalyzeFileContext(ctx context.Context, events []NormalizedEvent) ([]FileContextHistory, error) {
	// Mock implementation - return empty file contexts for now
	return []FileContextHistory{}, nil
}