package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/models"
	"github.com/DevAnuragT/context_keeper/internal/services"
)

// Repository implements the RepositoryStore interface
type Repository struct {
	db *sql.DB
}

// New creates a new repository instance
func New(db *sql.DB) services.RepositoryStore {
	return &Repository{db: db}
}

// Repository operations
func (r *Repository) CreateRepo(ctx context.Context, repo *models.Repository) error {
	query := `
		INSERT INTO repos (name, full_name, owner, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (full_name) DO UPDATE SET
			name = EXCLUDED.name,
			owner = EXCLUDED.owner,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	now := time.Now()
	repo.CreatedAt = now
	repo.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query, repo.Name, repo.FullName, repo.Owner, repo.CreatedAt, repo.UpdatedAt).Scan(&repo.ID)
	return err
}

func (r *Repository) GetReposByUser(ctx context.Context, userID string) ([]models.Repository, error) {
	query := `
		SELECT id, name, full_name, owner, project_id, created_at, updated_at
		FROM repos
		WHERE owner = $1
		ORDER BY updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []models.Repository
	for rows.Next() {
		var repo models.Repository
		err := rows.Scan(&repo.ID, &repo.Name, &repo.FullName, &repo.Owner, &repo.ProjectID, &repo.CreatedAt, &repo.UpdatedAt)
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}

	return repos, rows.Err()
}

