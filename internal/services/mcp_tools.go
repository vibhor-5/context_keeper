package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// searchProjectKnowledge implements the search_project_knowledge MCP tool
func (m *MCPServer) searchProjectKnowledge(ctx context.Context, arguments map[string]interface{}) (*MCPToolResult, error) {
	query := getStringParam(arguments, "query", "")
	if query == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "Error: query parameter is required",
			}},
			IsError: true,
		}, nil
	}

	entityTypes := getStringArrayParam(arguments, "entity_types")
	platforms := getStringArrayParam(arguments, "platforms")
	limit := getIntParam(arguments, "limit", 10)

	// Build knowledge graph query
	kgQuery := &models.KnowledgeGraphQuery{
		Query:          query,
		EntityTypes:    entityTypes,
		Platforms:      platforms,
		Limit:          limit,
		IncludeContent: true,
	}

	// Execute search
	results, err := m.knowledgeGraph.SearchKnowledge(ctx, kgQuery)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("Error searching knowledge: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format results
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Search Results for: %s\n\n", query))
	content.WriteString(fmt.Sprintf("Found %d results:\n\n", len(results)))

	for i, result := range results {
		content.WriteString(fmt.Sprintf("## %d. %s\n", i+1, result.Entity.Title))
		content.WriteString(fmt.Sprintf("**Type:** %s\n", result.Entity.EntityType))
		if result.Entity.PlatformSource != nil {
			content.WriteString(fmt.Sprintf("**Platform:** %s\n", *result.Entity.PlatformSource))
		}
		content.WriteString(fmt.Sprintf("**Similarity:** %.2f\n", result.Similarity))
		content.WriteString(fmt.Sprintf("**Created:** %s\n\n", result.Entity.CreatedAt.Format("2006-01-02 15:04")))
		
		if result.Entity.Content != "" {
			content.WriteString("**Content:**\n")
			content.WriteString(result.Entity.Content)
			content.WriteString("\n\n")
		}
		
		if len(result.Entity.Participants) > 0 {
			content.WriteString(fmt.Sprintf("**Participants:** %s\n\n", strings.Join(result.Entity.Participants, ", ")))
		}
		
		content.WriteString("---\n\n")
	}

	// Optimize response for token limits
	optimizedContent := m.responseOptimizer.OptimizeResponse(content.String(), "search_results")

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: optimizedContent,
		}},
	}, nil
}

