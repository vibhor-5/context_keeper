package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/DevAnuragT/context_keeper/internal/middleware"
	"github.com/DevAnuragT/context_keeper/internal/services"
)

// GitHubIntegrationHandlers contains handlers for GitHub integration management
type GitHubIntegrationHandlers struct {
	authSvc               services.AuthService
	githubIntegrationSvc  services.GitHubIntegrationService
	permissionSvc         services.PermissionService
}

// NewGitHubIntegrationHandlers creates new GitHub integration handlers
func NewGitHubIntegrationHandlers(
	authSvc services.AuthService,
	githubIntegrationSvc services.GitHubIntegrationService,
	permissionSvc services.PermissionService,
) *GitHubIntegrationHandlers {
	return &GitHubIntegrationHandlers{
		authSvc:              authSvc,
		githubIntegrationSvc: githubIntegrationSvc,
		permissionSvc:        permissionSvc,
	}
}

// HandleGitHubAppInstallation handles GitHub App installation callback
// POST /api/projects/{project_id}/integrations/github/app/install
func (h *GitHubIntegrationHandlers) HandleGitHubAppInstallation(w http.ResponseWriter, r *http.Request) {
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

	// Extract project ID from URL path
	projectID := extractProjectIDFromIntegrationPath(r.URL.Path, "/api/projects/", "/integrations/github/app/install")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project ID required")
		return
	}

	// Check project permissions
	canAdmin, err := h.permissionSvc.CanAdminProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check permissions")
		return
	}
	if !canAdmin {
		writeError(w, http.StatusForbidden, "insufficient_permissions", "Admin access required")
		return
	}

	// Parse request body
	var req services.GitHubInstallationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	req.ProjectID = projectID // Ensure consistency

	if req.InstallationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Installation ID required")
		return
	}

	// Process GitHub App installation
	integration, err := h.githubIntegrationSvc.ProcessAppInstallation(r.Context(), &req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, "integration_exists", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid installation") {
			writeError(w, http.StatusBadRequest, "invalid_installation", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "installation_error", fmt.Sprintf("Failed to process installation: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, integration)
}

// HandleGitHubOAuthInstallation handles GitHub OAuth installation callback
// POST /api/projects/{project_id}/integrations/github/oauth/install
func (h *GitHubIntegrationHandlers) HandleGitHubOAuthInstallation(w http.ResponseWriter, r *http.Request) {
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

	// Extract project ID from URL path
	projectID := extractProjectIDFromIntegrationPath(r.URL.Path, "/api/projects/", "/integrations/github/oauth/install")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project ID required")
		return
	}

	// Check project permissions
	canAdmin, err := h.permissionSvc.CanAdminProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check permissions")
		return
	}
	if !canAdmin {
		writeError(w, http.StatusForbidden, "insufficient_permissions", "Admin access required")
		return
	}

	// Parse request body
	var req services.GitHubOAuthInstallationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	req.ProjectID = projectID // Ensure consistency

	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Authorization code required")
		return
	}

	// Process GitHub OAuth installation
	integration, err := h.githubIntegrationSvc.ProcessOAuthInstallation(r.Context(), &req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, "integration_exists", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid code") || strings.Contains(err.Error(), "OAuth") {
			writeError(w, http.StatusBadRequest, "oauth_error", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "installation_error", fmt.Sprintf("Failed to process installation: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, integration)
}

// HandleGetAvailableRepositories gets available repositories for selection
// GET /api/projects/{project_id}/integrations/github/{integration_id}/repositories
func (h *GitHubIntegrationHandlers) HandleGetAvailableRepositories(w http.ResponseWriter, r *http.Request) {
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

	// Extract project ID and integration ID from URL path
	projectID, integrationID := extractProjectAndIntegrationIDs(r.URL.Path)
	if projectID == "" || integrationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project ID and Integration ID required")
		return
	}

	// Check project permissions
	canRead, err := h.permissionSvc.CanReadProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check permissions")
		return
	}
	if !canRead {
		writeError(w, http.StatusForbidden, "insufficient_permissions", "Read access required")
		return
	}

	// Get available repositories
	repositories, err := h.githubIntegrationSvc.GetAvailableRepositories(r.Context(), projectID, integrationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "token") {
			writeError(w, http.StatusUnauthorized, "integration_unauthorized", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "repository_error", fmt.Sprintf("Failed to get repositories: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"repositories": repositories,
	})
}

// HandleSelectRepositories handles repository selection for integration
// POST /api/projects/{project_id}/integrations/github/{integration_id}/repositories/select
func (h *GitHubIntegrationHandlers) HandleSelectRepositories(w http.ResponseWriter, r *http.Request) {
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

	// Extract project ID and integration ID from URL path
	projectID, integrationID := extractProjectAndIntegrationIDs(r.URL.Path)
	if projectID == "" || integrationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project ID and Integration ID required")
		return
	}

	// Check project permissions
	canAdmin, err := h.permissionSvc.CanAdminProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check permissions")
		return
	}
	if !canAdmin {
		writeError(w, http.StatusForbidden, "insufficient_permissions", "Admin access required")
		return
	}

	// Parse request body
	var req services.RepositorySelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	req.ProjectID = projectID         // Ensure consistency
	req.IntegrationID = integrationID // Ensure consistency

	if len(req.RepositoryIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "At least one repository ID required")
		return
	}

	// Select repositories
	dataSources, err := h.githubIntegrationSvc.SelectRepositories(r.Context(), &req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid repository") {
			writeError(w, http.StatusBadRequest, "invalid_repository", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "selection_error", fmt.Sprintf("Failed to select repositories: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"data_sources": dataSources,
	})
}