// GetReposByProject retrieves repositories for a specific project
func (r *Repository) GetReposByProject(ctx context.Context, projectID string) ([]models.Repository, error) {
	query := `
		SELECT id, name, full_name, owner, project_id, created_at, updated_at
		FROM repos
		WHERE project_id = $1
		ORDER BY updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []models.Repository
	for rows.Next() {
		var repo models.Repository
		err := rows.Scan(&repo.ID, &repo.Name, &repo.FullName, &repo.Owner, &repo.ProjectID, &repo.CreatedAt, &repo.UpdatedAt)
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}

	return repos, rows.Err()
}

func (r *Repository) GetRepoByID(ctx context.Context, repoID int64) (*models.Repository, error) {
	query := `
		SELECT id, name, full_name, owner, project_id, created_at, updated_at
		FROM repos
		WHERE id = $1`

	var repo models.Repository
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(
		&repo.ID, &repo.Name, &repo.FullName, &repo.Owner, &repo.ProjectID, &repo.CreatedAt, &repo.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

// GetRepoByIDAndProject retrieves a repository by ID with project validation
func (r *Repository) GetRepoByIDAndProject(ctx context.Context, repoID int64, projectID string) (*models.Repository, error) {
	query := `
		SELECT id, name, full_name, owner, project_id, created_at, updated_at
		FROM repos
		WHERE id = $1 AND project_id = $2`

	var repo models.Repository
	err := r.db.QueryRowContext(ctx, query, repoID, projectID).Scan(
		&repo.ID, &repo.Name, &repo.FullName, &repo.Owner, &repo.ProjectID, &repo.CreatedAt, &repo.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

// Pull request operations
func (r *Repository) CreatePullRequest(ctx context.Context, pr *models.PullRequest) error {
	query := `
		INSERT INTO pull_requests (id, repo_id, number, title, body, author, state, created_at, merged_at, files_changed, labels)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (repo_id, number) DO UPDATE SET
			title = EXCLUDED.title,
			body = EXCLUDED.body,
			state = EXCLUDED.state,
			merged_at = EXCLUDED.merged_at,
			files_changed = EXCLUDED.files_changed,
			labels = EXCLUDED.labels`

	_, err := r.db.ExecContext(ctx, query,
		pr.ID, pr.RepoID, pr.Number, pr.Title, pr.Body, pr.Author, pr.State,
		pr.CreatedAt, pr.MergedAt, pr.FilesChanged, pr.Labels)
	return err
}

func (r *Repository) GetRecentPRs(ctx context.Context, repoID int64, limit int) ([]models.PullRequest, error) {
	query := `
		SELECT id, repo_id, number, title, body, author, state, created_at, merged_at, files_changed, labels
		FROM pull_requests
		WHERE repo_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequest
	for rows.Next() {
		var pr models.PullRequest
		err := rows.Scan(&pr.ID, &pr.RepoID, &pr.Number, &pr.Title, &pr.Body, &pr.Author,
			&pr.State, &pr.CreatedAt, &pr.MergedAt, &pr.FilesChanged, &pr.Labels)
		if err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

// GetRecentPRsByProject retrieves recent pull requests for all repositories in a project
func (r *Repository) GetRecentPRsByProject(ctx context.Context, projectID string, limit int) ([]models.PullRequest, error) {
	query := `
		SELECT pr.id, pr.repo_id, pr.number, pr.title, pr.body, pr.author, pr.state, pr.created_at, pr.merged_at, pr.files_changed, pr.labels
		FROM pull_requests pr
		JOIN repos r ON pr.repo_id = r.id
		WHERE r.project_id = $1
		ORDER BY pr.created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequest
	for rows.Next() {
		var pr models.PullRequest
		err := rows.Scan(&pr.ID, &pr.RepoID, &pr.Number, &pr.Title, &pr.Body, &pr.Author,
			&pr.State, &pr.CreatedAt, &pr.MergedAt, &pr.FilesChanged, &pr.Labels)
		if err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

// Issue operations
func (r *Repository) CreateIssue(ctx context.Context, issue *models.Issue) error {
	query := `
		INSERT INTO issues (id, repo_id, title, body, author, state, created_at, closed_at, labels)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			body = EXCLUDED.body,
			state = EXCLUDED.state,
			closed_at = EXCLUDED.closed_at,
			labels = EXCLUDED.labels`

	_, err := r.db.ExecContext(ctx, query,
		issue.ID, issue.RepoID, issue.Title, issue.Body, issue.Author,
		issue.State, issue.CreatedAt, issue.ClosedAt, issue.Labels)
	return err
}

func (r *Repository) GetRecentIssues(ctx context.Context, repoID int64, limit int) ([]models.Issue, error) {
	query := `
		SELECT id, repo_id, title, body, author, state, created_at, closed_at, labels
		FROM issues
		WHERE repo_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []models.Issue
	for rows.Next() {
		var issue models.Issue
		err := rows.Scan(&issue.ID, &issue.RepoID, &issue.Title, &issue.Body, &issue.Author,
			&issue.State, &issue.CreatedAt, &issue.ClosedAt, &issue.Labels)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}

	return issues, rows.Err()
}

// GetRecentIssuesByProject retrieves recent issues for all repositories in a project
func (r *Repository) GetRecentIssuesByProject(ctx context.Context, projectID string, limit int) ([]models.Issue, error) {
	query := `
		SELECT i.id, i.repo_id, i.title, i.body, i.author, i.state, i.created_at, i.closed_at, i.labels
		FROM issues i
		JOIN repos r ON i.repo_id = r.id
		WHERE r.project_id = $1
		ORDER BY i.created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []models.Issue
	for rows.Next() {
		var issue models.Issue
		err := rows.Scan(&issue.ID, &issue.RepoID, &issue.Title, &issue.Body, &issue.Author,
			&issue.State, &issue.CreatedAt, &issue.ClosedAt, &issue.Labels)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}

	return issues, rows.Err()
}

// Commit operations
func (r *Repository) CreateCommit(ctx context.Context, commit *models.Commit) error {
	query := `
		INSERT INTO commits (sha, repo_id, message, author, created_at, files_changed)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (sha) DO UPDATE SET
			message = EXCLUDED.message,
			author = EXCLUDED.author,
			created_at = EXCLUDED.created_at,
			files_changed = EXCLUDED.files_changed`

	_, err := r.db.ExecContext(ctx, query,
		commit.SHA, commit.RepoID, commit.Message, commit.Author, commit.CreatedAt, commit.FilesChanged)
	return err
}

func (r *Repository) GetRecentCommits(ctx context.Context, repoID int64, limit int) ([]models.Commit, error) {
	query := `
		SELECT sha, repo_id, message, author, created_at, files_changed
		FROM commits
		WHERE repo_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []models.Commit
	for rows.Next() {
		var commit models.Commit
		err := rows.Scan(&commit.SHA, &commit.RepoID, &commit.Message, &commit.Author,
			&commit.CreatedAt, &commit.FilesChanged)
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}

	return commits, rows.Err()
}

// GetRecentCommitsByProject retrieves recent commits for all repositories in a project
func (r *Repository) GetRecentCommitsByProject(ctx context.Context, projectID string, limit int) ([]models.Commit, error) {
	query := `
		SELECT c.sha, c.repo_id, c.message, c.author, c.created_at, c.files_changed
		FROM commits c
		JOIN repos r ON c.repo_id = r.id
		WHERE r.project_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []models.Commit
	for rows.Next() {
		var commit models.Commit
		err := rows.Scan(&commit.SHA, &commit.RepoID, &commit.Message, &commit.Author,
			&commit.CreatedAt, &commit.FilesChanged)
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}

	return commits, rows.Err()
}

// Job operations
func (r *Repository) CreateJob(ctx context.Context, job *models.IngestionJob) error {
	query := `
		INSERT INTO ingestion_jobs (repo_id, status, started_at, finished_at, error_message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	err := r.db.QueryRowContext(ctx, query, job.RepoID, job.Status, job.StartedAt, job.FinishedAt, job.ErrorMsg).Scan(&job.ID)
	return err
}

func (r *Repository) UpdateJobStatus(ctx context.Context, jobID int64, status models.JobStatus, errorMsg *string) error {
	var query string
	var args []interface{}

	if status == models.JobStatusRunning {
		query = `UPDATE ingestion_jobs SET status = $1, started_at = $2 WHERE id = $3`
		args = []interface{}{status, time.Now(), jobID}
	} else {
		query = `UPDATE ingestion_jobs SET status = $1, finished_at = $2, error_message = $3 WHERE id = $4`
		args = []interface{}{status, time.Now(), errorMsg, jobID}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) GetJobByID(ctx context.Context, jobID int64) (*models.IngestionJob, error) {
	query := `
		SELECT id, repo_id, status, started_at, finished_at, error_message
		FROM ingestion_jobs
		WHERE id = $1`

	var job models.IngestionJob
	err := r.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.RepoID, &job.Status, &job.StartedAt, &job.FinishedAt, &job.ErrorMsg)
	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *Repository) GetJobsByRepo(ctx context.Context, repoID int64) ([]models.IngestionJob, error) {
	query := `
		SELECT id, repo_id, status, started_at, finished_at, error_message
		FROM ingestion_jobs
		WHERE repo_id = $1
		ORDER BY id DESC`

	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.IngestionJob
	for rows.Next() {
		var job models.IngestionJob
		err := rows.Scan(&job.ID, &job.RepoID, &job.Status, &job.StartedAt, &job.FinishedAt, &job.ErrorMsg)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// Knowledge Graph operations

func (r *Repository) CreateKnowledgeEntity(ctx context.Context, entity *models.KnowledgeEntity) error {
	query := `
		INSERT INTO knowledge_entities (id, entity_type, entity_id, title, content, metadata, platform_source, source_event_ids, participants, embedding, project_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (entity_type, entity_id) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			platform_source = EXCLUDED.platform_source,
			source_event_ids = EXCLUDED.source_event_ids,
			participants = EXCLUDED.participants,
			embedding = EXCLUDED.embedding,
			project_id = EXCLUDED.project_id,
			updated_at = EXCLUDED.updated_at`

	now := time.Now()
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = now
	}
	entity.UpdatedAt = now

	// Generate UUID if not provided
	if entity.ID == "" {
		entity.ID = generateUUID()
	}

	// Extract project_id from metadata if present
	var projectID *string
	if entity.Metadata != nil {
		if pid, ok := entity.Metadata["project_id"].(string); ok && pid != "" {
			projectID = &pid
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		entity.ID, entity.EntityType, entity.EntityID, entity.Title, entity.Content,
		models.JSONBMap(entity.Metadata), entity.PlatformSource, entity.SourceEventIDs,
		entity.Participants, convertFloatSliceToString(entity.Embedding), projectID, entity.CreatedAt, entity.UpdatedAt)
	return err
}

func (r *Repository) GetKnowledgeEntity(ctx context.Context, id string) (*models.KnowledgeEntity, error) {
	query := `
		SELECT id, entity_type, entity_id, title, content, metadata, platform_source, source_event_ids, participants, embedding, project_id, created_at, updated_at
		FROM knowledge_entities
		WHERE id = $1`

	var entity models.KnowledgeEntity
	var metadata models.JSONBMap
	var embeddingStr *string
	var projectID *string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title, &entity.Content,
		&metadata, &entity.PlatformSource, &entity.SourceEventIDs, &entity.Participants,
		&embeddingStr, &projectID, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		return nil, err
	}

	entity.Metadata = map[string]interface{}(metadata)
	if embeddingStr != nil {
		entity.Embedding = convertStringToFloatSlice(*embeddingStr)
	}

	// Add project_id to metadata if present
	if projectID != nil && *projectID != "" {
		if entity.Metadata == nil {
			entity.Metadata = make(map[string]interface{})
		}
		entity.Metadata["project_id"] = *projectID
	}

	return &entity, nil
}

func (r *Repository) UpdateKnowledgeEntity(ctx context.Context, entity *models.KnowledgeEntity) error {
	query := `
		UPDATE knowledge_entities 
		SET title = $2, content = $3, metadata = $4, platform_source = $5, source_event_ids = $6, participants = $7, embedding = $8, project_id = $9, updated_at = $10
		WHERE id = $1`

	entity.UpdatedAt = time.Now()

	// Extract project_id from metadata if present
	var projectID *string
	if entity.Metadata != nil {
		if pid, ok := entity.Metadata["project_id"].(string); ok && pid != "" {
			projectID = &pid
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		entity.ID, entity.Title, entity.Content, models.JSONBMap(entity.Metadata),
		entity.PlatformSource, entity.SourceEventIDs, entity.Participants,
		convertFloatSliceToString(entity.Embedding), projectID, entity.UpdatedAt)
	return err
}

func (r *Repository) DeleteKnowledgeEntity(ctx context.Context, id string) error {
	query := `DELETE FROM knowledge_entities WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) CreateKnowledgeRelationship(ctx context.Context, relationship *models.KnowledgeRelationship) error {
	query := `
		INSERT INTO knowledge_relationships (id, source_entity_id, target_entity_id, relationship_type, strength, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (source_entity_id, target_entity_id, relationship_type) DO UPDATE SET
			strength = EXCLUDED.strength,
			metadata = EXCLUDED.metadata`

	if relationship.CreatedAt.IsZero() {
		relationship.CreatedAt = time.Now()
	}

	// Generate UUID if not provided
	if relationship.ID == "" {
		relationship.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		relationship.ID, relationship.SourceEntityID, relationship.TargetEntityID,
		relationship.RelationshipType, relationship.Strength, models.JSONBMap(relationship.Metadata),
		relationship.CreatedAt)
	return err
}

func (r *Repository) GetKnowledgeRelationships(ctx context.Context, entityID string) ([]models.KnowledgeRelationship, error) {
	query := `
		SELECT id, source_entity_id, target_entity_id, relationship_type, strength, metadata, created_at
		FROM knowledge_relationships
		WHERE source_entity_id = $1 OR target_entity_id = $1
		ORDER BY strength DESC, created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []models.KnowledgeRelationship
	for rows.Next() {
		var rel models.KnowledgeRelationship
		var metadata models.JSONBMap

		err := rows.Scan(&rel.ID, &rel.SourceEntityID, &rel.TargetEntityID,
			&rel.RelationshipType, &rel.Strength, &metadata, &rel.CreatedAt)
		if err != nil {
			return nil, err
		}

		rel.Metadata = map[string]interface{}(metadata)
		relationships = append(relationships, rel)
	}

	return relationships, rows.Err()
}

func (r *Repository) DeleteKnowledgeRelationship(ctx context.Context, id string) error {
	query := `DELETE FROM knowledge_relationships WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) CreateDecisionRecord(ctx context.Context, decision *models.DecisionRecord) error {
	query := `
		INSERT INTO decision_records (id, entity_id, decision_id, title, decision, rationale, alternatives, consequences, status, platform_source, source_event_ids, participants, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (decision_id) DO UPDATE SET
			title = EXCLUDED.title,
			decision = EXCLUDED.decision,
			rationale = EXCLUDED.rationale,
			alternatives = EXCLUDED.alternatives,
			consequences = EXCLUDED.consequences,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at`

	now := time.Now()
	if decision.CreatedAt.IsZero() {
		decision.CreatedAt = now
	}
	decision.UpdatedAt = now

	// Generate UUID if not provided
	if decision.ID == "" {
		decision.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		decision.ID, decision.EntityID, decision.DecisionID, decision.Title, decision.Decision,
		decision.Rationale, decision.Alternatives, decision.Consequences, decision.Status,
		decision.PlatformSource, decision.SourceEventIDs, decision.Participants,
		decision.CreatedAt, decision.UpdatedAt)
	return err
}

func (r *Repository) GetDecisionRecord(ctx context.Context, decisionID string) (*models.DecisionRecord, error) {
	query := `
		SELECT id, entity_id, decision_id, title, decision, rationale, alternatives, consequences, status, platform_source, source_event_ids, participants, created_at, updated_at
		FROM decision_records
		WHERE decision_id = $1`

	var decision models.DecisionRecord
	err := r.db.QueryRowContext(ctx, query, decisionID).Scan(
		&decision.ID, &decision.EntityID, &decision.DecisionID, &decision.Title, &decision.Decision,
		&decision.Rationale, &decision.Alternatives, &decision.Consequences, &decision.Status,
		&decision.PlatformSource, &decision.SourceEventIDs, &decision.Participants,
		&decision.CreatedAt, &decision.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &decision, nil
}

func (r *Repository) UpdateDecisionRecord(ctx context.Context, decision *models.DecisionRecord) error {
	query := `
		UPDATE decision_records 
		SET title = $2, decision = $3, rationale = $4, alternatives = $5, consequences = $6, status = $7, updated_at = $8
		WHERE decision_id = $1`

	decision.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		decision.DecisionID, decision.Title, decision.Decision, decision.Rationale,
		decision.Alternatives, decision.Consequences, decision.Status, decision.UpdatedAt)
	return err
}

func (r *Repository) CreateDiscussionSummary(ctx context.Context, summary *models.DiscussionSummary) error {
	query := `
		INSERT INTO discussion_summaries (id, entity_id, summary_id, thread_id, platform, participants, summary, key_points, action_items, file_references, feature_references, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (summary_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			key_points = EXCLUDED.key_points,
			action_items = EXCLUDED.action_items,
			file_references = EXCLUDED.file_references,
			feature_references = EXCLUDED.feature_references`

	if summary.CreatedAt.IsZero() {
		summary.CreatedAt = time.Now()
	}

	// Generate UUID if not provided
	if summary.ID == "" {
		summary.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		summary.ID, summary.EntityID, summary.SummaryID, summary.ThreadID, summary.Platform,
		summary.Participants, summary.Summary, summary.KeyPoints, summary.ActionItems,
		summary.FileReferences, summary.FeatureReferences, summary.CreatedAt)
	return err
}

func (r *Repository) GetDiscussionSummary(ctx context.Context, summaryID string) (*models.DiscussionSummary, error) {
	query := `
		SELECT id, entity_id, summary_id, thread_id, platform, participants, summary, key_points, action_items, file_references, feature_references, created_at
		FROM discussion_summaries
		WHERE summary_id = $1`

	var summary models.DiscussionSummary
	err := r.db.QueryRowContext(ctx, query, summaryID).Scan(
		&summary.ID, &summary.EntityID, &summary.SummaryID, &summary.ThreadID, &summary.Platform,
		&summary.Participants, &summary.Summary, &summary.KeyPoints, &summary.ActionItems,
		&summary.FileReferences, &summary.FeatureReferences, &summary.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (r *Repository) CreateFeatureContext(ctx context.Context, feature *models.FeatureContext) error {
	query := `
		INSERT INTO feature_contexts (id, entity_id, feature_id, feature_name, description, status, contributors, related_files, discussions, decisions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (feature_id) DO UPDATE SET
			feature_name = EXCLUDED.feature_name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			contributors = EXCLUDED.contributors,
			related_files = EXCLUDED.related_files,
			discussions = EXCLUDED.discussions,
			decisions = EXCLUDED.decisions,
			updated_at = EXCLUDED.updated_at`

	now := time.Now()
	if feature.CreatedAt.IsZero() {
		feature.CreatedAt = now
	}
	feature.UpdatedAt = now

	// Generate UUID if not provided
	if feature.ID == "" {
		feature.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		feature.ID, feature.EntityID, feature.FeatureID, feature.FeatureName, feature.Description,
		feature.Status, feature.Contributors, feature.RelatedFiles, feature.Discussions,
		feature.Decisions, feature.CreatedAt, feature.UpdatedAt)
	return err
}

func (r *Repository) GetFeatureContext(ctx context.Context, featureID string) (*models.FeatureContext, error) {
	query := `
		SELECT id, entity_id, feature_id, feature_name, description, status, contributors, related_files, discussions, decisions, created_at, updated_at
		FROM feature_contexts
		WHERE feature_id = $1`

	var feature models.FeatureContext
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(
		&feature.ID, &feature.EntityID, &feature.FeatureID, &feature.FeatureName, &feature.Description,
		&feature.Status, &feature.Contributors, &feature.RelatedFiles, &feature.Discussions,
		&feature.Decisions, &feature.CreatedAt, &feature.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &feature, nil
}

func (r *Repository) UpdateFeatureContext(ctx context.Context, feature *models.FeatureContext) error {
	query := `
		UPDATE feature_contexts 
		SET feature_name = $2, description = $3, status = $4, contributors = $5, related_files = $6, discussions = $7, decisions = $8, updated_at = $9
		WHERE feature_id = $1`

	feature.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		feature.FeatureID, feature.FeatureName, feature.Description, feature.Status,
		feature.Contributors, feature.RelatedFiles, feature.Discussions, feature.Decisions,
		feature.UpdatedAt)
	return err
}

func (r *Repository) CreateFileContextHistory(ctx context.Context, fileContext *models.FileContextHistory) error {
	query := `
		INSERT INTO file_context_history (id, entity_id, context_id, file_path, change_reason, discussion_context, related_decisions, contributors, platform_sources, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (context_id) DO UPDATE SET
			change_reason = EXCLUDED.change_reason,
			discussion_context = EXCLUDED.discussion_context,
			related_decisions = EXCLUDED.related_decisions,
			contributors = EXCLUDED.contributors,
			platform_sources = EXCLUDED.platform_sources`

	if fileContext.CreatedAt.IsZero() {
		fileContext.CreatedAt = time.Now()
	}

	// Generate UUID if not provided
	if fileContext.ID == "" {
		fileContext.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		fileContext.ID, fileContext.EntityID, fileContext.ContextID, fileContext.FilePath,
		fileContext.ChangeReason, fileContext.DiscussionContext, fileContext.RelatedDecisions,
		fileContext.Contributors, models.JSONBMap(fileContext.PlatformSources), fileContext.CreatedAt)
	return err
}

func (r *Repository) GetFileContextHistory(ctx context.Context, filePath string) ([]models.FileContextHistory, error) {
	query := `
		SELECT id, entity_id, context_id, file_path, change_reason, discussion_context, related_decisions, contributors, platform_sources, created_at
		FROM file_context_history
		WHERE file_path = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contexts []models.FileContextHistory
	for rows.Next() {
		var context models.FileContextHistory
		var platformSources models.JSONBMap

		err := rows.Scan(&context.ID, &context.EntityID, &context.ContextID, &context.FilePath,
			&context.ChangeReason, &context.DiscussionContext, &context.RelatedDecisions,
			&context.Contributors, &platformSources, &context.CreatedAt)
		if err != nil {
			return nil, err
		}

		context.PlatformSources = map[string]interface{}(platformSources)
		contexts = append(contexts, context)
	}

	return contexts, rows.Err()
}

func (r *Repository) SearchKnowledgeEntities(ctx context.Context, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error) {
	// This is a simplified implementation - in production you'd want more sophisticated search
	sqlQuery := `
		SELECT id, entity_type, entity_id, title, content, metadata, platform_source, source_event_ids, participants, embedding, project_id, created_at, updated_at,
		       ts_rank(to_tsvector('english', title || ' ' || content), plainto_tsquery('english', $1)) as rank
		FROM knowledge_entities
		WHERE to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', $1)`

	args := []interface{}{query.Query}
	argIndex := 2

	// Add entity type filter
	if len(query.EntityTypes) > 0 {
		sqlQuery += fmt.Sprintf(" AND entity_type = ANY($%d)", argIndex)
		args = append(args, query.EntityTypes)
		argIndex++
	}

	// Add platform filter
	if len(query.Platforms) > 0 {
		sqlQuery += fmt.Sprintf(" AND platform_source = ANY($%d)", argIndex)
		args = append(args, query.Platforms)
		argIndex++
	}

	// Add date range filter
	if query.DateRange != nil {
		if query.DateRange.Start != nil {
			sqlQuery += fmt.Sprintf(" AND created_at >= $%d", argIndex)
			args = append(args, *query.DateRange.Start)
			argIndex++
		}
		if query.DateRange.End != nil {
			sqlQuery += fmt.Sprintf(" AND created_at <= $%d", argIndex)
			args = append(args, *query.DateRange.End)
			argIndex++
		}
	}

	sqlQuery += " ORDER BY rank DESC"

	// Add limit
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
	} else {
		sqlQuery += " LIMIT 50" // Default limit
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchResult
	rank := 1
	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var projectID *string
		var similarity float64

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &projectID, &entity.CreatedAt, &entity.UpdatedAt, &similarity)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if projectID != nil && *projectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *projectID
		}

		// Don't include content if not requested
		if !query.IncludeContent {
			entity.Content = ""
		}

		results = append(results, models.SearchResult{
			Entity:     entity,
			Similarity: similarity,
			Rank:       rank,
		})
		rank++
	}

	return results, rows.Err()
}

func (r *Repository) SearchSimilarEntities(ctx context.Context, embedding []float32, entityTypes []string, limit int) ([]models.SearchResult, error) {
	// Vector similarity search using cosine distance
	sqlQuery := `
		SELECT id, entity_type, entity_id, title, content, metadata, platform_source, source_event_ids, participants, embedding, project_id, created_at, updated_at,
		       1 - (embedding <=> $1) as similarity
		FROM knowledge_entities
		WHERE embedding IS NOT NULL`

	args := []interface{}{convertFloatSliceToString(embedding)}
	argIndex := 2

	// Add entity type filter
	if len(entityTypes) > 0 {
		sqlQuery += fmt.Sprintf(" AND entity_type = ANY($%d)", argIndex)
		args = append(args, entityTypes)
		argIndex++
	}

	sqlQuery += " ORDER BY embedding <=> $1"

	// Add limit
	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	} else {
		sqlQuery += " LIMIT 10" // Default limit
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchResult
	rank := 1
	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var projectID *string
		var similarity float64

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &projectID, &entity.CreatedAt, &entity.UpdatedAt, &similarity)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if projectID != nil && *projectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *projectID
		}

		results = append(results, models.SearchResult{
			Entity:     entity,
			Similarity: similarity,
			Rank:       rank,
		})
		rank++
	}

	return results, rows.Err()
}

func (r *Repository) TraverseKnowledgeGraph(ctx context.Context, startEntityID string, maxDepth int, relationshipTypes []string) (*models.GraphTraversalResult, error) {
	// Simplified graph traversal - in production you'd want more sophisticated algorithms
	query := `
		WITH RECURSIVE graph_traversal AS (
			-- Base case: start with the initial entity
			SELECT e.id, e.entity_type, e.entity_id, e.title, e.content, e.metadata, e.platform_source, e.source_event_ids, e.participants, e.embedding, e.project_id, e.created_at, e.updated_at,
			       0 as depth, ARRAY[e.id] as path, 0.0 as total_strength
			FROM knowledge_entities e
			WHERE e.id = $1
			
			UNION ALL
			
			-- Recursive case: follow relationships
			SELECT e.id, e.entity_type, e.entity_id, e.title, e.content, e.metadata, e.platform_source, e.source_event_ids, e.participants, e.embedding, e.project_id, e.created_at, e.updated_at,
			       gt.depth + 1, gt.path || e.id, gt.total_strength + r.strength
			FROM graph_traversal gt
			JOIN knowledge_relationships r ON (gt.id = r.source_entity_id OR gt.id = r.target_entity_id)
			JOIN knowledge_entities e ON (e.id = CASE WHEN gt.id = r.source_entity_id THEN r.target_entity_id ELSE r.source_entity_id END)
			WHERE gt.depth < $2
			  AND NOT e.id = ANY(gt.path)  -- Avoid cycles
		)
		SELECT * FROM graph_traversal
		ORDER BY depth, total_strength DESC`

	args := []interface{}{startEntityID, maxDepth}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []models.KnowledgeEntity
	var maxDepthFound int
	var totalStrength float64

	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var projectID *string
		var depth int
		var path []string
		var pathStrength float64

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &projectID, &entity.CreatedAt, &entity.UpdatedAt,
			&depth, &path, &pathStrength)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if projectID != nil && *projectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *projectID
		}

		entities = append(entities, entity)
		if depth > maxDepthFound {
			maxDepthFound = depth
		}
		totalStrength += pathStrength
	}

	// Get relationships for the traversal
	relationships, err := r.getRelationshipsForEntities(ctx, entities, relationshipTypes)
	if err != nil {
		return nil, err
	}

	return &models.GraphTraversalResult{
		Path:          entities,
		Relationships: relationships,
		Depth:         maxDepthFound,
		TotalStrength: totalStrength,
	}, nil
}

func (r *Repository) GetRelatedEntities(ctx context.Context, entityID string, relationshipTypes []string, limit int) ([]models.KnowledgeEntity, error) {
	query := `
		SELECT DISTINCT e.id, e.entity_type, e.entity_id, e.title, e.content, e.metadata, e.platform_source, e.source_event_ids, e.participants, e.embedding, e.project_id, e.created_at, e.updated_at
		FROM knowledge_entities e
		JOIN knowledge_relationships r ON (e.id = r.source_entity_id OR e.id = r.target_entity_id)
		WHERE (r.source_entity_id = $1 OR r.target_entity_id = $1) AND e.id != $1`

	args := []interface{}{entityID}
	argIndex := 2

	// Add relationship type filter
	if len(relationshipTypes) > 0 {
		query += fmt.Sprintf(" AND r.relationship_type = ANY($%d)", argIndex)
		args = append(args, relationshipTypes)
		argIndex++
	}

	query += " ORDER BY e.created_at DESC"

	// Add limit
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []models.KnowledgeEntity
	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var projectID *string

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &projectID, &entity.CreatedAt, &entity.UpdatedAt)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if projectID != nil && *projectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *projectID
		}

		entities = append(entities, entity)
	}

	return entities, rows.Err()
}