// getContextForFile implements the get_context_for_file MCP tool
func (m *MCPServer) getContextForFile(ctx context.Context, arguments map[string]interface{}) (*MCPToolResult, error) {
	filePath := getStringParam(arguments, "file_path", "")
	if filePath == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "Error: file_path parameter is required",
			}},
			IsError: true,
		}, nil
	}

	includeHistory := getBoolParam(arguments, "include_history", true)

	// Get file context from knowledge graph
	fileContext, err := m.knowledgeGraph.GetContextForFile(ctx, filePath)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("Error getting file context: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format response
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Context for File: %s\n\n", filePath))

	// File context history
	if len(fileContext.FileContexts) > 0 {
		content.WriteString("## File Context History\n\n")
		for i, fc := range fileContext.FileContexts {
			content.WriteString(fmt.Sprintf("### %d. %s\n", i+1, fc.CreatedAt.Format("2006-01-02 15:04")))
			if fc.ChangeReason != nil {
				content.WriteString(fmt.Sprintf("**Change Reason:** %s\n", *fc.ChangeReason))
			}
			content.WriteString(fmt.Sprintf("**Discussion Context:** %s\n", fc.DiscussionContext))
			if len(fc.Contributors) > 0 {
				content.WriteString(fmt.Sprintf("**Contributors:** %s\n", strings.Join(fc.Contributors, ", ")))
			}
			if len(fc.RelatedDecisions) > 0 {
				content.WriteString(fmt.Sprintf("**Related Decisions:** %s\n", strings.Join(fc.RelatedDecisions, ", ")))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Related entities
	if len(fileContext.RelatedEntities) > 0 {
		content.WriteString("## Related Entities\n\n")
		for i, entity := range fileContext.RelatedEntities {
			content.WriteString(fmt.Sprintf("### %d. %s (%s)\n", i+1, entity.Entity.Title, entity.Entity.EntityType))
			content.WriteString(fmt.Sprintf("**Similarity:** %.2f\n", entity.Similarity))
			if entity.Entity.Content != "" {
				// Truncate content for readability
				entityContent := entity.Entity.Content
				if len(entityContent) > 200 {
					entityContent = entityContent[:200] + "..."
				}
				content.WriteString(fmt.Sprintf("**Content:** %s\n", entityContent))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Related decisions
	if len(fileContext.RelatedDecisions) > 0 {
		content.WriteString("## Related Decisions\n\n")
		for i, decision := range fileContext.RelatedDecisions {
			content.WriteString(fmt.Sprintf("### %d. %s\n", i+1, decision.Entity.Title))
			content.WriteString(fmt.Sprintf("**Created:** %s\n", decision.Entity.CreatedAt.Format("2006-01-02 15:04")))
			if decision.Entity.Content != "" {
				content.WriteString(fmt.Sprintf("**Decision:** %s\n", decision.Entity.Content))
			}
			content.WriteString("\n")
		}
	}

	if includeHistory {
		// Add note about history inclusion
		content.WriteString("\n---\n*Note: Historical context and related PRs are included in the above information.*\n")
	}

	// Optimize response for token limits
	optimizedContent := m.responseOptimizer.OptimizeResponse(content.String(), "file_context")

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: optimizedContent,
		}},
	}, nil
}

// getDecisionHistory implements the get_decision_history MCP tool
func (m *MCPServer) getDecisionHistory(ctx context.Context, arguments map[string]interface{}) (*MCPToolResult, error) {
	target := getStringParam(arguments, "target", "")
	if target == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "Error: target parameter is required",
			}},
			IsError: true,
		}, nil
	}

	limit := getIntParam(arguments, "limit", 20)

	// Get decision history from knowledge graph
	decisionHistory, err := m.knowledgeGraph.GetDecisionHistory(ctx, target)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("Error getting decision history: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format response
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Decision History for: %s\n\n", target))

	if len(decisionHistory.Decisions) == 0 {
		content.WriteString("No decisions found for this target.\n")
	} else {
		content.WriteString(fmt.Sprintf("Found %d decisions:\n\n", len(decisionHistory.Decisions)))

		// Limit results
		decisions := decisionHistory.Decisions
		if len(decisions) > limit {
			decisions = decisions[:limit]
		}

		for i, decision := range decisions {
			content.WriteString(fmt.Sprintf("## %d. %s\n", i+1, decision.Title))
			content.WriteString(fmt.Sprintf("**Created:** %s\n", decision.CreatedAt.Format("2006-01-02 15:04")))
			content.WriteString(fmt.Sprintf("**Status:** %s\n", decision.Status))
			content.WriteString(fmt.Sprintf("**Platform:** %s\n\n", decision.PlatformSource))
			
			content.WriteString("**Decision:**\n")
			content.WriteString(decision.Decision)
			content.WriteString("\n\n")
			
			if decision.Rationale != nil && *decision.Rationale != "" {
				content.WriteString("**Rationale:**\n")
				content.WriteString(*decision.Rationale)
				content.WriteString("\n\n")
			}
			
			if len(decision.Alternatives) > 0 {
				content.WriteString("**Alternatives Considered:**\n")
				for _, alt := range decision.Alternatives {
					content.WriteString(fmt.Sprintf("- %s\n", alt))
				}
				content.WriteString("\n")
			}
			
			if len(decision.Consequences) > 0 {
				content.WriteString("**Consequences:**\n")
				for _, cons := range decision.Consequences {
					content.WriteString(fmt.Sprintf("- %s\n", cons))
				}
				content.WriteString("\n")
			}
			
			if len(decision.Participants) > 0 {
				content.WriteString(fmt.Sprintf("**Participants:** %s\n", strings.Join(decision.Participants, ", ")))
			}
			
			content.WriteString("\n---\n\n")
		}

		if len(decisionHistory.Decisions) > limit {
			content.WriteString(fmt.Sprintf("*Showing %d of %d decisions. Use limit parameter to see more.*\n", limit, len(decisionHistory.Decisions)))
		}
	}

	// Optimize response for token limits
	optimizedContent := m.responseOptimizer.OptimizeResponse(content.String(), "decision_history")

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: optimizedContent,
		}},
	}, nil
}