// HandleUpdateIntegrationConfiguration updates integration configuration
// PUT /api/projects/{project_id}/integrations/github/{integration_id}/configuration
func (h *GitHubIntegrationHandlers) HandleUpdateIntegrationConfiguration(w http.ResponseWriter, r *http.Request) {
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

	// Extract project ID and integration ID from URL path
	projectID, integrationID := extractProjectAndIntegrationIDs(r.URL.Path)
	if projectID == "" || integrationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project ID and Integration ID required")
		return
	}

	// Check project permissions
	canAdmin, err := h.permissionSvc.CanAdminProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check permissions")
		return
	}
	if !canAdmin {
		writeError(w, http.StatusForbidden, "insufficient_permissions", "Admin access required")
		return
	}

	// Parse request body
	var req services.IntegrationConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	req.ProjectID = projectID         // Ensure consistency
	req.IntegrationID = integrationID // Ensure consistency

	if req.Configuration == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Configuration required")
		return
	}

	// Update configuration
	integration, err := h.githubIntegrationSvc.UpdateConfiguration(r.Context(), &req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid configuration") {
			writeError(w, http.StatusBadRequest, "invalid_configuration", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "configuration_error", fmt.Sprintf("Failed to update configuration: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, integration)
}

// HandleGetIntegrationStatus gets integration status and health
// GET /api/projects/{project_id}/integrations/github/{integration_id}/status
func (h *GitHubIntegrationHandlers) HandleGetIntegrationStatus(w http.ResponseWriter, r *http.Request) {
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

	// Extract project ID and integration ID from URL path
	projectID, integrationID := extractProjectAndIntegrationIDs(r.URL.Path)
	if projectID == "" || integrationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project ID and Integration ID required")
		return
	}

	// Check project permissions
	canRead, err := h.permissionSvc.CanReadProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check permissions")
		return
	}
	if !canRead {
		writeError(w, http.StatusForbidden, "insufficient_permissions", "Read access required")
		return
	}

	// Get integration status
	status, err := h.githubIntegrationSvc.GetIntegrationStatus(r.Context(), projectID, integrationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "status_error", fmt.Sprintf("Failed to get status: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// HandleDeleteIntegration deletes a GitHub integration
// DELETE /api/projects/{project_id}/integrations/github/{integration_id}
func (h *GitHubIntegrationHandlers) HandleDeleteIntegration(w http.ResponseWriter, r *http.Request) {
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

	// Extract project ID and integration ID from URL path
	projectID, integrationID := extractProjectAndIntegrationIDs(r.URL.Path)
	if projectID == "" || integrationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project ID and Integration ID required")
		return
	}

	// Check project permissions
	canAdmin, err := h.permissionSvc.CanAdminProject(r.Context(), user.ID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_error", "Failed to check permissions")
		return
	}
	if !canAdmin {
		writeError(w, http.StatusForbidden, "insufficient_permissions", "Admin access required")
		return
	}

	// Delete integration
	err = h.githubIntegrationSvc.DeleteIntegration(r.Context(), projectID, integrationID, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "deletion_error", fmt.Sprintf("Failed to delete integration: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Integration deleted successfully",
	})
}

// Helper functions

// extractProjectIDFromIntegrationPath extracts project ID from URL path with prefix and suffix
func extractProjectIDFromIntegrationPath(path, prefix, suffix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	path = strings.TrimPrefix(path, prefix)
	
	if !strings.HasSuffix(path, suffix) {
		return ""
	}
	path = strings.TrimSuffix(path, suffix)
	
	return path
}

// extractProjectAndIntegrationIDs extracts project ID and integration ID from URL path
func extractProjectAndIntegrationIDs(path string) (projectID, integrationID string) {
	// Expected format: /api/projects/{project_id}/integrations/github/{integration_id}/...
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	if len(parts) < 6 {
		return "", ""
	}
	
	if parts[0] != "api" || parts[1] != "projects" || parts[3] != "integrations" || parts[4] != "github" {
		return "", ""
	}
	
	return parts[2], parts[5]
}