// Helper functions

func (r *Repository) getRelationshipsForEntities(ctx context.Context, entities []models.KnowledgeEntity, relationshipTypes []string) ([]models.KnowledgeRelationship, error) {
	if len(entities) == 0 {
		return []models.KnowledgeRelationship{}, nil
	}

	// Build entity ID list
	entityIDs := make([]string, len(entities))
	for i, entity := range entities {
		entityIDs[i] = entity.ID
	}

	query := `
		SELECT id, source_entity_id, target_entity_id, relationship_type, strength, metadata, created_at
		FROM knowledge_relationships
		WHERE source_entity_id = ANY($1) AND target_entity_id = ANY($1)`

	args := []interface{}{entityIDs}

	// Add relationship type filter
	if len(relationshipTypes) > 0 {
		query += " AND relationship_type = ANY($2)"
		args = append(args, relationshipTypes)
	}

	query += " ORDER BY strength DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []models.KnowledgeRelationship
	for rows.Next() {
		var rel models.KnowledgeRelationship
		var metadata models.JSONBMap

		err := rows.Scan(&rel.ID, &rel.SourceEntityID, &rel.TargetEntityID,
			&rel.RelationshipType, &rel.Strength, &metadata, &rel.CreatedAt)
		if err != nil {
			return nil, err
		}

		rel.Metadata = map[string]interface{}(metadata)
		relationships = append(relationships, rel)
	}

	return relationships, rows.Err()
}