// listRecentArchitectureDiscussions implements the list_recent_architecture_discussions MCP tool
func (m *MCPServer) listRecentArchitectureDiscussions(ctx context.Context, arguments map[string]interface{}) (*MCPToolResult, error) {
	limit := getIntParam(arguments, "limit", 10)
	daysBack := getIntParam(arguments, "days_back", 30)

	// Get recent architecture discussions
	discussions, err := m.knowledgeGraph.GetRecentArchitectureDiscussions(ctx, limit)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("Error getting architecture discussions: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Filter by date range
	cutoffDate := time.Now().AddDate(0, 0, -daysBack)
	var filteredDiscussions []models.DiscussionSummary
	for _, discussion := range discussions {
		if discussion.CreatedAt.After(cutoffDate) {
			filteredDiscussions = append(filteredDiscussions, discussion)
		}
	}

	// Format response
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Recent Architecture Discussions (Last %d days)\n\n", daysBack))

	if len(filteredDiscussions) == 0 {
		content.WriteString("No recent architecture discussions found.\n")
	} else {
		content.WriteString(fmt.Sprintf("Found %d discussions:\n\n", len(filteredDiscussions)))

		for i, discussion := range filteredDiscussions {
			content.WriteString(fmt.Sprintf("## %d. Discussion from %s\n", i+1, discussion.CreatedAt.Format("2006-01-02 15:04")))
			content.WriteString(fmt.Sprintf("**Platform:** %s\n", discussion.Platform))
			if discussion.ThreadID != nil {
				content.WriteString(fmt.Sprintf("**Thread ID:** %s\n", *discussion.ThreadID))
			}
			content.WriteString("\n")
			
			content.WriteString("**Summary:**\n")
			content.WriteString(discussion.Summary)
			content.WriteString("\n\n")
			
			if len(discussion.KeyPoints) > 0 {
				content.WriteString("**Key Points:**\n")
				for _, point := range discussion.KeyPoints {
					content.WriteString(fmt.Sprintf("- %s\n", point))
				}
				content.WriteString("\n")
			}
			
			if len(discussion.ActionItems) > 0 {
				content.WriteString("**Action Items:**\n")
				for _, item := range discussion.ActionItems {
					content.WriteString(fmt.Sprintf("- %s\n", item))
				}
				content.WriteString("\n")
			}
			
			if len(discussion.Participants) > 0 {
				content.WriteString(fmt.Sprintf("**Participants:** %s\n", strings.Join(discussion.Participants, ", ")))
			}
			
			if len(discussion.FileReferences) > 0 {
				content.WriteString(fmt.Sprintf("**File References:** %s\n", strings.Join(discussion.FileReferences, ", ")))
			}
			
			content.WriteString("\n---\n\n")
		}
	}

	// Optimize response for token limits
	optimizedContent := m.responseOptimizer.OptimizeResponse(content.String(), "architecture_discussions")

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: optimizedContent,
		}},
	}, nil
}

