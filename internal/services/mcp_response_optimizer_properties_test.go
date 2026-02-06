package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// **Property 18: Response Size Management**
// **Validates: Requirements 7.6, 7.7**
//
// This property verifies that the response optimizer:
// 1. Correctly estimates token counts and applies optimization when needed
// 2. Maintains content structure and readability after optimization
// 3. Provides appropriate truncation messages to users
// 4. Handles different content types with appropriate strategies
// 5. Creates properly structured streaming responses for large content
func TestProperty_ResponseSizeManagement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create response optimizer with small token limit for testing
		logger := &SimpleLogger{}
		maxTokens := rapid.IntRange(100, 1000).Draw(t, "max_tokens")
		optimizer := NewResponseOptimizer(maxTokens, logger)

		// Generate test content of various sizes
		contentType := rapid.SampledFrom([]string{
			"search_results",
			"file_context", 
			"decision_history",
			"architecture_discussions",
			"code_explanation",
			"generic",
		}).Draw(t, "content_type")

		// Generate content that may or may not exceed token limits
		contentSize := rapid.IntRange(50, 5000).Draw(t, "content_size")
		content := generateTestContent(t, contentType, contentSize)

		// Test 1: Response optimization should respect token limits
		optimizedContent := optimizer.OptimizeResponse(content, contentType)
		
		// If original content was large, optimized should be smaller or equal to limit
		originalTokens := EstimateTokenCount(content)
		if originalTokens > maxTokens {
			// The optimizer should attempt to reduce content size, but may add truncation messages
			// For very large content, we expect some optimization to occur
			if len(content) > maxTokens * 8 { // Very large content
				// Should show some sign of optimization (either smaller or with truncation indicators)
				containsTruncationIndicator := strings.Contains(optimizedContent, "truncated") ||
					strings.Contains(optimizedContent, "summarized") ||
					strings.Contains(optimizedContent, "brevity") ||
					strings.Contains(optimizedContent, "...") ||
					strings.Contains(optimizedContent, "[Additional") ||
					strings.Contains(optimizedContent, "[Summary")
				
				if len(optimizedContent) >= len(content) {
					assert.True(t, containsTruncationIndicator, "Large content that wasn't reduced should contain truncation indicators")
				}
				
				// Should not be more than double the original size (accounting for added messages)
				assert.Less(t, len(optimizedContent), len(content)*2, "Optimized content should not be excessively larger")
			}
		} else {
			// If original was within limits, it should be unchanged or minimally changed
			assert.LessOrEqual(t, len(optimizedContent), len(content)*2, "Small content should not be significantly expanded")
		}

		// Test 2: Optimized content should maintain structure
		if originalTokens > maxTokens {
			// Should contain some form of truncation notice or optimization indicator
			containsTruncationIndicator := strings.Contains(optimizedContent, "truncated") ||
				strings.Contains(optimizedContent, "summarized") ||
				strings.Contains(optimizedContent, "brevity") ||
				strings.Contains(optimizedContent, "...") ||
				strings.Contains(optimizedContent, "[Additional") ||
				strings.Contains(optimizedContent, "[Summary")
			
			assert.True(t, containsTruncationIndicator, "Large content should contain some truncation or optimization indicator")
			
			// Should maintain markdown structure for appropriate content types
			if strings.Contains(content, "#") {
				assert.Contains(t, optimizedContent, "#", "Should preserve markdown headers")
			}
			
			// Should maintain basic content structure
			switch contentType {
			case "search_results":
				if strings.Contains(content, "Search Results") {
					assert.Contains(t, optimizedContent, "Search Results", "Should preserve search results header")
				}
			case "file_context":
				if strings.Contains(content, "Context for File") {
					assert.Contains(t, optimizedContent, "Context for File", "Should preserve file context header")
				}
			case "decision_history":
				if strings.Contains(content, "Decision History") {
					assert.Contains(t, optimizedContent, "Decision History", "Should preserve decision history header")
				}
			case "architecture_discussions":
				if strings.Contains(content, "Architecture Discussions") {
					assert.Contains(t, optimizedContent, "Architecture Discussions", "Should preserve discussions header")
				}
			case "code_explanation":
				if strings.Contains(content, "Why") && strings.Contains(content, "Exists") {
					assert.True(t, strings.Contains(optimizedContent, "Why") || strings.Contains(optimizedContent, "Exists"), "Should preserve explanation context")
				}
			}
		}

		// Test 3: Streaming response creation
		chunkSize := rapid.IntRange(50, 500).Draw(t, "chunk_size")
		streamResponses := optimizer.CreateStreamedResponse(content, chunkSize)
		
		// Should have at least one response
		assert.Greater(t, len(streamResponses), 0, "Should create at least one stream response")
		
		// All responses should have valid structure
		totalContent := ""
		for i, response := range streamResponses {
			assert.NotNil(t, response.Content, "Stream response should have content")
			assert.Greater(t, len(response.Content), 0, "Stream response should have non-empty content")
			assert.Equal(t, "text", response.Content[0].Type, "Stream content should be text type")
			assert.NotEmpty(t, response.Content[0].Text, "Stream text should not be empty")
			
			// Check HasMore flag
			if i < len(streamResponses)-1 {
				assert.True(t, response.HasMore, "Non-final responses should have HasMore=true")
				assert.NotEmpty(t, response.Token, "Non-final responses should have token")
			} else {
				assert.False(t, response.HasMore, "Final response should have HasMore=false")
			}
			
			totalContent += response.Content[0].Text
		}
		
		// Reconstructed content should match original (for small content) or be reasonable (for large content)
		if len(content) <= chunkSize*4 {
			// Small content should be in single response
			assert.Equal(t, 1, len(streamResponses), "Small content should result in single response")
			assert.Equal(t, content, streamResponses[0].Content[0].Text, "Small content should be unchanged")
		} else {
			// Large content should be properly chunked
			assert.Greater(t, len(streamResponses), 1, "Large content should be chunked")
			// Total reconstructed content should be close to original
			assert.InDelta(t, len(content), len(totalContent), float64(len(content))*0.1, "Reconstructed content should be similar length")
		}

		// Test 4: Token estimation accuracy
		estimatedTokens := EstimateTokenCount(content)
		// Token estimation should be reasonable (rough approximation)
		expectedTokens := len(content) / 4 // Our estimation formula
		assert.Equal(t, expectedTokens, estimatedTokens, "Token estimation should match expected formula")
		assert.Greater(t, estimatedTokens, 0, "Token count should be positive for non-empty content")
	})
}

