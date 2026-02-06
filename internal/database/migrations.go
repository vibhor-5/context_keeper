package database

import (
	"database/sql"
	"fmt"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// migrations contains all database migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Name:    "create_repos_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS repos (
				id BIGSERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				full_name VARCHAR(255) NOT NULL UNIQUE,
				owner VARCHAR(255) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			
			CREATE INDEX IF NOT EXISTS idx_repos_owner ON repos(owner);
		`,
	},
	{
		Version: 2,
		Name:    "create_pull_requests_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS pull_requests (
				id BIGINT PRIMARY KEY,
				repo_id BIGINT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
				number INTEGER NOT NULL,
				title TEXT NOT NULL,
				body TEXT,
				author VARCHAR(255) NOT NULL,
				state VARCHAR(50) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				merged_at TIMESTAMP WITH TIME ZONE,
				files_changed JSONB,
				labels JSONB,
				UNIQUE(repo_id, number)
			);
			
			CREATE INDEX IF NOT EXISTS idx_pull_requests_repo_created ON pull_requests(repo_id, created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_pull_requests_repo_author ON pull_requests(repo_id, author);
		`,
	},
	{
		Version: 3,
		Name:    "create_issues_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS issues (
				id BIGINT PRIMARY KEY,
				repo_id BIGINT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
				title TEXT NOT NULL,
				body TEXT,
				author VARCHAR(255) NOT NULL,
				state VARCHAR(50) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				closed_at TIMESTAMP WITH TIME ZONE,
				labels JSONB
			);
			
			CREATE INDEX IF NOT EXISTS idx_issues_repo_created ON issues(repo_id, created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_issues_repo_author ON issues(repo_id, author);
		`,
	},
	{
		Version: 4,
		Name:    "create_commits_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS commits (
				sha VARCHAR(40) PRIMARY KEY,
				repo_id BIGINT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
				message TEXT NOT NULL,
				author VARCHAR(255) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				files_changed JSONB
			);
			
			CREATE INDEX IF NOT EXISTS idx_commits_repo_created ON commits(repo_id, created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_commits_repo_author ON commits(repo_id, author);
		`,
	},
	{
		Version: 5,
		Name:    "create_ingestion_jobs_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS ingestion_jobs (
				id BIGSERIAL PRIMARY KEY,
				repo_id BIGINT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
				status VARCHAR(50) NOT NULL DEFAULT 'pending',
				started_at TIMESTAMP WITH TIME ZONE,
				finished_at TIMESTAMP WITH TIME ZONE,
				error_message TEXT
			);
			
			CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_repo ON ingestion_jobs(repo_id);
		`,
	},
	{
		Version: 6,
		Name:    "create_schema_migrations_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
		`,
	},
	{
		Version: 7,
		Name:    "create_knowledge_entities_table",
		SQL: `
			-- Enable pgvector extension for vector embeddings
			CREATE EXTENSION IF NOT EXISTS vector;
			
			-- Knowledge entities table for storing all types of knowledge objects
			CREATE TABLE IF NOT EXISTS knowledge_entities (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				entity_type VARCHAR(50) NOT NULL, -- decision, discussion, feature, file_context
				entity_id VARCHAR(255) NOT NULL, -- Original ID from processing
				title TEXT NOT NULL,
				content TEXT NOT NULL,
				metadata JSONB NOT NULL DEFAULT '{}',
				platform_source VARCHAR(50),
				source_event_ids TEXT[],
				participants TEXT[],
				embedding vector(1536), -- OpenAI embedding dimension
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				UNIQUE(entity_type, entity_id)
			);
			
			-- Indexes for efficient querying
			CREATE INDEX IF NOT EXISTS idx_knowledge_entities_type ON knowledge_entities(entity_type);
			CREATE INDEX IF NOT EXISTS idx_knowledge_entities_platform ON knowledge_entities(platform_source);
			CREATE INDEX IF NOT EXISTS idx_knowledge_entities_created ON knowledge_entities(created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_knowledge_entities_participants ON knowledge_entities USING GIN(participants);
			CREATE INDEX IF NOT EXISTS idx_knowledge_entities_metadata ON knowledge_entities USING GIN(metadata);
			
			-- Vector similarity search index
			CREATE INDEX IF NOT EXISTS idx_knowledge_entities_embedding ON knowledge_entities 
			USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
		`,
	},
	{
		Version: 8,
		Name:    "create_knowledge_relationships_table",
		SQL: `
			-- Knowledge relationships table for storing connections between entities
			CREATE TABLE IF NOT EXISTS knowledge_relationships (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				source_entity_id UUID NOT NULL REFERENCES knowledge_entities(id) ON DELETE CASCADE,
				target_entity_id UUID NOT NULL REFERENCES knowledge_entities(id) ON DELETE CASCADE,
				relationship_type VARCHAR(50) NOT NULL, -- relates_to, introduced_by, modified_by, discussed_in, contributed_by
				strength DECIMAL(3,2) NOT NULL DEFAULT 0.0 CHECK (strength >= 0.0 AND strength <= 1.0),
				metadata JSONB NOT NULL DEFAULT '{}',
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				UNIQUE(source_entity_id, target_entity_id, relationship_type)
			);
			
			-- Indexes for relationship queries
			CREATE INDEX IF NOT EXISTS idx_knowledge_relationships_source ON knowledge_relationships(source_entity_id);
			CREATE INDEX IF NOT EXISTS idx_knowledge_relationships_target ON knowledge_relationships(target_entity_id);
			CREATE INDEX IF NOT EXISTS idx_knowledge_relationships_type ON knowledge_relationships(relationship_type);
			CREATE INDEX IF NOT EXISTS idx_knowledge_relationships_strength ON knowledge_relationships(strength DESC);
			CREATE INDEX IF NOT EXISTS idx_knowledge_relationships_metadata ON knowledge_relationships USING GIN(metadata);
		`,
	},
	{
		Version: 9,
		Name:    "create_decision_records_table",
		SQL: `
			-- Decision records table for engineering decisions
			CREATE TABLE IF NOT EXISTS decision_records (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				entity_id UUID NOT NULL REFERENCES knowledge_entities(id) ON DELETE CASCADE,
				decision_id VARCHAR(255) NOT NULL UNIQUE, -- Original decision ID from processing
				title TEXT NOT NULL,
				decision TEXT NOT NULL,
				rationale TEXT,
				alternatives TEXT[],
				consequences TEXT[],
				status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, superseded, deprecated
				platform_source VARCHAR(50) NOT NULL,
				source_event_ids TEXT[] NOT NULL,
				participants TEXT[] NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			
			-- Indexes for decision queries
			CREATE INDEX IF NOT EXISTS idx_decision_records_status ON decision_records(status);
			CREATE INDEX IF NOT EXISTS idx_decision_records_platform ON decision_records(platform_source);
			CREATE INDEX IF NOT EXISTS idx_decision_records_created ON decision_records(created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_decision_records_participants ON decision_records USING GIN(participants);
			CREATE INDEX IF NOT EXISTS idx_decision_records_alternatives ON decision_records USING GIN(alternatives);
		`,
	},
	{
		Version: 10,
		Name:    "create_discussion_summaries_table",
		SQL: `
			-- Discussion summaries table for conversation threads
			CREATE TABLE IF NOT EXISTS discussion_summaries (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				entity_id UUID NOT NULL REFERENCES knowledge_entities(id) ON DELETE CASCADE,
				summary_id VARCHAR(255) NOT NULL UNIQUE, -- Original summary ID from processing
				thread_id VARCHAR(255),
				platform VARCHAR(50) NOT NULL,
				participants TEXT[] NOT NULL,
				summary TEXT NOT NULL,
				key_points TEXT[],
				action_items TEXT[],
				file_references TEXT[],
				feature_references TEXT[],
				created_at TIMESTAMP WITH TIME ZONE NOT NULL
			);
			
			-- Indexes for discussion queries
			CREATE INDEX IF NOT EXISTS idx_discussion_summaries_thread ON discussion_summaries(thread_id);
			CREATE INDEX IF NOT EXISTS idx_discussion_summaries_platform ON discussion_summaries(platform);
			CREATE INDEX IF NOT EXISTS idx_discussion_summaries_created ON discussion_summaries(created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_discussion_summaries_participants ON discussion_summaries USING GIN(participants);
			CREATE INDEX IF NOT EXISTS idx_discussion_summaries_files ON discussion_summaries USING GIN(file_references);
			CREATE INDEX IF NOT EXISTS idx_discussion_summaries_features ON discussion_summaries USING GIN(feature_references);
		`,
	},
	{
		Version: 11,
		Name:    "create_feature_contexts_table",
		SQL: `
			-- Feature contexts table for feature development history
			CREATE TABLE IF NOT EXISTS feature_contexts (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				entity_id UUID NOT NULL REFERENCES knowledge_entities(id) ON DELETE CASCADE,
				feature_id VARCHAR(255) NOT NULL UNIQUE, -- Original feature ID from processing
				feature_name VARCHAR(255) NOT NULL,
				description TEXT,
				status VARCHAR(50) NOT NULL DEFAULT 'planned', -- planned, in_progress, completed, deprecated
				contributors TEXT[] NOT NULL,
				related_files TEXT[],
				discussions TEXT[],
				decisions TEXT[],
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			
			-- Indexes for feature queries
			CREATE INDEX IF NOT EXISTS idx_feature_contexts_name ON feature_contexts(feature_name);
			CREATE INDEX IF NOT EXISTS idx_feature_contexts_status ON feature_contexts(status);
			CREATE INDEX IF NOT EXISTS idx_feature_contexts_created ON feature_contexts(created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_feature_contexts_updated ON feature_contexts(updated_at DESC);
			CREATE INDEX IF NOT EXISTS idx_feature_contexts_contributors ON feature_contexts USING GIN(contributors);
			CREATE INDEX IF NOT EXISTS idx_feature_contexts_files ON feature_contexts USING GIN(related_files);
		`,
	},
	{
		Version: 12,
		Name:    "create_file_context_history_table",
		SQL: `
			-- File context history table for file change and discussion history
			CREATE TABLE IF NOT EXISTS file_context_history (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				entity_id UUID NOT NULL REFERENCES knowledge_entities(id) ON DELETE CASCADE,
				context_id VARCHAR(255) NOT NULL UNIQUE, -- Original context ID from processing
				file_path TEXT NOT NULL,
				change_reason TEXT,
				discussion_context TEXT NOT NULL,
				related_decisions TEXT[],
				contributors TEXT[] NOT NULL,
				platform_sources JSONB NOT NULL DEFAULT '{}',
				created_at TIMESTAMP WITH TIME ZONE NOT NULL
			);
			
			-- Indexes for file context queries
			CREATE INDEX IF NOT EXISTS idx_file_context_history_path ON file_context_history(file_path);
			CREATE INDEX IF NOT EXISTS idx_file_context_history_created ON file_context_history(created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_file_context_history_contributors ON file_context_history USING GIN(contributors);
			CREATE INDEX IF NOT EXISTS idx_file_context_history_decisions ON file_context_history USING GIN(related_decisions);
			CREATE INDEX IF NOT EXISTS idx_file_context_history_platforms ON file_context_history USING GIN(platform_sources);
			
			-- Full-text search index for discussion context
			CREATE INDEX IF NOT EXISTS idx_file_context_history_discussion_fts 
			ON file_context_history USING gin(to_tsvector('english', discussion_context));
		`,
	},
	{
		Version: 13,
		Name:    "create_knowledge_graph_views",
		SQL: `
			-- View for entity relationship graph traversal
			CREATE OR REPLACE VIEW knowledge_graph AS
			SELECT 
				e1.id as source_id,
				e1.entity_type as source_type,
				e1.title as source_title,
				r.relationship_type,
				r.strength,
				e2.id as target_id,
				e2.entity_type as target_type,
				e2.title as target_title,
				r.metadata as relationship_metadata,
				r.created_at as relationship_created_at
			FROM knowledge_entities e1
			JOIN knowledge_relationships r ON e1.id = r.source_entity_id
			JOIN knowledge_entities e2 ON e2.id = r.target_entity_id;
			
			-- View for semantic search with entity details
			CREATE OR REPLACE VIEW searchable_knowledge AS
			SELECT 
				e.id,
				e.entity_type,
				e.entity_id,
				e.title,
				e.content,
				e.metadata,
				e.platform_source,
				e.participants,
				e.embedding,
				e.created_at,
				CASE 
					WHEN e.entity_type = 'decision' THEN d.status
					WHEN e.entity_type = 'feature' THEN f.status
					ELSE NULL
				END as status,
				CASE 
					WHEN e.entity_type = 'discussion' THEN ds.thread_id
					ELSE NULL
				END as thread_id,
				CASE 
					WHEN e.entity_type = 'file_context' THEN fch.file_path
					ELSE NULL
				END as file_path
			FROM knowledge_entities e
			LEFT JOIN decision_records d ON e.id = d.entity_id
			LEFT JOIN discussion_summaries ds ON e.id = ds.entity_id
			LEFT JOIN feature_contexts f ON e.id = f.entity_id
			LEFT JOIN file_context_history fch ON e.id = fch.entity_id;
		`,
	},
	{
		Version: 14,
		Name:    "create_users_table",
		SQL: `
			-- Users table for multi-tenant authentication
			CREATE TABLE IF NOT EXISTS users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL UNIQUE,
				password_hash VARCHAR(255), -- NULL for OAuth-only users
				first_name VARCHAR(100),
				last_name VARCHAR(100),
				avatar_url TEXT,
				email_verified BOOLEAN NOT NULL DEFAULT FALSE,
				email_verification_token VARCHAR(255),
				email_verification_expires_at TIMESTAMP WITH TIME ZONE,
				password_reset_token VARCHAR(255),
				password_reset_expires_at TIMESTAMP WITH TIME ZONE,
				last_login_at TIMESTAMP WITH TIME ZONE,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
			);
			
			-- Indexes for user queries
			CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
			CREATE INDEX IF NOT EXISTS idx_users_email_verification_token ON users(email_verification_token);
			CREATE INDEX IF NOT EXISTS idx_users_password_reset_token ON users(password_reset_token);
			CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
		`,
	},
	{
		Version: 15,
		Name:    "create_user_oauth_accounts_table",
		SQL: `
			-- OAuth accounts table for linking external providers
			CREATE TABLE IF NOT EXISTS user_oauth_accounts (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				provider VARCHAR(50) NOT NULL, -- github, google
				provider_user_id VARCHAR(255) NOT NULL,
				provider_username VARCHAR(255),
				access_token TEXT NOT NULL,
				refresh_token TEXT,
				token_expires_at TIMESTAMP WITH TIME ZONE,
				scope TEXT,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				UNIQUE(provider, provider_user_id)
			);
			
			-- Indexes for OAuth queries
			CREATE INDEX IF NOT EXISTS idx_user_oauth_accounts_user_id ON user_oauth_accounts(user_id);
			CREATE INDEX IF NOT EXISTS idx_user_oauth_accounts_provider ON user_oauth_accounts(provider, provider_user_id);
		`,
	},
	{
		Version: 16,
		Name:    "create_project_workspaces_table",
		SQL: `
			-- Project workspaces table for multi-tenant project organization
			CREATE TABLE IF NOT EXISTS project_workspaces (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name VARCHAR(255) NOT NULL,
				description TEXT,
				owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				settings JSONB NOT NULL DEFAULT '{}',
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
			);
			
			-- Indexes for project queries
			CREATE INDEX IF NOT EXISTS idx_project_workspaces_owner_id ON project_workspaces(owner_id);
			CREATE INDEX IF NOT EXISTS idx_project_workspaces_name ON project_workspaces(name);
			CREATE INDEX IF NOT EXISTS idx_project_workspaces_created_at ON project_workspaces(created_at DESC);
		`,
	},
	{
		Version: 17,
		Name:    "create_project_members_table",
		SQL: `
			-- Project members table for workspace access control
			CREATE TABLE IF NOT EXISTS project_members (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				project_id UUID NOT NULL REFERENCES project_workspaces(id) ON DELETE CASCADE,
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				role VARCHAR(50) NOT NULL DEFAULT 'member', -- owner, admin, member, viewer
				permissions JSONB NOT NULL DEFAULT '{}',
				invited_by UUID REFERENCES users(id),
				invited_at TIMESTAMP WITH TIME ZONE,
				joined_at TIMESTAMP WITH TIME ZONE,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				UNIQUE(project_id, user_id)
			);
			
			-- Indexes for project member queries
			CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members(project_id);
			CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);
			CREATE INDEX IF NOT EXISTS idx_project_members_role ON project_members(role);
		`,
	},
	{
		Version: 18,
		Name:    "add_project_scoping_to_existing_tables",
		SQL: `
			-- Add project_id to existing tables for multi-tenant scoping
			ALTER TABLE repos ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES project_workspaces(id) ON DELETE CASCADE;
			ALTER TABLE knowledge_entities ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES project_workspaces(id) ON DELETE CASCADE;
			
			-- Create indexes for project scoping
			CREATE INDEX IF NOT EXISTS idx_repos_project_id ON repos(project_id);
			CREATE INDEX IF NOT EXISTS idx_knowledge_entities_project_id ON knowledge_entities(project_id);
			
			-- Update existing repos to have a default project (will need to be handled in application logic)
			-- This migration assumes existing data will be migrated to appropriate projects
		`,
	},
	{
		Version: 19,
		Name:    "create_project_integrations_table",
		SQL: `
			-- Project integrations table for platform connections
			CREATE TABLE IF NOT EXISTS project_integrations (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				project_id UUID NOT NULL REFERENCES project_workspaces(id) ON DELETE CASCADE,
				platform VARCHAR(50) NOT NULL, -- github, slack, discord
				integration_type VARCHAR(50) NOT NULL, -- oauth, bot, webhook
				status VARCHAR(50) NOT NULL DEFAULT 'pending', -- active, inactive, error, pending
				configuration JSONB NOT NULL DEFAULT '{}',
				credentials JSONB NOT NULL DEFAULT '{}', -- Encrypted storage
				last_sync_at TIMESTAMP WITH TIME ZONE,
				last_sync_status VARCHAR(50),
				error_message TEXT,
				sync_checkpoint JSONB NOT NULL DEFAULT '{}',
				created_by UUID NOT NULL REFERENCES users(id),
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				UNIQUE(project_id, platform)
			);
			
			-- Indexes for project integration queries
			CREATE INDEX IF NOT EXISTS idx_project_integrations_project_id ON project_integrations(project_id);
			CREATE INDEX IF NOT EXISTS idx_project_integrations_platform ON project_integrations(platform);
			CREATE INDEX IF NOT EXISTS idx_project_integrations_status ON project_integrations(status);
			CREATE INDEX IF NOT EXISTS idx_project_integrations_created_by ON project_integrations(created_by);
		`,
	},
	{
		Version: 20,
		Name:    "create_project_data_sources_table",
		SQL: `
			-- Project data sources table for integration data sources
			CREATE TABLE IF NOT EXISTS project_data_sources (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				project_id UUID NOT NULL REFERENCES project_workspaces(id) ON DELETE CASCADE,
				integration_id UUID NOT NULL REFERENCES project_integrations(id) ON DELETE CASCADE,
				source_type VARCHAR(50) NOT NULL, -- repository, channel, server
				source_id VARCHAR(255) NOT NULL, -- Platform-specific ID
				source_name VARCHAR(255) NOT NULL,
				configuration JSONB NOT NULL DEFAULT '{}',
				is_active BOOLEAN NOT NULL DEFAULT TRUE,
				last_ingestion_at TIMESTAMP WITH TIME ZONE,
				ingestion_status VARCHAR(50),
				error_message TEXT,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				UNIQUE(integration_id, source_id)
			);
			
			-- Indexes for project data source queries
			CREATE INDEX IF NOT EXISTS idx_project_data_sources_project_id ON project_data_sources(project_id);
			CREATE INDEX IF NOT EXISTS idx_project_data_sources_integration_id ON project_data_sources(integration_id);
			CREATE INDEX IF NOT EXISTS idx_project_data_sources_source_type ON project_data_sources(source_type);
			CREATE INDEX IF NOT EXISTS idx_project_data_sources_is_active ON project_data_sources(is_active);
			CREATE INDEX IF NOT EXISTS idx_project_data_sources_source_id ON project_data_sources(source_id);
		`,
	},
}

// Migrate runs all pending migrations
func Migrate(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if err := runMigration(db, migration); err != nil {
			return fmt.Errorf("failed to run migration %d (%s): %w", migration.Version, migration.Name, err)
		}
	}

	return nil
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	_, err := db.Exec(query)
	return err
}

func getCurrentVersion(db *sql.DB) (int, error) {
	var version int
	query := "SELECT COALESCE(MAX(version), 0) FROM schema_migrations"
	err := db.QueryRow(query).Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func runMigration(db *sql.DB, migration Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.SQL); err != nil {
		return err
	}

	// Record migration
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version); err != nil {
		return err
	}

	return tx.Commit()
}