// Utility functions for UUID generation and embedding conversion
func generateUUID() string {
	// Simple UUID generation - in production use a proper UUID library
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

func convertFloatSliceToString(embedding []float32) *string {
	if embedding == nil {
		return nil
	}
	// Convert to PostgreSQL vector format: [1.0,2.0,3.0]
	result := "["
	for i, val := range embedding {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%.6f", val)
	}
	result += "]"
	return &result
}

func convertStringToFloatSlice(embeddingStr string) []float32 {
	// Parse PostgreSQL vector format back to float slice
	// This is a simplified implementation - in production use proper parsing
	if embeddingStr == "" || embeddingStr == "[]" {
		return nil
	}
	// For now, return empty slice - proper implementation would parse the string
	return []float32{}
}

// Project-scoped Knowledge Graph operations

// GetKnowledgeEntitiesByProject retrieves knowledge entities for a specific project
func (r *Repository) GetKnowledgeEntitiesByProject(ctx context.Context, projectID string, entityTypes []string, limit int) ([]models.KnowledgeEntity, error) {
	query := `
		SELECT id, entity_type, entity_id, title, content, metadata, platform_source, source_event_ids, participants, embedding, project_id, created_at, updated_at
		FROM knowledge_entities
		WHERE project_id = $1`

	args := []interface{}{projectID}
	argIndex := 2

	// Add entity type filter
	if len(entityTypes) > 0 {
		query += fmt.Sprintf(" AND entity_type = ANY($%d)", argIndex)
		args = append(args, entityTypes)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	// Add limit
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []models.KnowledgeEntity
	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var entityProjectID *string

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &entityProjectID, &entity.CreatedAt, &entity.UpdatedAt)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if entityProjectID != nil && *entityProjectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *entityProjectID
		}

		entities = append(entities, entity)
	}

	return entities, rows.Err()
}