// Helper function to generate test content of various types and sizes
func generateTestContent(t *rapid.T, contentType string, targetSize int) string {
	var content strings.Builder
	
	switch contentType {
	case "search_results":
		content.WriteString("# Search Results for: test query\n\n")
		content.WriteString("Found 5 results:\n\n")
		for i := 0; i < 3; i++ {
			content.WriteString("## Result " + string(rune('1'+i)) + "\n")
			content.WriteString("**Type:** discussion\n")
			content.WriteString("**Platform:** github\n")
			content.WriteString("**Similarity:** 0.85\n\n")
			content.WriteString("**Content:**\n")
			content.WriteString(strings.Repeat("This is sample content for search results. ", targetSize/100))
			content.WriteString("\n\n---\n\n")
		}
		
	case "file_context":
		content.WriteString("# Context for File: test/file.go\n\n")
		content.WriteString("## File Context History\n\n")
		for i := 0; i < 2; i++ {
			content.WriteString("### Context Entry " + string(rune('1'+i)) + "\n")
			content.WriteString("**Change Reason:** Added new feature\n")
			content.WriteString("**Discussion Context:** ")
			content.WriteString(strings.Repeat("Context information about file changes. ", targetSize/200))
			content.WriteString("\n\n")
		}
		
	case "decision_history":
		content.WriteString("# Decision History for: feature\n\n")
		content.WriteString("Found 3 decisions:\n\n")
		for i := 0; i < 2; i++ {
			content.WriteString("## Decision " + string(rune('1'+i)) + "\n")
			content.WriteString("**Created:** 2024-01-01\n")
			content.WriteString("**Status:** accepted\n\n")
			content.WriteString("**Decision:**\n")
			content.WriteString(strings.Repeat("We decided to implement this approach because it provides better performance. ", targetSize/300))
			content.WriteString("\n\n**Rationale:**\n")
			content.WriteString(strings.Repeat("The rationale for this decision includes various technical considerations. ", targetSize/400))
			content.WriteString("\n\n---\n\n")
		}
		
	case "architecture_discussions":
		content.WriteString("# Recent Architecture Discussions (Last 30 days)\n\n")
		content.WriteString("Found 2 discussions:\n\n")
		for i := 0; i < 2; i++ {
			content.WriteString("## Discussion " + string(rune('1'+i)) + "\n")
			content.WriteString("**Platform:** slack\n")
			content.WriteString("**Thread ID:** thread123\n\n")
			content.WriteString("**Summary:**\n")
			content.WriteString(strings.Repeat("Discussion about system architecture and design decisions. ", targetSize/200))
			content.WriteString("\n\n---\n\n")
		}
		
	case "code_explanation":
		content.WriteString("# Why test/file.go Exists\n\n")
		content.WriteString("## Historical Context\n\n")
		content.WriteString("### 2024-01-01\n")
		content.WriteString("**Reason for Change:** Initial implementation\n\n")
		content.WriteString("**Context:** ")
		content.WriteString(strings.Repeat("This file was created to handle specific functionality in the system. ", targetSize/100))
		content.WriteString("\n\n## Summary\n\n")
		content.WriteString(strings.Repeat("This code exists as a result of architectural decisions and requirements. ", targetSize/200))
		
	default: // generic
		content.WriteString("# Generic Content\n\n")
		content.WriteString(strings.Repeat("This is generic content that can be of any type. It contains various information and details. ", targetSize/50))
	}
	
	// Ensure we reach approximately the target size
	currentSize := content.Len()
	if currentSize < targetSize {
		padding := strings.Repeat("Additional content to reach target size. ", (targetSize-currentSize)/40+1)
		content.WriteString("\n\n")
		remainingSize := targetSize - currentSize
		if len(padding) > remainingSize {
			content.WriteString(padding[:remainingSize])
		} else {
			content.WriteString(padding)
		}
	}
	
	return content.String()
}