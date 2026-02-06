package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/DevAnuragT/context_keeper/internal/middleware"
	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/DevAnuragT/context_keeper/internal/services"
)

// ProjectHandlers contains project-related HTTP handlers
type ProjectHandlers struct {
	projectSvc     services.ProjectWorkspaceService
	permissionSvc  services.PermissionService
	contextSvc     services.ContextService
	knowledgeSvc   services.KnowledgeGraphService
}

// NewProjectHandlers creates a new project handlers instance
func NewProjectHandlers(
	projectSvc services.ProjectWorkspaceService,
	permissionSvc services.PermissionService,
	contextSvc services.ContextService,
	knowledgeSvc services.KnowledgeGraphService,
) *ProjectHandlers {
	return &ProjectHandlers{
		projectSvc:    projectSvc,
		permissionSvc: permissionSvc,
		contextSvc:    contextSvc,
		knowledgeSvc:  knowledgeSvc,
	}
}

// HandleCreateProject creates a new project workspace
// POST /api/projects
func (h *ProjectHandlers) HandleCreateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	var req services.CreateProjectWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project name is required")
		return
	}

	// Create project workspace
	project, err := h.projectSvc.CreateProjectWorkspace(r.Context(), user.ID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "project_creation_error", fmt.Sprintf("Failed to create project: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

// HandleGetProjects retrieves all projects for the authenticated user
// GET /api/projects
func (h *ProjectHandlers) HandleGetProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Get user's project workspaces
	projects, err := h.projectSvc.GetUserProjectWorkspaces(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database_error", "Failed to get projects")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"projects": projects,
	})
}

// HandleGetProject retrieves a specific project
// GET /api/projects/{id}
func (h *ProjectHandlers) HandleGetProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	// Get project workspace
	project, err := h.projectSvc.GetProjectWorkspace(r.Context(), user.ID, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") {
			writeError(w, http.StatusForbidden, "access_denied", err.Error())
			return
		}
		writeError(w, http.StatusNotFound, "project_not_found", "Project not found")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

// HandleUpdateProject updates a project workspace
// PUT /api/projects/{id}
func (h *ProjectHandlers) HandleUpdateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	var req services.UpdateProjectWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	// Update project workspace
	project, err := h.projectSvc.UpdateProjectWorkspace(r.Context(), user.ID, projectID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") || strings.Contains(err.Error(), "admin access required") {
			writeError(w, http.StatusForbidden, "access_denied", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "project_update_error", fmt.Sprintf("Failed to update project: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, project)
}

// HandleDeleteProject deletes a project workspace
// DELETE /api/projects/{id}
func (h *ProjectHandlers) HandleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	// Delete project workspace
	err := h.projectSvc.DeleteProjectWorkspace(r.Context(), user.ID, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "only project owner") {
			writeError(w, http.StatusForbidden, "access_denied", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "project_deletion_error", fmt.Sprintf("Failed to delete project: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Project deleted successfully",
	})
}

// HandleGetProjectMembers retrieves project members
// GET /api/projects/{id}/members
func (h *ProjectHandlers) HandleGetProjectMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	// Get project members
	members, err := h.projectSvc.GetProjectMembers(r.Context(), user.ID, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") {
			writeError(w, http.StatusForbidden, "access_denied", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "database_error", "Failed to get project members")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"members": members,
	})
}

// HandleAddProjectMember adds a member to a project
// POST /api/projects/{id}/members
func (h *ProjectHandlers) HandleAddProjectMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	var req services.AddProjectMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "User ID is required")
		return
	}

	// Add project member
	member, err := h.projectSvc.AddProjectMember(r.Context(), user.ID, projectID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "admin access required") {
			writeError(w, http.StatusForbidden, "access_denied", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "user_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "member_addition_error", fmt.Sprintf("Failed to add project member: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, member)
}

// HandleProjectContextQuery processes context queries for a project
// POST /api/projects/{id}/context/query
func (h *ProjectHandlers) HandleProjectContextQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	// Check project access
	canAccess, err := h.permissionSvc.CanAccessProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check project access")
		return
	}
	if !canAccess {
		writeError(w, http.StatusForbidden, "access_denied", "Access denied to project")
		return
	}

	// Parse request body
	var req struct {
		Query string `json:"query"`
		Mode  string `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Query required")
		return
	}

	// Default mode if not specified
	if req.Mode == "" {
		req.Mode = "query"
	}

	// Process project context query
	response, err := h.contextSvc.ProcessQueryByProject(r.Context(), projectID, req.Query, req.Mode)
	if err != nil {
		// Check if it's a timeout error
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			writeError(w, http.StatusGatewayTimeout, "ai_service_timeout", "AI service request timed out")
			return
		}

		// Check if it's an AI service error
		if strings.Contains(err.Error(), "AI service") {
			writeError(w, http.StatusBadGateway, "ai_service_error", fmt.Sprintf("AI service error: %v", err))
			return
		}

		writeError(w, http.StatusInternalServerError, "context_query_error", fmt.Sprintf("Failed to process context query: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleProjectKnowledgeSearch performs knowledge graph search within a project
// POST /api/projects/{id}/knowledge/search
func (h *ProjectHandlers) HandleProjectKnowledgeSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	// Check project access
	canAccess, err := h.permissionSvc.CanAccessProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check project access")
		return
	}
	if !canAccess {
		writeError(w, http.StatusForbidden, "access_denied", "Access denied to project")
		return
	}

	var req models.KnowledgeGraphQuery
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Query required")
		return
	}

	// Perform project-scoped knowledge search
	results, err := h.knowledgeSvc.SearchKnowledgeByProject(r.Context(), projectID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search_error", fmt.Sprintf("Failed to search knowledge: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

// HandleProjectFileContext retrieves file context within a project
// GET /api/projects/{id}/files/context?file_path={path}
func (h *ProjectHandlers) HandleProjectFileContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	// Check project access
	canAccess, err := h.permissionSvc.CanAccessProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check project access")
		return
	}
	if !canAccess {
		writeError(w, http.StatusForbidden, "access_denied", "Access denied to project")
		return
	}

	// Get file path from query parameter
	filePath := r.URL.Query().Get("file_path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "file_path parameter required")
		return
	}

	// Get file context within project
	context, err := h.knowledgeSvc.GetContextForFileByProject(r.Context(), projectID, filePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "context_error", fmt.Sprintf("Failed to get file context: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, context)
}

// HandleProjectDecisionHistory retrieves decision history within a project
// GET /api/projects/{id}/decisions/history?target={target}
func (h *ProjectHandlers) HandleProjectDecisionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Get user from context
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	// Extract project ID from URL
	projectID := extractProjectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid project ID")
		return
	}

	// Check project access
	canAccess, err := h.permissionSvc.CanAccessProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check project access")
		return
	}
	if !canAccess {
		writeError(w, http.StatusForbidden, "access_denied", "Access denied to project")
		return
	}

	// Get target from query parameter
	target := r.URL.Query().Get("target")
	if target == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "target parameter required")
		return
	}

	// Get decision history within project
	history, err := h.knowledgeSvc.GetDecisionHistoryByProject(r.Context(), projectID, target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "decision_error", fmt.Sprintf("Failed to get decision history: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, history)
}

// Helper function to extract project ID from URL path
func extractProjectIDFromPath(path string) string {
	// Remove /api/projects/ prefix
	path = strings.TrimPrefix(path, "/api/projects/")
	
	// Split by / and take the first part as project ID
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	
	return ""
}