// GetKnowledgeEntityByIDAndProject retrieves a knowledge entity by ID with project validation
func (r *Repository) GetKnowledgeEntityByIDAndProject(ctx context.Context, id, projectID string) (*models.KnowledgeEntity, error) {
	query := `
		SELECT id, entity_type, entity_id, title, content, metadata, platform_source, source_event_ids, participants, embedding, project_id, created_at, updated_at
		FROM knowledge_entities
		WHERE id = $1 AND project_id = $2`

	var entity models.KnowledgeEntity
	var metadata models.JSONBMap
	var embeddingStr *string
	var entityProjectID *string

	err := r.db.QueryRowContext(ctx, query, id, projectID).Scan(
		&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title, &entity.Content,
		&metadata, &entity.PlatformSource, &entity.SourceEventIDs, &entity.Participants,
		&embeddingStr, &entityProjectID, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		return nil, err
	}

	entity.Metadata = map[string]interface{}(metadata)
	if embeddingStr != nil {
		entity.Embedding = convertStringToFloatSlice(*embeddingStr)
	}

	// Add project_id to metadata if present
	if entityProjectID != nil && *entityProjectID != "" {
		if entity.Metadata == nil {
			entity.Metadata = make(map[string]interface{})
		}
		entity.Metadata["project_id"] = *entityProjectID
	}

	return &entity, nil
}

// SearchKnowledgeEntitiesByProject performs search within a specific project
func (r *Repository) SearchKnowledgeEntitiesByProject(ctx context.Context, projectID string, query *models.KnowledgeGraphQuery) ([]models.SearchResult, error) {
	sqlQuery := `
		SELECT id, entity_type, entity_id, title, content, metadata, platform_source, source_event_ids, participants, embedding, project_id, created_at, updated_at,
		       ts_rank(to_tsvector('english', title || ' ' || content), plainto_tsquery('english', $1)) as rank
		FROM knowledge_entities
		WHERE project_id = $2 AND to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', $1)`

	args := []interface{}{query.Query, projectID}
	argIndex := 3

	// Add entity type filter
	if len(query.EntityTypes) > 0 {
		sqlQuery += fmt.Sprintf(" AND entity_type = ANY($%d)", argIndex)
		args = append(args, query.EntityTypes)
		argIndex++
	}

	// Add platform filter
	if len(query.Platforms) > 0 {
		sqlQuery += fmt.Sprintf(" AND platform_source = ANY($%d)", argIndex)
		args = append(args, query.Platforms)
		argIndex++
	}

	// Add date range filter
	if query.DateRange != nil {
		if query.DateRange.Start != nil {
			sqlQuery += fmt.Sprintf(" AND created_at >= $%d", argIndex)
			args = append(args, *query.DateRange.Start)
			argIndex++
		}
		if query.DateRange.End != nil {
			sqlQuery += fmt.Sprintf(" AND created_at <= $%d", argIndex)
			args = append(args, *query.DateRange.End)
			argIndex++
		}
	}

	sqlQuery += " ORDER BY rank DESC"

	// Add limit
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
	} else {
		sqlQuery += " LIMIT 50" // Default limit
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchResult
	rank := 1
	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var entityProjectID *string
		var similarity float64

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &entityProjectID, &entity.CreatedAt, &entity.UpdatedAt, &similarity)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if entityProjectID != nil && *entityProjectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *entityProjectID
		}

		// Don't include content if not requested
		if !query.IncludeContent {
			entity.Content = ""
		}

		results = append(results, models.SearchResult{
			Entity:     entity,
			Similarity: similarity,
			Rank:       rank,
		})
		rank++
	}

	return results, rows.Err()
}

// SearchSimilarEntitiesByProject performs vector similarity search within a specific project
func (r *Repository) SearchSimilarEntitiesByProject(ctx context.Context, projectID string, embedding []float32, entityTypes []string, limit int) ([]models.SearchResult, error) {
	sqlQuery := `
		SELECT id, entity_type, entity_id, title, content, metadata, platform_source, source_event_ids, participants, embedding, project_id, created_at, updated_at,
		       1 - (embedding <=> $1) as similarity
		FROM knowledge_entities
		WHERE project_id = $2 AND embedding IS NOT NULL`

	args := []interface{}{convertFloatSliceToString(embedding), projectID}
	argIndex := 3

	// Add entity type filter
	if len(entityTypes) > 0 {
		sqlQuery += fmt.Sprintf(" AND entity_type = ANY($%d)", argIndex)
		args = append(args, entityTypes)
		argIndex++
	}

	sqlQuery += " ORDER BY embedding <=> $1"

	// Add limit
	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	} else {
		sqlQuery += " LIMIT 10" // Default limit
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchResult
	rank := 1
	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var entityProjectID *string
		var similarity float64

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &entityProjectID, &entity.CreatedAt, &entity.UpdatedAt, &similarity)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if entityProjectID != nil && *entityProjectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *entityProjectID
		}

		results = append(results, models.SearchResult{
			Entity:     entity,
			Similarity: similarity,
			Rank:       rank,
		})
		rank++
	}

	return results, rows.Err()
}

