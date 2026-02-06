package services

import (
	"context"
	"fmt"

	"github.com/DevAnuragT/context_keeper/internal/models"
)

// PermissionServiceImpl implements the PermissionService interface
type PermissionServiceImpl struct {
	store RepositoryStore
}

// NewPermissionService creates a new permission service
func NewPermissionService(store RepositoryStore) PermissionService {
	return &PermissionServiceImpl{
		store: store,
	}
}

// CanAccessProject checks if a user can access a project (any level of access)
func (p *PermissionServiceImpl) CanAccessProject(ctx context.Context, userID, projectID string) (bool, error) {
	// Check if user is project owner
	workspace, err := p.store.GetProjectWorkspace(ctx, projectID)
	if err != nil {
		return false, fmt.Errorf("failed to get project workspace: %w", err)
	}

	if workspace.OwnerID == userID {
		return true, nil
	}

	// Check if user is a project member
	member, err := p.store.GetProjectMember(ctx, projectID, userID)
	if err != nil {
		// User is not a member
		return false, nil
	}

	// Any project member has access
	return member != nil, nil
}

// CanReadProject checks if a user can read from a project
func (p *PermissionServiceImpl) CanReadProject(ctx context.Context, userID, projectID string) (bool, error) {
	// For now, any project access grants read access
	return p.CanAccessProject(ctx, userID, projectID)
}

// CanWriteProject checks if a user can write to a project
func (p *PermissionServiceImpl) CanWriteProject(ctx context.Context, userID, projectID string) (bool, error) {
	role, err := p.GetUserProjectRole(ctx, userID, projectID)
	if err != nil {
		return false, err
	}

	// Owner, admin, and member roles can write
	switch role {
	case models.RoleOwner, models.RoleAdmin, models.RoleMember:
		return true, nil
	case models.RoleViewer:
		return false, nil
	default:
		return false, nil
	}
}

// CanAdminProject checks if a user can administer a project
func (p *PermissionServiceImpl) CanAdminProject(ctx context.Context, userID, projectID string) (bool, error) {
	role, err := p.GetUserProjectRole(ctx, userID, projectID)
	if err != nil {
		return false, err
	}

	// Only owner and admin roles can administer
	switch role {
	case models.RoleOwner, models.RoleAdmin:
		return true, nil
	default:
		return false, nil
	}
}

// CanAccessRepository checks if a user can access a repository
func (p *PermissionServiceImpl) CanAccessRepository(ctx context.Context, userID string, repoID int64) (bool, error) {
	// Get repository
	repo, err := p.store.GetRepoByID(ctx, repoID)
	if err != nil {
		return false, fmt.Errorf("failed to get repository: %w", err)
	}

	// If repository has no project_id, use legacy owner-based access
	if repo.ProjectID == nil {
		return repo.Owner == userID, nil
	}

	// Check project access
	return p.CanAccessProject(ctx, userID, *repo.ProjectID)
}

// CanAccessKnowledgeEntity checks if a user can access a knowledge entity
func (p *PermissionServiceImpl) CanAccessKnowledgeEntity(ctx context.Context, userID, entityID string) (bool, error) {
	// Get knowledge entity
	entity, err := p.store.GetKnowledgeEntity(ctx, entityID)
	if err != nil {
		return false, fmt.Errorf("failed to get knowledge entity: %w", err)
	}

	// If entity has no project_id, allow access (legacy data)
	if entity.Metadata == nil {
		return true, nil
	}

	projectID, ok := entity.Metadata["project_id"].(string)
	if !ok || projectID == "" {
		return true, nil
	}

	// Check project access
	return p.CanAccessProject(ctx, userID, projectID)
}

// GetUserProjectRole gets a user's role in a project
func (p *PermissionServiceImpl) GetUserProjectRole(ctx context.Context, userID, projectID string) (models.UserRole, error) {
	// Check if user is project owner
	workspace, err := p.store.GetProjectWorkspace(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("failed to get project workspace: %w", err)
	}

	if workspace.OwnerID == userID {
		return models.RoleOwner, nil
	}

	// Check project membership
	member, err := p.store.GetProjectMember(ctx, projectID, userID)
	if err != nil {
		return "", fmt.Errorf("user is not a project member")
	}

	return models.UserRole(member.Role), nil
}

// IsProjectMember checks if a user is a member of a project
func (p *PermissionServiceImpl) IsProjectMember(ctx context.Context, userID, projectID string) (bool, error) {
	_, err := p.GetUserProjectRole(ctx, userID, projectID)
	return err == nil, nil
}

