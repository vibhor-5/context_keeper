package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/DevAnuragT/context_keeper/internal/middleware"
	"github.com/DevAnuragT/context_keeper/internal/services"
)

// SlackIntegrationHandlers contains handlers for Slack integration management
type SlackIntegrationHandlers struct {
	authSvc              services.AuthService
	slackIntegrationSvc  services.SlackIntegrationService
	permissionSvc        services.PermissionService
}

// NewSlackIntegrationHandlers creates new Slack integration handlers
func NewSlackIntegrationHandlers(
	authSvc services.AuthService,
	slackIntegrationSvc services.SlackIntegrationService,
	permissionSvc services.PermissionService,
) *SlackIntegrationHandlers {
	return &SlackIntegrationHandlers{
		authSvc:             authSvc,
		slackIntegrationSvc: slackIntegrationSvc,
		permissionSvc:       permissionSvc,
	}
}

// HandleSlackOAuthInstallation handles Slack OAuth installation callback
// POST /api/projects/{project_id}/integrations/slack/oauth/install
func (h *SlackIntegrationHandlers) HandleSlackOAuthInstallation(w http.ResponseWriter, r *http.Request) {
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
	projectID := extractProjectIDFromIntegrationPath(r.URL.Path, "/api/projects/", "/integrations/slack/oauth/install")
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
	var req services.SlackOAuthInstallationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	req.ProjectID = projectID // Ensure consistency

	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Authorization code required")
		return
	}

	// Process Slack OAuth installation
	integration, err := h.slackIntegrationSvc.ProcessOAuthInstallation(r.Context(), &req, user.ID)
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

// HandleGetAvailableChannels gets available Slack channels for selection
// GET /api/projects/{project_id}/integrations/slack/{integration_id}/channels
func (h *SlackIntegrationHandlers) HandleGetAvailableChannels(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndSlackIntegrationIDs(r.URL.Path)
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

	// Get available channels
	channels, err := h.slackIntegrationSvc.GetAvailableChannels(r.Context(), projectID, integrationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "token") {
			writeError(w, http.StatusUnauthorized, "integration_unauthorized", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "channel_error", fmt.Sprintf("Failed to get channels: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
	})
}

// HandleSelectChannels handles channel selection for integration
// POST /api/projects/{project_id}/integrations/slack/{integration_id}/channels/select
func (h *SlackIntegrationHandlers) HandleSelectChannels(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndSlackIntegrationIDs(r.URL.Path)
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
	var req services.ChannelSelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	req.ProjectID = projectID         // Ensure consistency
	req.IntegrationID = integrationID // Ensure consistency

	if len(req.ChannelIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "At least one channel ID required")
		return
	}

	// Select channels
	dataSources, err := h.slackIntegrationSvc.SelectChannels(r.Context(), &req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid channel") {
			writeError(w, http.StatusBadRequest, "invalid_channel", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "selection_error", fmt.Sprintf("Failed to select channels: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"data_sources": dataSources,
	})
}

// HandleUpdateSlackConfiguration updates Slack integration configuration
// PUT /api/projects/{project_id}/integrations/slack/{integration_id}/configuration
func (h *SlackIntegrationHandlers) HandleUpdateSlackConfiguration(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndSlackIntegrationIDs(r.URL.Path)
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
	integration, err := h.slackIntegrationSvc.UpdateConfiguration(r.Context(), &req, user.ID)
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

// HandleGetSlackIntegrationStatus gets Slack integration status and health
// GET /api/projects/{project_id}/integrations/slack/{integration_id}/status
func (h *SlackIntegrationHandlers) HandleGetSlackIntegrationStatus(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndSlackIntegrationIDs(r.URL.Path)
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
	status, err := h.slackIntegrationSvc.GetIntegrationStatus(r.Context(), projectID, integrationID)
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

// HandleDeleteSlackIntegration deletes a Slack integration
// DELETE /api/projects/{project_id}/integrations/slack/{integration_id}
func (h *SlackIntegrationHandlers) HandleDeleteSlackIntegration(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndSlackIntegrationIDs(r.URL.Path)
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
	err = h.slackIntegrationSvc.DeleteIntegration(r.Context(), projectID, integrationID, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "deletion_error", fmt.Sprintf("Failed to delete integration: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Slack integration deleted successfully",
	})
}

// HandleValidateWorkspaceConnection validates Slack workspace connection
// POST /api/projects/{project_id}/integrations/slack/{integration_id}/validate
func (h *SlackIntegrationHandlers) HandleValidateWorkspaceConnection(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndSlackIntegrationIDs(r.URL.Path)
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

	// Validate workspace connection
	err = h.slackIntegrationSvc.ValidateWorkspaceConnection(r.Context(), integrationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("Workspace connection validation failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":   true,
		"message": "Workspace connection is valid",
	})
}

// Helper functions

// extractProjectAndSlackIntegrationIDs extracts project ID and integration ID from Slack URL path
func extractProjectAndSlackIntegrationIDs(path string) (projectID, integrationID string) {
	// Expected format: /api/projects/{project_id}/integrations/slack/{integration_id}/...
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	if len(parts) < 6 {
		return "", ""
	}
	
	if parts[0] != "api" || parts[1] != "projects" || parts[3] != "integrations" || parts[4] != "slack" {
		return "", ""
	}
	
	return parts[2], parts[5]
}
