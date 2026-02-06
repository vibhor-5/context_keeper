package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// MCPServer implements the Model Context Protocol JSON-RPC 2.0 server
type MCPServer struct {
	knowledgeGraph     KnowledgeGraphService
	contextService     ContextService
	responseOptimizer  *ResponseOptimizer
	logger             Logger
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(knowledgeGraph KnowledgeGraphService, contextService ContextService, logger Logger) *MCPServer {
	// Default to 4000 tokens max response size
	responseOptimizer := NewResponseOptimizer(4000, logger)
	
	return &MCPServer{
		knowledgeGraph:    knowledgeGraph,
		contextService:    contextService,
		responseOptimizer: responseOptimizer,
		logger:            logger,
	}
}

// JSON-RPC 2.0 request structure
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
	ID      interface{}            `json:"id"`
}

// JSON-RPC 2.0 response structure
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// JSON-RPC 2.0 error structure
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Tool definition structure
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCP Tool result structure
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCP Content structure
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// ServeHTTP implements the HTTP handler for MCP JSON-RPC requests
func (m *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendError(w, ParseError, "Parse error", nil, nil)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		m.sendError(w, InvalidRequest, "Invalid Request", nil, req.ID)
		return
	}

	ctx := r.Context()
	response := m.handleRequest(ctx, &req)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRequest processes a JSON-RPC request and returns a response
func (m *MCPServer) handleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "tools/list":
		return m.handleToolsList(ctx, req)
	case "tools/call":
		return m.handleToolsCall(ctx, req)
	case "initialize":
		return m.handleInitialize(ctx, req)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    MethodNotFound,
				Message: "Method not found",
			},
			ID: req.ID,
		}
	}
}

// handleInitialize handles MCP initialization
func (m *MCPServer) handleInitialize(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "context-keeper-mcp",
			"version": "1.0.0",
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

// handleToolsList returns the list of available MCP tools
func (m *MCPServer) handleToolsList(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	tools := []MCPTool{
		{
			Name:        "search_project_knowledge",
			Description: "Search across all project knowledge including decisions, discussions, and file contexts",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query for project knowledge",
					},
					"entity_types": map[string]interface{}{
						"type":        "array",
						"description": "Filter by entity types (decision, discussion, feature, file_context)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"platforms": map[string]interface{}{
						"type":        "array",
						"description": "Filter by platforms (github, slack, discord)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return",
						"default":     10,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_context_for_file",
			Description: "Get comprehensive context information for a specific file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file relative to repository root",
					},
					"include_history": map[string]interface{}{
						"type":        "boolean",
						"description": "Include change history and related PRs",
						"default":     true,
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			Name:        "get_decision_history",
			Description: "Get decision history for a feature or file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]interface{}{
						"type":        "string",
						"description": "Feature name or file path to get decision history for",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of decisions to return",
						"default":     20,
					},
				},
				"required": []string{"target"},
			},
		},
		{
			Name:        "list_recent_architecture_discussions",
			Description: "List recent architecture-related discussions",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of discussions to return",
						"default":     10,
					},
					"days_back": map[string]interface{}{
						"type":        "integer",
						"description": "Number of days to look back for discussions",
						"default":     30,
					},
				},
			},
		},
		{
			Name:        "explain_why_code_exists",
			Description: "Explain why specific code exists based on historical context",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to explain",
					},
					"line_range": map[string]interface{}{
						"type":        "object",
						"description": "Optional line range to focus on",
						"properties": map[string]interface{}{
							"start": map[string]interface{}{
								"type": "integer",
							},
							"end": map[string]interface{}{
								"type": "integer",
							},
						},
					},
				},
				"required": []string{"file_path"},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

// handleToolsCall executes a specific MCP tool
func (m *MCPServer) handleToolsCall(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	// Extract tool name and arguments
	name, ok := req.Params["name"].(string)
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    InvalidParams,
				Message: "Missing or invalid tool name",
			},
			ID: req.ID,
		}
	}

	arguments, ok := req.Params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	// Execute the tool
	result, err := m.callTool(ctx, name, arguments)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    InternalError,
				Message: err.Error(),
			},
			ID: req.ID,
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

// callTool executes a specific tool by name
func (m *MCPServer) callTool(ctx context.Context, name string, arguments map[string]interface{}) (*MCPToolResult, error) {
	m.logger.Info("Executing MCP tool", map[string]interface{}{
		"tool":      name,
		"arguments": arguments,
	})

	switch name {
	case "search_project_knowledge":
		return m.searchProjectKnowledge(ctx, arguments)
	case "get_context_for_file":
		return m.getContextForFile(ctx, arguments)
	case "get_decision_history":
		return m.getDecisionHistory(ctx, arguments)
	case "list_recent_architecture_discussions":
		return m.listRecentArchitectureDiscussions(ctx, arguments)
	case "explain_why_code_exists":
		return m.explainWhyCodeExists(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// sendError sends a JSON-RPC error response
func (m *MCPServer) sendError(w http.ResponseWriter, code int, message string, data interface{}, id interface{}) {
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors are still HTTP 200
	json.NewEncoder(w).Encode(response)
}

// Helper function to get string parameter with default
func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultValue
}

// Helper function to get int parameter with default
func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	if val, ok := params[key].(int); ok {
		return val
	}
	return defaultValue
}

// Helper function to get bool parameter with default
func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	return defaultValue
}

// Helper function to get string array parameter
func getStringArrayParam(params map[string]interface{}, key string) []string {
	if val, ok := params[key].([]interface{}); ok {
		result := make([]string, 0, len(val))
		for _, item := range val {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}