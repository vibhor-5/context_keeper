package services

import (
	"context"
	"fmt"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// ProjectWorkspaceService handles project workspace operations
type ProjectWorkspaceService interface {
	// Project workspace management
	CreateProjectWorkspace(ctx context.Context, userID string, req *CreateProjectWorkspaceRequest) (*models.ProjectWorkspace, error)
	GetProjectWorkspace(ctx context.Context, userID, projectID string) (*models.ProjectWorkspace, error)
	GetUserProjectWorkspaces(ctx context.Context, userID string) ([]models.ProjectWorkspace, error)
	UpdateProjectWorkspace(ctx context.Context, userID, projectID string, updates *UpdateProjectWorkspaceRequest) (*models.ProjectWorkspace, error)
	DeleteProjectWorkspace(ctx context.Context, userID, projectID string) error
	
	// Project member management
	AddProjectMember(ctx context.Context, userID, projectID string, req *AddProjectMemberRequest) (*models.ProjectMember, error)
	GetProjectMembers(ctx context.Context, userID, projectID string) ([]models.ProjectMember, error)
	UpdateProjectMember(ctx context.Context, userID, projectID, memberUserID string, updates *UpdateProjectMemberRequest) (*models.ProjectMember, error)
	RemoveProjectMember(ctx context.Context, userID, projectID, memberUserID string) error
	
	// Project integration management
	CreateProjectIntegration(ctx context.Context, userID, projectID string, req *CreateProjectIntegrationRequest) (*models.ProjectIntegration, error)
	GetProjectIntegrations(ctx context.Context, userID, projectID string) ([]models.ProjectIntegration, error)
	GetProjectIntegration(ctx context.Context, userID, projectID, integrationID string) (*models.ProjectIntegration, error)
	UpdateProjectIntegration(ctx context.Context, userID, projectID, integrationID string, updates *UpdateProjectIntegrationRequest) (*models.ProjectIntegration, error)
	DeleteProjectIntegration(ctx context.Context, userID, projectID, integrationID string) error
	
	// Project data source management
	CreateProjectDataSource(ctx context.Context, userID, projectID, integrationID string, req *CreateProjectDataSourceRequest) (*models.ProjectDataSource, error)
	GetProjectDataSources(ctx context.Context, userID, projectID string) ([]models.ProjectDataSource, error)
	UpdateProjectDataSource(ctx context.Context, userID, projectID, dataSourceID string, updates *UpdateProjectDataSourceRequest) (*models.ProjectDataSource, error)
	DeleteProjectDataSource(ctx context.Context, userID, projectID, dataSourceID string) error
}

// Request/Response models for project workspace operations

type CreateProjectWorkspaceRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description *string                `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

type UpdateProjectWorkspaceRequest struct {
	Name        *string                `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string                `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

type AddProjectMemberRequest struct {
	UserID      string                 `json:"user_id" validate:"required"`
	Role        models.UserRole        `json:"role" validate:"required,oneof=admin member viewer"`
	Permissions map[string]interface{} `json:"permissions,omitempty"`
}

type UpdateProjectMemberRequest struct {
	Role        *models.UserRole       `json:"role,omitempty" validate:"omitempty,oneof=admin member viewer"`
	Permissions map[string]interface{} `json:"permissions,omitempty"`
}

type CreateProjectIntegrationRequest struct {
	Platform        models.Platform        `json:"platform" validate:"required,oneof=github slack discord"`
	IntegrationType models.IntegrationType `json:"integration_type" validate:"required,oneof=oauth bot webhook"`
	Configuration   map[string]interface{} `json:"configuration,omitempty"`
	Credentials     map[string]interface{} `json:"credentials,omitempty"`
}

type UpdateProjectIntegrationRequest struct {
	Status        *models.IntegrationStatus `json:"status,omitempty" validate:"omitempty,oneof=active inactive error pending"`
	Configuration map[string]interface{}    `json:"configuration,omitempty"`
	Credentials   map[string]interface{}    `json:"credentials,omitempty"`
	ErrorMessage  *string                   `json:"error_message,omitempty"`
}

type CreateProjectDataSourceRequest struct {
	SourceType    models.SourceType      `json:"source_type" validate:"required,oneof=repository channel server"`
	SourceID      string                 `json:"source_id" validate:"required"`
	SourceName    string                 `json:"source_name" validate:"required"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	IsActive      bool                   `json:"is_active"`
}

