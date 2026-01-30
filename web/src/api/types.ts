// API Types

// Project types
export interface ChunkConfig {
  min_size: number          // 200-1000 tokens
  max_size: number          // 500-2000 tokens
  overlap: number           // 0-500 tokens
  respect_boundaries: boolean // true recommended
}

export interface Project {
  id: string
  name: string
  host: string
  has_token: boolean
  model: string
  db_path: string
  created_at: string
  chunk_config?: ChunkConfig
}

export interface ProjectListResponse {
  projects: Project[]
  active_project_id: string | null
}

export interface ActiveProjectResponse {
  project: Project | null
  status: 'running' | 'stopped'
}

export interface CreateProjectRequest {
  name: string
  host: string
  token?: string
  model: string
}

export interface UpdateProjectRequest {
  name: string
  host: string
  token?: string
  model: string
  chunk_config?: ChunkConfig
}

export interface HealthResponse {
  status: string
  version: string
  timestamp: string
  checks: Record<string, string>
}

export interface StatsResponse {
  documents: number
  chunks: number
  embeddings: number
  cross_references: number
  cache_entries: number
  doc_types: Record<string, number>
  last_indexed: string | null
}

export interface DocumentSummary {
  id: string
  doc_type: string
  title: string
  status?: string
  version?: string
  author?: string
  indexed_at: string
  chunk_count: number
}

export interface Pagination {
  page: number
  limit: number
  total: number
  total_pages: number
}

export interface DocumentsResponse {
  documents: DocumentSummary[]
  pagination: Pagination
}

export interface ChunkDetail {
  id: number
  chunk_index: number
  section_path: string
  section_level: number
  content: string
  token_count: number
  start_line: number
  end_line: number
  has_code_block: boolean
  has_table: boolean
}

export interface RelatedDoc {
  id: string
  title: string
  doc_type: string
}

export interface DocumentDetail {
  id: string
  doc_type: string
  title: string
  file_path: string
  status?: string
  version?: string
  author?: string
  created_date?: string
  last_updated?: string
  indexed_at: string
  metadata?: string
}

export interface DocumentResponse {
  document: DocumentDetail
  chunks?: ChunkDetail[]
  related_docs?: RelatedDoc[]
}

export interface SearchResultChunk {
  id: number
  section_path: string
  content: string
  start_line: number
  end_line: number
}

export interface SearchResult {
  doc_id: string
  doc_title: string
  doc_type: string
  chunk: SearchResultChunk
  score: number
  related_docs?: string[]
}

export interface SearchResponse {
  query: string
  results: SearchResult[]
  total_results: number
  search_time_ms: number
}

export interface SearchRequest {
  query: string
  doc_types?: string[]
  top_k?: number
  min_score?: number
  include_content?: boolean
}

export interface UploadResult {
  filename: string
  doc_id: string
  status: 'success' | 'error' | 'replaced'
  chunk_count: number
  error?: string
}

export interface UploadSummary {
  total: number
  succeeded: number
  failed: number
  replaced: number
}

export interface UploadResponse {
  results: UploadResult[]
  summary: UploadSummary
}
