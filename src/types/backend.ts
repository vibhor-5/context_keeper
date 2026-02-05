/**
 * TypeScript interfaces for Go backend integration
 * These match the Go models and API responses
 */

export interface Repository {
  id: number;
  name: string;
  full_name: string;
  owner: string;
  created_at: string;
  updated_at: string;
}

export interface PullRequest {
  id: number;
  repo_id: number;
  number: number;
  title: string;
  body: string;
  author: string;
  state: string;
  created_at: string;
  merged_at?: string;
  files_changed: string[];
  labels: string[];
}

export interface Issue {
  id: number;
  repo_id: number;
  title: string;
  body: string;
  author: string;
  state: string;
  created_at: string;
  closed_at?: string;
  labels: string[];
}

export interface Commit {
  sha: string;
  repo_id: number;
  message: string;
  author: string;
  created_at: string;
  files_changed: string[];
}

export interface IngestionJob {
  id: number;
  repo_id: number;
  status: JobStatus;
  started_at?: string;
  finished_at?: string;
  error_message?: string;
}

export type JobStatus = 'pending' | 'running' | 'completed' | 'partial' | 'failed';

export interface User {
  id: string;
  login: string;
  email: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface ContextQuery {
  repo_id: number;
  query: string;
  mode?: string; // "restore" or "clarify" or "query"
}

export interface Task {
  title: string;
  acceptance: string;
}

export interface PRScaffold {
  branch: string;
  title: string;
  body: string;
}

export interface ContextResponse {
  clarified_goal?: string;
  tasks?: Task[];
  questions?: string[];
  pr_scaffold?: PRScaffold;
  context?: Record<string, any>;
}

export interface RepoContext {
  pull_requests: PullRequest[];
  issues: Issue[];
  commits: Commit[];
}

export interface FilteredRepoData {
  repo: string;
  query: string;
  context: RepoContext;
}

export interface ErrorResponse {
  error: string;
  message: string;
  code: number;
}

// Repository status response (from /api/repos/{id}/status)
export interface RepositoryStatus {
  id: number;
  repo_id: number;
  status: JobStatus;
  started_at?: string;
  finished_at?: string;
  error_message?: string;
}

// Repository list response (from /api/repos)
export interface RepositoryListResponse {
  repositories: Repository[];
}

// Ingest repository request (for /api/repos/ingest)
export interface IngestRepoRequest {
  repo_id: number;
}

// Ingest repository response
export interface IngestRepoResponse {
  job_id: number;
  status: JobStatus;
}