type UpdateProjectDataSourceRequest struct {
	SourceName      *string                `json:"source_name,omitempty"`
	Configuration   map[string]interface{} `json:"configuration,omitempty"`
	IsActive        *bool                  `json:"is_active,omitempty"`
	IngestionStatus *string                `json:"ingestion_status,omitempty"`
	ErrorMessage    *string                `json:"error_message,omitempty"`
}

// ProjectWorkspaceServiceImpl implements the ProjectWorkspaceService interface
type ProjectWorkspaceServiceImpl struct {
	store           RepositoryStore
	permissionSvc   PermissionService
	encryptionSvc   EncryptionService
}

// NewProjectWorkspaceService creates a new project workspace service
func NewProjectWorkspaceService(store RepositoryStore, permissionSvc PermissionService, encryptionSvc EncryptionService) ProjectWorkspaceService {
	return &ProjectWorkspaceServiceImpl{
		store:         store,
		permissionSvc: permissionSvc,
		encryptionSvc: encryptionSvc,
	}
}

// CreateProjectWorkspace creates a new project workspace
func (p *ProjectWorkspaceServiceImpl) CreateProjectWorkspace(ctx context.Context, userID string, req *CreateProjectWorkspaceRequest) (*models.ProjectWorkspace, error) {
	// Create project workspace
	workspace := &models.ProjectWorkspace{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
		Settings:    req.Settings,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if workspace.Settings == nil {
		workspace.Settings = make(map[string]interface{})
	}

	if err := p.store.CreateProjectWorkspace(ctx, workspace); err != nil {
		return nil, fmt.Errorf("failed to create project workspace: %w", err)
	}

	// Add owner as project member
	ownerMember := &models.ProjectMember{
		ProjectID:   workspace.ID,
		UserID:      userID,
		Role:        string(models.RoleOwner),
		Permissions: make(map[string]interface{}),
		JoinedAt:    &workspace.CreatedAt,
		CreatedAt:   workspace.CreatedAt,
	}

	if err := p.store.CreateProjectMember(ctx, ownerMember); err != nil {
		// Try to clean up the workspace if member creation fails
		p.store.DeleteProjectWorkspace(ctx, workspace.ID)
		return nil, fmt.Errorf("failed to add owner as project member: %w", err)
	}

	return workspace, nil
}

// GetProjectWorkspace retrieves a project workspace
func (p *ProjectWorkspaceServiceImpl) GetProjectWorkspace(ctx context.Context, userID, projectID string) (*models.ProjectWorkspace, error) {
	// Check permissions
	canAccess, err := p.permissionSvc.CanAccessProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project access: %w", err)
	}
	if !canAccess {
		return nil, fmt.Errorf("access denied to project %s", projectID)
	}

	workspace, err := p.store.GetProjectWorkspace(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project workspace: %w", err)
	}

	return workspace, nil
}

// GetUserProjectWorkspaces retrieves all project workspaces for a user
func (p *ProjectWorkspaceServiceImpl) GetUserProjectWorkspaces(ctx context.Context, userID string) ([]models.ProjectWorkspace, error) {
	workspaces, err := p.store.GetProjectWorkspacesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user project workspaces: %w", err)
	}

	return workspaces, nil
}

// UpdateProjectWorkspace updates a project workspace
func (p *ProjectWorkspaceServiceImpl) UpdateProjectWorkspace(ctx context.Context, userID, projectID string, req *UpdateProjectWorkspaceRequest) (*models.ProjectWorkspace, error) {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return nil, fmt.Errorf("admin access required for project %s", projectID)
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Settings != nil {
		updates["settings"] = models.JSONBMap(req.Settings)
	}

	if len(updates) == 0 {
		// No updates, just return current workspace
		return p.GetProjectWorkspace(ctx, userID, projectID)
	}

	if err := p.store.UpdateProjectWorkspace(ctx, projectID, updates); err != nil {
		return nil, fmt.Errorf("failed to update project workspace: %w", err)
	}

	return p.GetProjectWorkspace(ctx, userID, projectID)
}

