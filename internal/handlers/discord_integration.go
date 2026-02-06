package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/DevAnuragT/context_keeper/internal/middleware"
	"github.com/DevAnuragT/context_keeper/internal/services"
)

// DiscordIntegrationHandlers contains handlers for Discord integration management
type DiscordIntegrationHandlers struct {
	authSvc                services.AuthService
	discordIntegrationSvc  services.DiscordIntegrationService
	permissionSvc          services.PermissionService
}

// NewDiscordIntegrationHandlers creates new Discord integration handlers
func NewDiscordIntegrationHandlers(
	authSvc services.AuthService,
	discordIntegrationSvc services.DiscordIntegrationService,
	permissionSvc services.PermissionService,
) *DiscordIntegrationHandlers {
	return &DiscordIntegrationHandlers{
		authSvc:               authSvc,
		discordIntegrationSvc: discordIntegrationSvc,
		permissionSvc:         permissionSvc,
	}
}

// HandleDiscordBotInstallation handles Discord bot installation
// POST /api/projects/{project_id}/integrations/discord/bot/install
func (h *DiscordIntegrationHandlers) HandleDiscordBotInstallation(w http.ResponseWriter, r *http.Request) {
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
	projectID := extractProjectIDFromIntegrationPath(r.URL.Path, "/api/projects/", "/integrations/discord/bot/install")
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
	var req services.DiscordBotInstallationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	req.ProjectID = projectID // Ensure consistency

	if req.BotToken == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Bot token required")
		return
	}

	// Process Discord bot installation
	integration, err := h.discordIntegrationSvc.ProcessBotInstallation(r.Context(), &req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, "integration_exists", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "validate") {
			writeError(w, http.StatusBadRequest, "invalid_token", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "installation_error", fmt.Sprintf("Failed to process installation: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, integration)
}

// HandleGetAvailableServers gets available Discord servers for selection
// GET /api/projects/{project_id}/integrations/discord/{integration_id}/servers
func (h *DiscordIntegrationHandlers) HandleGetAvailableServers(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndDiscordIntegrationIDs(r.URL.Path)
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

	// Get available servers
	servers, err := h.discordIntegrationSvc.GetAvailableServers(r.Context(), projectID, integrationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "token") {
			writeError(w, http.StatusUnauthorized, "integration_unauthorized", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", fmt.Sprintf("Failed to get servers: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"servers": servers,
	})
}

// HandleGetAvailableChannels gets available channels for a Discord server
// GET /api/projects/{project_id}/integrations/discord/{integration_id}/servers/{guild_id}/channels
func (h *DiscordIntegrationHandlers) HandleGetAvailableChannels(w http.ResponseWriter, r *http.Request) {
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

	// Extract project ID, integration ID, and guild ID from URL path
	projectID, integrationID, guildID := extractProjectIntegrationAndGuildIDs(r.URL.Path)
	if projectID == "" || integrationID == "" || guildID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Project ID, Integration ID, and Guild ID required")
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
	channels, err := h.discordIntegrationSvc.GetAvailableChannels(r.Context(), projectID, integrationID, guildID)
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
// POST /api/projects/{project_id}/integrations/discord/{integration_id}/channels/select
func (h *DiscordIntegrationHandlers) HandleSelectChannels(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndDiscordIntegrationIDs(r.URL.Path)
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
	var req services.DiscordChannelSelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	req.ProjectID = projectID         // Ensure consistency
	req.IntegrationID = integrationID // Ensure consistency

	if req.GuildID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Guild ID required")
		return
	}

	if len(req.ChannelIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "At least one channel ID required")
		return
	}

	// Select channels
	dataSources, err := h.discordIntegrationSvc.SelectChannels(r.Context(), &req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid channel") || strings.Contains(err.Error(), "does not belong") {
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

// HandleUpdateDiscordConfiguration updates Discord integration configuration
// PUT /api/projects/{project_id}/integrations/discord/{integration_id}/configuration
func (h *DiscordIntegrationHandlers) HandleUpdateDiscordConfiguration(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndDiscordIntegrationIDs(r.URL.Path)
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
	integration, err := h.discordIntegrationSvc.UpdateConfiguration(r.Context(), &req, user.ID)
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

// HandleGetDiscordIntegrationStatus gets Discord integration status and health
// GET /api/projects/{project_id}/integrations/discord/{integration_id}/status
func (h *DiscordIntegrationHandlers) HandleGetDiscordIntegrationStatus(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndDiscordIntegrationIDs(r.URL.Path)
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
	status, err := h.discordIntegrationSvc.GetIntegrationStatus(r.Context(), projectID, integrationID)
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

// HandleDeleteDiscordIntegration deletes a Discord integration
// DELETE /api/projects/{project_id}/integrations/discord/{integration_id}
func (h *DiscordIntegrationHandlers) HandleDeleteDiscordIntegration(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndDiscordIntegrationIDs(r.URL.Path)
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
	err = h.discordIntegrationSvc.DeleteIntegration(r.Context(), projectID, integrationID, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "deletion_error", fmt.Sprintf("Failed to delete integration: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Discord integration deleted successfully",
	})
}

// HandleValidateBotConnection validates Discord bot connection
// POST /api/projects/{project_id}/integrations/discord/{integration_id}/validate
func (h *DiscordIntegrationHandlers) HandleValidateBotConnection(w http.ResponseWriter, r *http.Request) {
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
	projectID, integrationID := extractProjectAndDiscordIntegrationIDs(r.URL.Path)
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

	// Validate bot connection
	err = h.discordIntegrationSvc.ValidateBotConnection(r.Context(), integrationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "integration_not_found", err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("Bot connection validation failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":   true,
		"message": "Bot connection is valid",
	})
}

// Helper functions

// extractProjectAndDiscordIntegrationIDs extracts project ID and integration ID from Discord URL path
func extractProjectAndDiscordIntegrationIDs(path string) (projectID, integrationID string) {
	// Expected format: /api/projects/{project_id}/integrations/discord/{integration_id}/...
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	if len(parts) < 6 {
		return "", ""
	}
	
	if parts[0] != "api" || parts[1] != "projects" || parts[3] != "integrations" || parts[4] != "discord" {
		return "", ""
	}
	
	return parts[2], parts[5]
}

// extractProjectIntegrationAndGuildIDs extracts project ID, integration ID, and guild ID from URL path
func extractProjectIntegrationAndGuildIDs(path string) (projectID, integrationID, guildID string) {
	// Expected format: /api/projects/{project_id}/integrations/discord/{integration_id}/servers/{guild_id}/channels
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	if len(parts) < 8 {
		return "", "", ""
	}
	
	if parts[0] != "api" || parts[1] != "projects" || parts[3] != "integrations" || parts[4] != "discord" || parts[6] != "servers" {
		return "", "", ""
	}
	
	return parts[2], parts[5], parts[7]
}