// TraverseKnowledgeGraphByProject performs graph traversal within a specific project
func (r *Repository) TraverseKnowledgeGraphByProject(ctx context.Context, projectID, startEntityID string, maxDepth int, relationshipTypes []string) (*models.GraphTraversalResult, error) {
	query := `
		WITH RECURSIVE graph_traversal AS (
			-- Base case: start with the initial entity
			SELECT e.id, e.entity_type, e.entity_id, e.title, e.content, e.metadata, e.platform_source, e.source_event_ids, e.participants, e.embedding, e.project_id, e.created_at, e.updated_at,
			       0 as depth, ARRAY[e.id] as path, 0.0 as total_strength
			FROM knowledge_entities e
			WHERE e.id = $1 AND e.project_id = $2
			
			UNION ALL
			
			-- Recursive case: follow relationships
			SELECT e.id, e.entity_type, e.entity_id, e.title, e.content, e.metadata, e.platform_source, e.source_event_ids, e.participants, e.embedding, e.project_id, e.created_at, e.updated_at,
			       gt.depth + 1, gt.path || e.id, gt.total_strength + r.strength
			FROM graph_traversal gt
			JOIN knowledge_relationships r ON (gt.id = r.source_entity_id OR gt.id = r.target_entity_id)
			JOIN knowledge_entities e ON (e.id = CASE WHEN gt.id = r.source_entity_id THEN r.target_entity_id ELSE r.source_entity_id END)
			WHERE gt.depth < $3
			  AND e.project_id = $2
			  AND NOT e.id = ANY(gt.path)  -- Avoid cycles
		)
		SELECT * FROM graph_traversal
		ORDER BY depth, total_strength DESC`

	args := []interface{}{startEntityID, projectID, maxDepth}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []models.KnowledgeEntity
	var maxDepthFound int
	var totalStrength float64

	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var entityProjectID *string
		var depth int
		var path []string
		var pathStrength float64

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &entityProjectID, &entity.CreatedAt, &entity.UpdatedAt,
			&depth, &path, &pathStrength)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if entityProjectID != nil && *entityProjectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *entityProjectID
		}

		entities = append(entities, entity)
		if depth > maxDepthFound {
			maxDepthFound = depth
		}
		totalStrength += pathStrength
	}

	// Get relationships for the traversal
	relationships, err := r.getRelationshipsForEntities(ctx, entities, relationshipTypes)
	if err != nil {
		return nil, err
	}

	return &models.GraphTraversalResult{
		Path:          entities,
		Relationships: relationships,
		Depth:         maxDepthFound,
		TotalStrength: totalStrength,
	}, nil
}

// GetRelatedEntitiesByProject retrieves related entities within a specific project
func (r *Repository) GetRelatedEntitiesByProject(ctx context.Context, projectID, entityID string, relationshipTypes []string, limit int) ([]models.KnowledgeEntity, error) {
	query := `
		SELECT DISTINCT e.id, e.entity_type, e.entity_id, e.title, e.content, e.metadata, e.platform_source, e.source_event_ids, e.participants, e.embedding, e.project_id, e.created_at, e.updated_at
		FROM knowledge_entities e
		JOIN knowledge_relationships r ON (e.id = r.source_entity_id OR e.id = r.target_entity_id)
		WHERE (r.source_entity_id = $1 OR r.target_entity_id = $1) AND e.id != $1 AND e.project_id = $2`

	args := []interface{}{entityID, projectID}
	argIndex := 3

	// Add relationship type filter
	if len(relationshipTypes) > 0 {
		query += fmt.Sprintf(" AND r.relationship_type = ANY($%d)", argIndex)
		args = append(args, relationshipTypes)
		argIndex++
	}

	query += " ORDER BY e.created_at DESC"

	// Add limit
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []models.KnowledgeEntity
	for rows.Next() {
		var entity models.KnowledgeEntity
		var metadata models.JSONBMap
		var embeddingStr *string
		var entityProjectID *string

		err := rows.Scan(&entity.ID, &entity.EntityType, &entity.EntityID, &entity.Title,
			&entity.Content, &metadata, &entity.PlatformSource, &entity.SourceEventIDs,
			&entity.Participants, &embeddingStr, &entityProjectID, &entity.CreatedAt, &entity.UpdatedAt)
		if err != nil {
			return nil, err
		}

		entity.Metadata = map[string]interface{}(metadata)
		if embeddingStr != nil {
			entity.Embedding = convertStringToFloatSlice(*embeddingStr)
		}

		// Add project_id to metadata if present
		if entityProjectID != nil && *entityProjectID != "" {
			if entity.Metadata == nil {
				entity.Metadata = make(map[string]interface{})
			}
			entity.Metadata["project_id"] = *entityProjectID
		}

		entities = append(entities, entity)
	}

	return entities, rows.Err()
}

// User operations