// DeleteProjectWorkspace deletes a project workspace
func (p *ProjectWorkspaceServiceImpl) DeleteProjectWorkspace(ctx context.Context, userID, projectID string) error {
	// Check if user is owner
	isOwner, err := p.permissionSvc.IsProjectOwner(ctx, userID, projectID)
	if err != nil {
		return fmt.Errorf("failed to check project ownership: %w", err)
	}
	if !isOwner {
		return fmt.Errorf("only project owner can delete project %s", projectID)
	}

	if err := p.store.DeleteProjectWorkspace(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project workspace: %w", err)
	}

	return nil
}

// AddProjectMember adds a member to a project
func (p *ProjectWorkspaceServiceImpl) AddProjectMember(ctx context.Context, userID, projectID string, req *AddProjectMemberRequest) (*models.ProjectMember, error) {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return nil, fmt.Errorf("admin access required for project %s", projectID)
	}

	// Check if user exists
	_, err = p.store.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user %s not found: %w", req.UserID, err)
	}

	// Create project member
	now := time.Now()
	member := &models.ProjectMember{
		ProjectID:   projectID,
		UserID:      req.UserID,
		Role:        string(req.Role),
		Permissions: req.Permissions,
		InvitedBy:   &userID,
		InvitedAt:   &now,
		JoinedAt:    &now,
		CreatedAt:   now,
	}

	if member.Permissions == nil {
		member.Permissions = make(map[string]interface{})
	}

	if err := p.store.CreateProjectMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add project member: %w", err)
	}

	return member, nil
}

// GetProjectMembers retrieves all members of a project
func (p *ProjectWorkspaceServiceImpl) GetProjectMembers(ctx context.Context, userID, projectID string) ([]models.ProjectMember, error) {
	// Check permissions
	canAccess, err := p.permissionSvc.CanAccessProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project access: %w", err)
	}
	if !canAccess {
		return nil, fmt.Errorf("access denied to project %s", projectID)
	}

	members, err := p.store.GetProjectMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project members: %w", err)
	}

	return members, nil
}

// UpdateProjectMember updates a project member
func (p *ProjectWorkspaceServiceImpl) UpdateProjectMember(ctx context.Context, userID, projectID, memberUserID string, req *UpdateProjectMemberRequest) (*models.ProjectMember, error) {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return nil, fmt.Errorf("admin access required for project %s", projectID)
	}

	// Get current member
	member, err := p.store.GetProjectMember(ctx, projectID, memberUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project member: %w", err)
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Role != nil {
		updates["role"] = string(*req.Role)
	}
	if req.Permissions != nil {
		updates["permissions"] = models.JSONBMap(req.Permissions)
	}

	if len(updates) == 0 {
		// No updates, return current member
		return member, nil
	}

	if err := p.store.UpdateProjectMember(ctx, member.ID, updates); err != nil {
		return nil, fmt.Errorf("failed to update project member: %w", err)
	}

	return p.store.GetProjectMember(ctx, projectID, memberUserID)
}

// RemoveProjectMember removes a member from a project
func (p *ProjectWorkspaceServiceImpl) RemoveProjectMember(ctx context.Context, userID, projectID, memberUserID string) error {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return fmt.Errorf("admin access required for project %s", projectID)
	}

	// Get member to remove
	member, err := p.store.GetProjectMember(ctx, projectID, memberUserID)
	if err != nil {
		return fmt.Errorf("failed to get project member: %w", err)
	}

	// Don't allow removing the owner
	if member.Role == string(models.RoleOwner) {
		return fmt.Errorf("cannot remove project owner")
	}

	if err := p.store.DeleteProjectMember(ctx, member.ID); err != nil {
		return fmt.Errorf("failed to remove project member: %w", err)
	}

	return nil
}

// CreateProjectIntegration creates a new project integration
func (p *ProjectWorkspaceServiceImpl) CreateProjectIntegration(ctx context.Context, userID, projectID string, req *CreateProjectIntegrationRequest) (*models.ProjectIntegration, error) {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return nil, fmt.Errorf("admin access required for project %s", projectID)
	}

	// Encrypt credentials if provided
	encryptedCredentials := req.Credentials
	if len(req.Credentials) > 0 && p.encryptionSvc != nil {
		var err error
		encryptedCredentials, err = p.encryptionSvc.EncryptMap(ctx, req.Credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
		}
	}

	// Create integration
	integration := &models.ProjectIntegration{
		ProjectID:       projectID,
		Platform:        string(req.Platform),
		IntegrationType: string(req.IntegrationType),
		Status:          string(models.IntegrationStatusPending),
		Configuration:   req.Configuration,
		Credentials:     encryptedCredentials,
		SyncCheckpoint:  make(map[string]interface{}),
		CreatedBy:       userID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if integration.Configuration == nil {
		integration.Configuration = make(map[string]interface{})
	}

	if err := p.store.CreateProjectIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create project integration: %w", err)
	}

	return integration, nil
}