// IsProjectOwner checks if a user is the owner of a project
func (p *PermissionServiceImpl) IsProjectOwner(ctx context.Context, userID, projectID string) (bool, error) {
	role, err := p.GetUserProjectRole(ctx, userID, projectID)
	if err != nil {
		return false, err
	}

	return role == models.RoleOwner, nil
}

// MockPermissionService is a mock implementation for testing
type MockPermissionService struct {
	CanAccessProjectFunc        func(ctx context.Context, userID, projectID string) (bool, error)
	CanReadProjectFunc          func(ctx context.Context, userID, projectID string) (bool, error)
	CanWriteProjectFunc         func(ctx context.Context, userID, projectID string) (bool, error)
	CanAdminProjectFunc         func(ctx context.Context, userID, projectID string) (bool, error)
	CanAccessRepositoryFunc     func(ctx context.Context, userID string, repoID int64) (bool, error)
	CanAccessKnowledgeEntityFunc func(ctx context.Context, userID, entityID string) (bool, error)
	GetUserProjectRoleFunc      func(ctx context.Context, userID, projectID string) (models.UserRole, error)
	IsProjectMemberFunc         func(ctx context.Context, userID, projectID string) (bool, error)
	IsProjectOwnerFunc          func(ctx context.Context, userID, projectID string) (bool, error)
}

// NewMockPermissionService creates a new mock permission service
func NewMockPermissionService() *MockPermissionService {
	return &MockPermissionService{
		CanAccessProjectFunc: func(ctx context.Context, userID, projectID string) (bool, error) {
			return true, nil
		},
		CanReadProjectFunc: func(ctx context.Context, userID, projectID string) (bool, error) {
			return true, nil
		},
		CanWriteProjectFunc: func(ctx context.Context, userID, projectID string) (bool, error) {
			return true, nil
		},
		CanAdminProjectFunc: func(ctx context.Context, userID, projectID string) (bool, error) {
			return true, nil
		},
		CanAccessRepositoryFunc: func(ctx context.Context, userID string, repoID int64) (bool, error) {
			return true, nil
		},
		CanAccessKnowledgeEntityFunc: func(ctx context.Context, userID, entityID string) (bool, error) {
			return true, nil
		},
		GetUserProjectRoleFunc: func(ctx context.Context, userID, projectID string) (models.UserRole, error) {
			return models.RoleOwner, nil
		},
		IsProjectMemberFunc: func(ctx context.Context, userID, projectID string) (bool, error) {
			return true, nil
		},
		IsProjectOwnerFunc: func(ctx context.Context, userID, projectID string) (bool, error) {
			return true, nil
		},
	}
}

func (m *MockPermissionService) CanAccessProject(ctx context.Context, userID, projectID string) (bool, error) {
	return m.CanAccessProjectFunc(ctx, userID, projectID)
}

func (m *MockPermissionService) CanReadProject(ctx context.Context, userID, projectID string) (bool, error) {
	return m.CanReadProjectFunc(ctx, userID, projectID)
}

func (m *MockPermissionService) CanWriteProject(ctx context.Context, userID, projectID string) (bool, error) {
	return m.CanWriteProjectFunc(ctx, userID, projectID)
}

func (m *MockPermissionService) CanAdminProject(ctx context.Context, userID, projectID string) (bool, error) {
	return m.CanAdminProjectFunc(ctx, userID, projectID)
}

func (m *MockPermissionService) CanAccessRepository(ctx context.Context, userID string, repoID int64) (bool, error) {
	return m.CanAccessRepositoryFunc(ctx, userID, repoID)
}

func (m *MockPermissionService) CanAccessKnowledgeEntity(ctx context.Context, userID, entityID string) (bool, error) {
	return m.CanAccessKnowledgeEntityFunc(ctx, userID, entityID)
}

func (m *MockPermissionService) GetUserProjectRole(ctx context.Context, userID, projectID string) (models.UserRole, error) {
	return m.GetUserProjectRoleFunc(ctx, userID, projectID)
}

func (m *MockPermissionService) IsProjectMember(ctx context.Context, userID, projectID string) (bool, error) {
	return m.IsProjectMemberFunc(ctx, userID, projectID)
}

func (m *MockPermissionService) IsProjectOwner(ctx context.Context, userID, projectID string) (bool, error) {
	return m.IsProjectOwnerFunc(ctx, userID, projectID)
}