func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, avatar_url, email_verified, email_verification_token, email_verification_expires_at, password_reset_token, password_reset_expires_at, last_login_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (email) DO NOTHING`

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	// Generate UUID if not provided
	if user.ID == "" {
		user.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.AvatarURL, user.EmailVerified, user.EmailVerificationToken,
		user.EmailVerificationExpiresAt, user.PasswordResetToken,
		user.PasswordResetExpiresAt, user.LastLoginAt, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *Repository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, avatar_url, email_verified, email_verification_token, email_verification_expires_at, password_reset_token, password_reset_expires_at, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.AvatarURL, &user.EmailVerified, &user.EmailVerificationToken,
		&user.EmailVerificationExpiresAt, &user.PasswordResetToken,
		&user.PasswordResetExpiresAt, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, avatar_url, email_verified, email_verification_token, email_verification_expires_at, password_reset_token, password_reset_expires_at, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.AvatarURL, &user.EmailVerified, &user.EmailVerificationToken,
		&user.EmailVerificationExpiresAt, &user.PasswordResetToken,
		&user.PasswordResetExpiresAt, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", 
		fmt.Sprintf("%s", setParts), argIndex)
	args = append(args, userID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// OAuth account operations

func (r *Repository) CreateOAuthAccount(ctx context.Context, account *models.UserOAuthAccount) error {
	query := `
		INSERT INTO user_oauth_accounts (id, user_id, provider, provider_user_id, provider_username, access_token, refresh_token, token_expires_at, scope, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (provider, provider_user_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			provider_username = EXCLUDED.provider_username,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expires_at = EXCLUDED.token_expires_at,
			scope = EXCLUDED.scope,
			updated_at = EXCLUDED.updated_at`

	now := time.Now()
	if account.CreatedAt.IsZero() {
		account.CreatedAt = now
	}
	account.UpdatedAt = now

	// Generate UUID if not provided
	if account.ID == "" {
		account.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.UserID, account.Provider, account.ProviderUserID,
		account.ProviderUsername, account.AccessToken, account.RefreshToken,
		account.TokenExpiresAt, account.Scope, account.CreatedAt, account.UpdatedAt)
	return err
}

func (r *Repository) GetOAuthAccount(ctx context.Context, provider, providerUserID string) (*models.UserOAuthAccount, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, provider_username, access_token, refresh_token, token_expires_at, scope, created_at, updated_at
		FROM user_oauth_accounts
		WHERE provider = $1 AND provider_user_id = $2`

	var account models.UserOAuthAccount
	err := r.db.QueryRowContext(ctx, query, provider, providerUserID).Scan(
		&account.ID, &account.UserID, &account.Provider, &account.ProviderUserID,
		&account.ProviderUsername, &account.AccessToken, &account.RefreshToken,
		&account.TokenExpiresAt, &account.Scope, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (r *Repository) GetOAuthAccountsByUser(ctx context.Context, userID string) ([]models.UserOAuthAccount, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, provider_username, access_token, refresh_token, token_expires_at, scope, created_at, updated_at
		FROM user_oauth_accounts
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.UserOAuthAccount
	for rows.Next() {
		var account models.UserOAuthAccount
		err := rows.Scan(&account.ID, &account.UserID, &account.Provider, &account.ProviderUserID,
			&account.ProviderUsername, &account.AccessToken, &account.RefreshToken,
			&account.TokenExpiresAt, &account.Scope, &account.CreatedAt, &account.UpdatedAt)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	return accounts, rows.Err()
}

func (r *Repository) UpdateOAuthAccount(ctx context.Context, accountID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE user_oauth_accounts SET %s WHERE id = $%d", 
		fmt.Sprintf("%s", setParts), argIndex)
	args = append(args, accountID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) DeleteOAuthAccount(ctx context.Context, accountID string) error {
	query := `DELETE FROM user_oauth_accounts WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, accountID)
	return err
}

// Project workspace operations

func (r *Repository) CreateProjectWorkspace(ctx context.Context, workspace *models.ProjectWorkspace) error {
	query := `
		INSERT INTO project_workspaces (id, name, description, owner_id, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	now := time.Now()
	if workspace.CreatedAt.IsZero() {
		workspace.CreatedAt = now
	}
	workspace.UpdatedAt = now

	// Generate UUID if not provided
	if workspace.ID == "" {
		workspace.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		workspace.ID, workspace.Name, workspace.Description, workspace.OwnerID,
		models.JSONBMap(workspace.Settings), workspace.CreatedAt, workspace.UpdatedAt)
	return err
}

func (r *Repository) GetProjectWorkspace(ctx context.Context, projectID string) (*models.ProjectWorkspace, error) {
	query := `
		SELECT id, name, description, owner_id, settings, created_at, updated_at
		FROM project_workspaces
		WHERE id = $1`

	var workspace models.ProjectWorkspace
	var settings models.JSONBMap
	err := r.db.QueryRowContext(ctx, query, projectID).Scan(
		&workspace.ID, &workspace.Name, &workspace.Description, &workspace.OwnerID,
		&settings, &workspace.CreatedAt, &workspace.UpdatedAt)
	if err != nil {
		return nil, err
	}

	workspace.Settings = map[string]interface{}(settings)
	return &workspace, nil
}

func (r *Repository) GetProjectWorkspacesByUser(ctx context.Context, userID string) ([]models.ProjectWorkspace, error) {
	query := `
		SELECT DISTINCT pw.id, pw.name, pw.description, pw.owner_id, pw.settings, pw.created_at, pw.updated_at
		FROM project_workspaces pw
		LEFT JOIN project_members pm ON pw.id = pm.project_id
		WHERE pw.owner_id = $1 OR pm.user_id = $1
		ORDER BY pw.updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []models.ProjectWorkspace
	for rows.Next() {
		var workspace models.ProjectWorkspace
		var settings models.JSONBMap
		err := rows.Scan(&workspace.ID, &workspace.Name, &workspace.Description,
			&workspace.OwnerID, &settings, &workspace.CreatedAt, &workspace.UpdatedAt)
		if err != nil {
			return nil, err
		}
		workspace.Settings = map[string]interface{}(settings)
		workspaces = append(workspaces, workspace)
	}

	return workspaces, rows.Err()
}

func (r *Repository) UpdateProjectWorkspace(ctx context.Context, projectID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE project_workspaces SET %s WHERE id = $%d", 
		fmt.Sprintf("%s", setParts), argIndex)
	args = append(args, projectID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) DeleteProjectWorkspace(ctx context.Context, projectID string) error {
	query := `DELETE FROM project_workspaces WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, projectID)
	return err
}

// Project member operations

func (r *Repository) CreateProjectMember(ctx context.Context, member *models.ProjectMember) error {
	query := `
		INSERT INTO project_members (id, project_id, user_id, role, permissions, invited_by, invited_at, joined_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (project_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			permissions = EXCLUDED.permissions,
			invited_by = EXCLUDED.invited_by,
			invited_at = EXCLUDED.invited_at,
			joined_at = EXCLUDED.joined_at`

	if member.CreatedAt.IsZero() {
		member.CreatedAt = time.Now()
	}

	// Generate UUID if not provided
	if member.ID == "" {
		member.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		member.ID, member.ProjectID, member.UserID, member.Role,
		models.JSONBMap(member.Permissions), member.InvitedBy,
		member.InvitedAt, member.JoinedAt, member.CreatedAt)
	return err
}

func (r *Repository) GetProjectMember(ctx context.Context, projectID, userID string) (*models.ProjectMember, error) {
	query := `
		SELECT id, project_id, user_id, role, permissions, invited_by, invited_at, joined_at, created_at
		FROM project_members
		WHERE project_id = $1 AND user_id = $2`

	var member models.ProjectMember
	var permissions models.JSONBMap
	err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(
		&member.ID, &member.ProjectID, &member.UserID, &member.Role,
		&permissions, &member.InvitedBy, &member.InvitedAt,
		&member.JoinedAt, &member.CreatedAt)
	if err != nil {
		return nil, err
	}

	member.Permissions = map[string]interface{}(permissions)
	return &member, nil
}

func (r *Repository) GetProjectMembers(ctx context.Context, projectID string) ([]models.ProjectMember, error) {
	query := `
		SELECT id, project_id, user_id, role, permissions, invited_by, invited_at, joined_at, created_at
		FROM project_members
		WHERE project_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.ProjectMember
	for rows.Next() {
		var member models.ProjectMember
		var permissions models.JSONBMap
		err := rows.Scan(&member.ID, &member.ProjectID, &member.UserID, &member.Role,
			&permissions, &member.InvitedBy, &member.InvitedAt,
			&member.JoinedAt, &member.CreatedAt)
		if err != nil {
			return nil, err
		}
		member.Permissions = map[string]interface{}(permissions)
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *Repository) GetUserProjectMemberships(ctx context.Context, userID string) ([]models.ProjectMember, error) {
	query := `
		SELECT id, project_id, user_id, role, permissions, invited_by, invited_at, joined_at, created_at
		FROM project_members
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memberships []models.ProjectMember
	for rows.Next() {
		var member models.ProjectMember
		var permissions models.JSONBMap
		err := rows.Scan(&member.ID, &member.ProjectID, &member.UserID, &member.Role,
			&permissions, &member.InvitedBy, &member.InvitedAt,
			&member.JoinedAt, &member.CreatedAt)
		if err != nil {
			return nil, err
		}
		member.Permissions = map[string]interface{}(permissions)
		memberships = append(memberships, member)
	}

	return memberships, rows.Err()
}

func (r *Repository) UpdateProjectMember(ctx context.Context, memberID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE project_members SET %s WHERE id = $%d", 
		fmt.Sprintf("%s", setParts), argIndex)
	args = append(args, memberID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) DeleteProjectMember(ctx context.Context, memberID string) error {
	query := `DELETE FROM project_members WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, memberID)
	return err
}

// Project integration operations

func (r *Repository) CreateProjectIntegration(ctx context.Context, integration *models.ProjectIntegration) error {
	query := `
		INSERT INTO project_integrations (id, project_id, platform, integration_type, status, configuration, credentials, last_sync_at, last_sync_status, error_message, sync_checkpoint, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (project_id, platform) DO UPDATE SET
			integration_type = EXCLUDED.integration_type,
			status = EXCLUDED.status,
			configuration = EXCLUDED.configuration,
			credentials = EXCLUDED.credentials,
			last_sync_at = EXCLUDED.last_sync_at,
			last_sync_status = EXCLUDED.last_sync_status,
			error_message = EXCLUDED.error_message,
			sync_checkpoint = EXCLUDED.sync_checkpoint,
			updated_at = EXCLUDED.updated_at`

	now := time.Now()
	if integration.CreatedAt.IsZero() {
		integration.CreatedAt = now
	}
	integration.UpdatedAt = now

	// Generate UUID if not provided
	if integration.ID == "" {
		integration.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		integration.ID, integration.ProjectID, integration.Platform, integration.IntegrationType,
		integration.Status, models.JSONBMap(integration.Configuration),
		models.JSONBMap(integration.Credentials), integration.LastSyncAt,
		integration.LastSyncStatus, integration.ErrorMessage,
		models.JSONBMap(integration.SyncCheckpoint), integration.CreatedBy,
		integration.CreatedAt, integration.UpdatedAt)
	return err
}

func (r *Repository) GetProjectIntegration(ctx context.Context, integrationID string) (*models.ProjectIntegration, error) {
	query := `
		SELECT id, project_id, platform, integration_type, status, configuration, credentials, last_sync_at, last_sync_status, error_message, sync_checkpoint, created_by, created_at, updated_at
		FROM project_integrations
		WHERE id = $1`

	var integration models.ProjectIntegration
	var configuration, credentials, syncCheckpoint models.JSONBMap
	err := r.db.QueryRowContext(ctx, query, integrationID).Scan(
		&integration.ID, &integration.ProjectID, &integration.Platform, &integration.IntegrationType,
		&integration.Status, &configuration, &credentials, &integration.LastSyncAt,
		&integration.LastSyncStatus, &integration.ErrorMessage, &syncCheckpoint,
		&integration.CreatedBy, &integration.CreatedAt, &integration.UpdatedAt)
	if err != nil {
		return nil, err
	}

	integration.Configuration = map[string]interface{}(configuration)
	integration.Credentials = map[string]interface{}(credentials)
	integration.SyncCheckpoint = map[string]interface{}(syncCheckpoint)
	return &integration, nil
}

func (r *Repository) GetProjectIntegrations(ctx context.Context, projectID string) ([]models.ProjectIntegration, error) {
	query := `
		SELECT id, project_id, platform, integration_type, status, configuration, credentials, last_sync_at, last_sync_status, error_message, sync_checkpoint, created_by, created_at, updated_at
		FROM project_integrations
		WHERE project_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var integrations []models.ProjectIntegration
	for rows.Next() {
		var integration models.ProjectIntegration
		var configuration, credentials, syncCheckpoint models.JSONBMap
		err := rows.Scan(&integration.ID, &integration.ProjectID, &integration.Platform,
			&integration.IntegrationType, &integration.Status, &configuration, &credentials,
			&integration.LastSyncAt, &integration.LastSyncStatus, &integration.ErrorMessage,
			&syncCheckpoint, &integration.CreatedBy, &integration.CreatedAt, &integration.UpdatedAt)
		if err != nil {
			return nil, err
		}
		integration.Configuration = map[string]interface{}(configuration)
		integration.Credentials = map[string]interface{}(credentials)
		integration.SyncCheckpoint = map[string]interface{}(syncCheckpoint)
		integrations = append(integrations, integration)
	}

	return integrations, rows.Err()
}

func (r *Repository) GetProjectIntegrationByPlatform(ctx context.Context, projectID, platform string) (*models.ProjectIntegration, error) {
	query := `
		SELECT id, project_id, platform, integration_type, status, configuration, credentials, last_sync_at, last_sync_status, error_message, sync_checkpoint, created_by, created_at, updated_at
		FROM project_integrations
		WHERE project_id = $1 AND platform = $2`

	var integration models.ProjectIntegration
	var configuration, credentials, syncCheckpoint models.JSONBMap
	err := r.db.QueryRowContext(ctx, query, projectID, platform).Scan(
		&integration.ID, &integration.ProjectID, &integration.Platform, &integration.IntegrationType,
		&integration.Status, &configuration, &credentials, &integration.LastSyncAt,
		&integration.LastSyncStatus, &integration.ErrorMessage, &syncCheckpoint,
		&integration.CreatedBy, &integration.CreatedAt, &integration.UpdatedAt)
	if err != nil {
		return nil, err
	}

	integration.Configuration = map[string]interface{}(configuration)
	integration.Credentials = map[string]interface{}(credentials)
	integration.SyncCheckpoint = map[string]interface{}(syncCheckpoint)
	return &integration, nil
}

func (r *Repository) UpdateProjectIntegration(ctx context.Context, integrationID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE project_integrations SET %s WHERE id = $%d", 
		fmt.Sprintf("%s", setParts), argIndex)
	args = append(args, integrationID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) DeleteProjectIntegration(ctx context.Context, integrationID string) error {
	query := `DELETE FROM project_integrations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, integrationID)
	return err
}

// Project data source operations

func (r *Repository) CreateProjectDataSource(ctx context.Context, dataSource *models.ProjectDataSource) error {
	query := `
		INSERT INTO project_data_sources (id, project_id, integration_id, source_type, source_id, source_name, configuration, is_active, last_ingestion_at, ingestion_status, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (integration_id, source_id) DO UPDATE SET
			source_name = EXCLUDED.source_name,
			configuration = EXCLUDED.configuration,
			is_active = EXCLUDED.is_active,
			last_ingestion_at = EXCLUDED.last_ingestion_at,
			ingestion_status = EXCLUDED.ingestion_status,
			error_message = EXCLUDED.error_message,
			updated_at = EXCLUDED.updated_at`

	now := time.Now()
	if dataSource.CreatedAt.IsZero() {
		dataSource.CreatedAt = now
	}
	dataSource.UpdatedAt = now

	// Generate UUID if not provided
	if dataSource.ID == "" {
		dataSource.ID = generateUUID()
	}

	_, err := r.db.ExecContext(ctx, query,
		dataSource.ID, dataSource.ProjectID, dataSource.IntegrationID, dataSource.SourceType,
		dataSource.SourceID, dataSource.SourceName, models.JSONBMap(dataSource.Configuration),
		dataSource.IsActive, dataSource.LastIngestionAt, dataSource.IngestionStatus,
		dataSource.ErrorMessage, dataSource.CreatedAt, dataSource.UpdatedAt)
	return err
}

func (r *Repository) GetProjectDataSource(ctx context.Context, dataSourceID string) (*models.ProjectDataSource, error) {
	query := `
		SELECT id, project_id, integration_id, source_type, source_id, source_name, configuration, is_active, last_ingestion_at, ingestion_status, error_message, created_at, updated_at
		FROM project_data_sources
		WHERE id = $1`

	var dataSource models.ProjectDataSource
	var configuration models.JSONBMap
	err := r.db.QueryRowContext(ctx, query, dataSourceID).Scan(
		&dataSource.ID, &dataSource.ProjectID, &dataSource.IntegrationID, &dataSource.SourceType,
		&dataSource.SourceID, &dataSource.SourceName, &configuration, &dataSource.IsActive,
		&dataSource.LastIngestionAt, &dataSource.IngestionStatus, &dataSource.ErrorMessage,
		&dataSource.CreatedAt, &dataSource.UpdatedAt)
	if err != nil {
		return nil, err
	}

	dataSource.Configuration = map[string]interface{}(configuration)
	return &dataSource, nil
}

func (r *Repository) GetProjectDataSources(ctx context.Context, projectID string) ([]models.ProjectDataSource, error) {
	query := `
		SELECT id, project_id, integration_id, source_type, source_id, source_name, configuration, is_active, last_ingestion_at, ingestion_status, error_message, created_at, updated_at
		FROM project_data_sources
		WHERE project_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataSources []models.ProjectDataSource
	for rows.Next() {
		var dataSource models.ProjectDataSource
		var configuration models.JSONBMap
		err := rows.Scan(&dataSource.ID, &dataSource.ProjectID, &dataSource.IntegrationID,
			&dataSource.SourceType, &dataSource.SourceID, &dataSource.SourceName, &configuration,
			&dataSource.IsActive, &dataSource.LastIngestionAt, &dataSource.IngestionStatus,
			&dataSource.ErrorMessage, &dataSource.CreatedAt, &dataSource.UpdatedAt)
		if err != nil {
			return nil, err
		}
		dataSource.Configuration = map[string]interface{}(configuration)
		dataSources = append(dataSources, dataSource)
	}

	return dataSources, rows.Err()
}

func (r *Repository) GetProjectDataSourcesByIntegration(ctx context.Context, integrationID string) ([]models.ProjectDataSource, error) {
	query := `
		SELECT id, project_id, integration_id, source_type, source_id, source_name, configuration, is_active, last_ingestion_at, ingestion_status, error_message, created_at, updated_at
		FROM project_data_sources
		WHERE integration_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, integrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataSources []models.ProjectDataSource
	for rows.Next() {
		var dataSource models.ProjectDataSource
		var configuration models.JSONBMap
		err := rows.Scan(&dataSource.ID, &dataSource.ProjectID, &dataSource.IntegrationID,
			&dataSource.SourceType, &dataSource.SourceID, &dataSource.SourceName, &configuration,
			&dataSource.IsActive, &dataSource.LastIngestionAt, &dataSource.IngestionStatus,
			&dataSource.ErrorMessage, &dataSource.CreatedAt, &dataSource.UpdatedAt)
		if err != nil {
			return nil, err
		}
		dataSource.Configuration = map[string]interface{}(configuration)
		dataSources = append(dataSources, dataSource)
	}

	return dataSources, rows.Err()
}

func (r *Repository) UpdateProjectDataSource(ctx context.Context, dataSourceID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE project_data_sources SET %s WHERE id = $%d", 
		fmt.Sprintf("%s", setParts), argIndex)
	args = append(args, dataSourceID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) DeleteProjectDataSource(ctx context.Context, dataSourceID string) error {
	query := `DELETE FROM project_data_sources WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, dataSourceID)
	return err
}