// GetProjectIntegrations retrieves all integrations for a project
func (p *ProjectWorkspaceServiceImpl) GetProjectIntegrations(ctx context.Context, userID, projectID string) ([]models.ProjectIntegration, error) {
	// Check permissions
	canAccess, err := p.permissionSvc.CanAccessProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project access: %w", err)
	}
	if !canAccess {
		return nil, fmt.Errorf("access denied to project %s", projectID)
	}

	integrations, err := p.store.GetProjectIntegrations(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project integrations: %w", err)
	}

	// Don't expose credentials in list view
	for i := range integrations {
		integrations[i].Credentials = make(map[string]interface{})
	}

	return integrations, nil
}

// GetProjectIntegration retrieves a specific project integration
func (p *ProjectWorkspaceServiceImpl) GetProjectIntegration(ctx context.Context, userID, projectID, integrationID string) (*models.ProjectIntegration, error) {
	// Check permissions
	canAccess, err := p.permissionSvc.CanAccessProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project access: %w", err)
	}
	if !canAccess {
		return nil, fmt.Errorf("access denied to project %s", projectID)
	}

	integration, err := p.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project integration: %w", err)
	}

	// Verify integration belongs to the project
	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration %s does not belong to project %s", integrationID, projectID)
	}

	// Decrypt credentials if user has admin access
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err == nil && canAdmin && p.encryptionSvc != nil && len(integration.Credentials) > 0 {
		decryptedCredentials, err := p.encryptionSvc.DecryptMap(ctx, integration.Credentials)
		if err == nil {
			integration.Credentials = decryptedCredentials
		}
	} else {
		// Don't expose credentials to non-admin users
		integration.Credentials = make(map[string]interface{})
	}

	return integration, nil
}

// UpdateProjectIntegration updates a project integration
func (p *ProjectWorkspaceServiceImpl) UpdateProjectIntegration(ctx context.Context, userID, projectID, integrationID string, req *UpdateProjectIntegrationRequest) (*models.ProjectIntegration, error) {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return nil, fmt.Errorf("admin access required for project %s", projectID)
	}

	// Get current integration
	integration, err := p.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project integration: %w", err)
	}

	// Verify integration belongs to the project
	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration %s does not belong to project %s", integrationID, projectID)
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Status != nil {
		updates["status"] = string(*req.Status)
	}
	if req.Configuration != nil {
		updates["configuration"] = models.JSONBMap(req.Configuration)
	}
	if req.Credentials != nil {
		// Encrypt credentials
		encryptedCredentials := req.Credentials
		if p.encryptionSvc != nil {
			encryptedCredentials, err = p.encryptionSvc.EncryptMap(ctx, req.Credentials)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
			}
		}
		updates["credentials"] = models.JSONBMap(encryptedCredentials)
	}
	if req.ErrorMessage != nil {
		updates["error_message"] = *req.ErrorMessage
	}

	if len(updates) == 0 {
		// No updates, return current integration
		return p.GetProjectIntegration(ctx, userID, projectID, integrationID)
	}

	if err := p.store.UpdateProjectIntegration(ctx, integrationID, updates); err != nil {
		return nil, fmt.Errorf("failed to update project integration: %w", err)
	}

	return p.GetProjectIntegration(ctx, userID, projectID, integrationID)
}

// DeleteProjectIntegration deletes a project integration
func (p *ProjectWorkspaceServiceImpl) DeleteProjectIntegration(ctx context.Context, userID, projectID, integrationID string) error {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return fmt.Errorf("admin access required for project %s", projectID)
	}

	// Get integration to verify it belongs to the project
	integration, err := p.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get project integration: %w", err)
	}

	if integration.ProjectID != projectID {
		return fmt.Errorf("integration %s does not belong to project %s", integrationID, projectID)
	}

	if err := p.store.DeleteProjectIntegration(ctx, integrationID); err != nil {
		return fmt.Errorf("failed to delete project integration: %w", err)
	}

	return nil
}

