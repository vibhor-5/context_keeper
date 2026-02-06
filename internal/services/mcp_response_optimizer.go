package services

import (
	"fmt"
	"strings"
	"time"
)

// ResponseOptimizer handles response size management and summarization
type ResponseOptimizer struct {
	maxTokens int
	logger    Logger
}

// NewResponseOptimizer creates a new response optimizer
func NewResponseOptimizer(maxTokens int, logger Logger) *ResponseOptimizer {
	return &ResponseOptimizer{
		maxTokens: maxTokens,
		logger:    logger,
	}
}

// OptimizeResponse optimizes response content for token limits
func (ro *ResponseOptimizer) OptimizeResponse(content string, contentType string) string {
	// Estimate token count (rough approximation: 1 token ≈ 4 characters)
	estimatedTokens := len(content) / 4
	
	if estimatedTokens <= ro.maxTokens {
		return content
	}

	ro.logger.Info("Response exceeds token limit, optimizing", map[string]interface{}{
		"estimated_tokens": estimatedTokens,
		"max_tokens":       ro.maxTokens,
		"content_type":     contentType,
	})

	// Apply optimization based on content type
	switch contentType {
	case "search_results":
		return ro.optimizeSearchResults(content)
	case "file_context":
		return ro.optimizeFileContext(content)
	case "decision_history":
		return ro.optimizeDecisionHistory(content)
	case "architecture_discussions":
		return ro.optimizeArchitectureDiscussions(content)
	case "code_explanation":
		return ro.optimizeCodeExplanation(content)
	default:
		return ro.genericOptimization(content)
	}
}

// optimizeSearchResults optimizes search result responses
func (ro *ResponseOptimizer) optimizeSearchResults(content string) string {
	lines := strings.Split(content, "\n")
	var optimized strings.Builder
	
	// Keep header and summary
	for idx, line := range lines {
		if idx < 5 || strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			optimized.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "**Content:**") {
			// Truncate content sections
			optimized.WriteString("**Content:** [Content truncated for brevity]\n")
			// Skip until next section
			for j := idx + 1; j < len(lines) && !strings.HasPrefix(lines[j], "**"); j++ {
				idx = j
			}
		} else {
			optimized.WriteString(line + "\n")
		}
	}
	
	optimized.WriteString("\n*Note: Some content has been truncated due to response size limits. Use more specific queries for detailed information.*\n")
	return optimized.String()
}

// optimizeFileContext optimizes file context responses
func (ro *ResponseOptimizer) optimizeFileContext(content string) string {
	lines := strings.Split(content, "\n")
	var optimized strings.Builder
	
	// Keep headers and first few context entries
	contextCount := 0
	maxContexts := 3
	
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			optimized.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "### ") && strings.Contains(content, "File Context History") {
			if contextCount < maxContexts {
				optimized.WriteString(line + "\n")
				contextCount++
			} else {
				optimized.WriteString("### [Additional contexts truncated]\n")
				break
			}
		} else if contextCount <= maxContexts {
			optimized.WriteString(line + "\n")
		}
	}
	
	optimized.WriteString("\n*Note: File context has been truncated. Use get_decision_history for more detailed information.*\n")
	return optimized.String()
}

// optimizeDecisionHistory optimizes decision history responses
func (ro *ResponseOptimizer) optimizeDecisionHistory(content string) string {
	lines := strings.Split(content, "\n")
	var optimized strings.Builder
	
	// Keep headers and first few decisions
	decisionCount := 0
	maxDecisions := 5
	
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "Found ") {
			optimized.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "## ") {
			if decisionCount < maxDecisions {
				optimized.WriteString(line + "\n")
				decisionCount++
			} else {
				optimized.WriteString("## [Additional decisions truncated]\n")
				break
			}
		} else if decisionCount <= maxDecisions {
			// Truncate long rationale and consequence sections
			if strings.HasPrefix(line, "**Rationale:**") || strings.HasPrefix(line, "**Consequences:**") {
				optimized.WriteString(line + "\n")
				// Add next line but truncate if too long
				if i+1 < len(lines) {
					nextLine := lines[i+1]
					if len(nextLine) > 200 {
						nextLine = nextLine[:200] + "..."
					}
					optimized.WriteString(nextLine + "\n")
					i++ // Skip the next line since we processed it
				}
			} else {
				optimized.WriteString(line + "\n")
			}
		}
	}
	
	optimized.WriteString("\n*Note: Decision history has been truncated. Query specific decisions for full details.*\n")
	return optimized.String()
}