// explainWhyCodeExists implements the explain_why_code_exists MCP tool
func (m *MCPServer) explainWhyCodeExists(ctx context.Context, arguments map[string]interface{}) (*MCPToolResult, error) {
	filePath := getStringParam(arguments, "file_path", "")
	if filePath == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "Error: file_path parameter is required",
			}},
			IsError: true,
		}, nil
	}

	// Extract line range if provided
	var lineRange *struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	if lr, ok := arguments["line_range"].(map[string]interface{}); ok {
		lineRange = &struct {
			Start int `json:"start"`
			End   int `json:"end"`
		}{
			Start: getIntParam(lr, "start", 0),
			End:   getIntParam(lr, "end", 0),
		}
	}

	// Get comprehensive file context
	fileContext, err := m.knowledgeGraph.GetContextForFile(ctx, filePath)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("Error getting file context: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Search for decisions related to this file
	decisionHistory, err := m.knowledgeGraph.GetDecisionHistory(ctx, filePath)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("Error getting decision history: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format comprehensive explanation
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Why %s Exists\n\n", filePath))

	if lineRange != nil {
		content.WriteString(fmt.Sprintf("*Focusing on lines %d-%d*\n\n", lineRange.Start, lineRange.End))
	}

	// Historical context
	if len(fileContext.FileContexts) > 0 {
		content.WriteString("## Historical Context\n\n")
		
		// Sort by creation date (most recent first)
		contexts := fileContext.FileContexts
		
		for i, fc := range contexts {
			content.WriteString(fmt.Sprintf("### %s\n", fc.CreatedAt.Format("2006-01-02 15:04")))
			if fc.ChangeReason != nil && *fc.ChangeReason != "" {
				content.WriteString(fmt.Sprintf("**Reason for Change:** %s\n\n", *fc.ChangeReason))
			}
			content.WriteString(fmt.Sprintf("**Context:** %s\n", fc.DiscussionContext))
			if len(fc.Contributors) > 0 {
				content.WriteString(fmt.Sprintf("**Contributors:** %s\n", strings.Join(fc.Contributors, ", ")))
			}
			content.WriteString("\n")
			
			if i < len(contexts)-1 {
				content.WriteString("---\n\n")
			}
		}
		content.WriteString("\n")
	}

	// Related decisions
	if len(decisionHistory.Decisions) > 0 {
		content.WriteString("## Related Decisions\n\n")
		content.WriteString("The following architectural decisions influenced this code:\n\n")
		
		for i, decision := range decisionHistory.Decisions {
			content.WriteString(fmt.Sprintf("### %d. %s\n", i+1, decision.Title))
			content.WriteString(fmt.Sprintf("**Date:** %s\n", decision.CreatedAt.Format("2006-01-02")))
			content.WriteString(fmt.Sprintf("**Decision:** %s\n", decision.Decision))
			
			if decision.Rationale != nil && *decision.Rationale != "" {
				content.WriteString(fmt.Sprintf("**Rationale:** %s\n", *decision.Rationale))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Related discussions and features
	if len(fileContext.RelatedEntities) > 0 {
		content.WriteString("## Related Context\n\n")
		
		for _, entity := range fileContext.RelatedEntities {
			if entity.Entity.EntityType == "discussion" || entity.Entity.EntityType == "feature" {
				content.WriteString(fmt.Sprintf("### %s (%s)\n", entity.Entity.Title, entity.Entity.EntityType))
				if entity.Entity.Content != "" {
					// Truncate for readability
					entityContent := entity.Entity.Content
					if len(entityContent) > 300 {
						entityContent = entityContent[:300] + "..."
					}
					content.WriteString(fmt.Sprintf("%s\n\n", entityContent))
				}
			}
		}
	}

	// Summary
	content.WriteString("## Summary\n\n")
	if len(fileContext.FileContexts) > 0 || len(decisionHistory.Decisions) > 0 {
		content.WriteString("This code exists as a result of the historical context, decisions, and discussions outlined above. ")
		content.WriteString("The evolution of this file reflects the team's architectural choices and problem-solving approach over time.")
	} else {
		content.WriteString("Limited historical context is available for this file. ")
		content.WriteString("This could indicate it's a newer file or that relevant discussions happened outside the tracked platforms.")
	}

	// Optimize response for token limits
	optimizedContent := m.responseOptimizer.OptimizeResponse(content.String(), "code_explanation")

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: optimizedContent,
		}},
	}, nil
}