// CreateProjectDataSource creates a new project data source
func (p *ProjectWorkspaceServiceImpl) CreateProjectDataSource(ctx context.Context, userID, projectID, integrationID string, req *CreateProjectDataSourceRequest) (*models.ProjectDataSource, error) {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return nil, fmt.Errorf("admin access required for project %s", projectID)
	}

	// Verify integration exists and belongs to project
	integration, err := p.store.GetProjectIntegration(ctx, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project integration: %w", err)
	}
	if integration.ProjectID != projectID {
		return nil, fmt.Errorf("integration %s does not belong to project %s", integrationID, projectID)
	}

	// Create data source
	dataSource := &models.ProjectDataSource{
		ProjectID:     projectID,
		IntegrationID: integrationID,
		SourceType:    string(req.SourceType),
		SourceID:      req.SourceID,
		SourceName:    req.SourceName,
		Configuration: req.Configuration,
		IsActive:      req.IsActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if dataSource.Configuration == nil {
		dataSource.Configuration = make(map[string]interface{})
	}

	if err := p.store.CreateProjectDataSource(ctx, dataSource); err != nil {
		return nil, fmt.Errorf("failed to create project data source: %w", err)
	}

	return dataSource, nil
}

// GetProjectDataSources retrieves all data sources for a project
func (p *ProjectWorkspaceServiceImpl) GetProjectDataSources(ctx context.Context, userID, projectID string) ([]models.ProjectDataSource, error) {
	// Check permissions
	canAccess, err := p.permissionSvc.CanAccessProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project access: %w", err)
	}
	if !canAccess {
		return nil, fmt.Errorf("access denied to project %s", projectID)
	}

	dataSources, err := p.store.GetProjectDataSources(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project data sources: %w", err)
	}

	return dataSources, nil
}

// UpdateProjectDataSource updates a project data source
func (p *ProjectWorkspaceServiceImpl) UpdateProjectDataSource(ctx context.Context, userID, projectID, dataSourceID string, req *UpdateProjectDataSourceRequest) (*models.ProjectDataSource, error) {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return nil, fmt.Errorf("admin access required for project %s", projectID)
	}

	// Get current data source
	dataSource, err := p.store.GetProjectDataSource(ctx, dataSourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project data source: %w", err)
	}

	// Verify data source belongs to the project
	if dataSource.ProjectID != projectID {
		return nil, fmt.Errorf("data source %s does not belong to project %s", dataSourceID, projectID)
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.SourceName != nil {
		updates["source_name"] = *req.SourceName
	}
	if req.Configuration != nil {
		updates["configuration"] = models.JSONBMap(req.Configuration)
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.IngestionStatus != nil {
		updates["ingestion_status"] = *req.IngestionStatus
	}
	if req.ErrorMessage != nil {
		updates["error_message"] = *req.ErrorMessage
	}

	if len(updates) == 0 {
		// No updates, return current data source
		return dataSource, nil
	}

	if err := p.store.UpdateProjectDataSource(ctx, dataSourceID, updates); err != nil {
		return nil, fmt.Errorf("failed to update project data source: %w", err)
	}

	return p.store.GetProjectDataSource(ctx, dataSourceID)
}

// DeleteProjectDataSource deletes a project data source
func (p *ProjectWorkspaceServiceImpl) DeleteProjectDataSource(ctx context.Context, userID, projectID, dataSourceID string) error {
	// Check permissions
	canAdmin, err := p.permissionSvc.CanAdminProject(ctx, userID, projectID)
	if err != nil {
		return fmt.Errorf("failed to check project admin access: %w", err)
	}
	if !canAdmin {
		return fmt.Errorf("admin access required for project %s", projectID)
	}

	// Get data source to verify it belongs to the project
	dataSource, err := p.store.GetProjectDataSource(ctx, dataSourceID)
	if err != nil {
		return fmt.Errorf("failed to get project data source: %w", err)
	}

	if dataSource.ProjectID != projectID {
		return fmt.Errorf("data source %s does not belong to project %s", dataSourceID, projectID)
	}

	if err := p.store.DeleteProjectDataSource(ctx, dataSourceID); err != nil {
		return fmt.Errorf("failed to delete project data source: %w", err)
	}

	return nil
}