// optimizeArchitectureDiscussions optimizes architecture discussion responses
func (ro *ResponseOptimizer) optimizeArchitectureDiscussions(content string) string {
	lines := strings.Split(content, "\n")
	var optimized strings.Builder
	
	// Keep headers and first few discussions
	discussionCount := 0
	maxDiscussions := 3
	
	for idx, line := range lines {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "Found ") {
			optimized.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "## ") {
			if discussionCount < maxDiscussions {
				optimized.WriteString(line + "\n")
				discussionCount++
			} else {
				optimized.WriteString("## [Additional discussions truncated]\n")
				break
			}
		} else if discussionCount <= maxDiscussions {
			// Truncate long summaries
			if strings.HasPrefix(line, "**Summary:**") {
				optimized.WriteString(line + "\n")
				// Add next few lines but truncate
				for j := idx + 1; j < len(lines) && j < idx+4 && !strings.HasPrefix(lines[j], "**"); j++ {
					optimized.WriteString(lines[j] + "\n")
					idx = j
				}
				if idx+1 < len(lines) && !strings.HasPrefix(lines[idx+1], "**") {
					optimized.WriteString("[Summary truncated...]\n")
				}
			} else {
				optimized.WriteString(line + "\n")
			}
		}
	}
	
	optimized.WriteString("\n*Note: Architecture discussions have been truncated. Use search_project_knowledge for specific topics.*\n")
	return optimized.String()
}

// optimizeCodeExplanation optimizes code explanation responses
func (ro *ResponseOptimizer) optimizeCodeExplanation(content string) string {
	lines := strings.Split(content, "\n")
	var optimized strings.Builder
	
	// Keep headers and most important sections
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			optimized.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "### ") {
			// Keep first few subsections
			optimized.WriteString(line + "\n")
		} else {
			// Truncate long paragraphs
			if len(line) > 300 {
				line = line[:300] + "..."
			}
			optimized.WriteString(line + "\n")
		}
	}
	
	optimized.WriteString("\n*Note: Code explanation has been summarized. Use get_context_for_file for detailed context.*\n")
	return optimized.String()
}

// genericOptimization applies generic content truncation
func (ro *ResponseOptimizer) genericOptimization(content string) string {
	// Simple truncation with ellipsis
	targetLength := ro.maxTokens * 4 // Rough character estimate
	
	if len(content) <= targetLength {
		return content
	}
	
	// Find a good break point (end of sentence or paragraph)
	truncated := content[:targetLength]
	
	// Look for last sentence ending
	lastPeriod := strings.LastIndex(truncated, ".")
	lastNewline := strings.LastIndex(truncated, "\n\n")
	
	breakPoint := targetLength
	if lastNewline > targetLength-200 {
		breakPoint = lastNewline
	} else if lastPeriod > targetLength-100 {
		breakPoint = lastPeriod + 1
	}
	
	result := content[:breakPoint]
	result += "\n\n*Note: Response has been truncated due to size limits. Use more specific queries for detailed information.*"
	
	return result
}

// StreamResponse handles streaming responses for large content
type StreamResponse struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
	HasMore bool         `json:"hasMore,omitempty"`
	Token   string       `json:"token,omitempty"`
}

// CreateStreamedResponse creates a streamed response for large content
func (ro *ResponseOptimizer) CreateStreamedResponse(content string, chunkSize int) []StreamResponse {
	if len(content) <= chunkSize*4 { // If content is small enough, return single response
		return []StreamResponse{{
			Content: []MCPContent{{
				Type: "text",
				Text: content,
			}},
			HasMore: false,
		}}
	}

	// Split content into chunks
	var chunks []StreamResponse
	lines := strings.Split(content, "\n")
	
	var currentChunk strings.Builder
	currentSize := 0
	chunkIndex := 0
	
	for _, line := range lines {
		lineSize := len(line) + 1 // +1 for newline
		
		if currentSize+lineSize > chunkSize*4 && currentChunk.Len() > 0 {
			// Create chunk
			chunks = append(chunks, StreamResponse{
				Content: []MCPContent{{
					Type: "text",
					Text: currentChunk.String(),
				}},
				HasMore: true,
				Token:   generateStreamToken(chunkIndex),
			})
			
			// Reset for next chunk
			currentChunk.Reset()
			currentSize = 0
			chunkIndex++
		}
		
		currentChunk.WriteString(line + "\n")
		currentSize += lineSize
	}
	
	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, StreamResponse{
			Content: []MCPContent{{
				Type: "text",
				Text: currentChunk.String(),
			}},
			HasMore: false,
			Token:   generateStreamToken(chunkIndex),
		})
	}
	
	ro.logger.Info("Created streamed response", map[string]interface{}{
		"total_chunks": len(chunks),
		"content_size": len(content),
	})
	
	return chunks
}

// generateStreamToken generates a unique token for streaming
func generateStreamToken(index int) string {
	return fmt.Sprintf("stream_%d_%d", time.Now().Unix(), index)
}

// Helper function to estimate token count
func EstimateTokenCount(text string) int {
	// Rough approximation: 1 token ≈ 4 characters
	// This is a simplified estimate; real tokenization would be more accurate
	return len(text) / 4
}