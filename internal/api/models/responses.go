package models

import "time"

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// StatsResponse represents index statistics
type StatsResponse struct {
	Documents       int64            `json:"documents"`
	Chunks          int64            `json:"chunks"`
	Embeddings      int64            `json:"embeddings"`
	CrossReferences int64            `json:"cross_references"`
	CacheEntries    int64            `json:"cache_entries"`
	DocTypes        map[string]int64 `json:"doc_types"`
	LastIndexed     *time.Time       `json:"last_indexed,omitempty"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// DocumentSummary represents a document in list view
type DocumentSummary struct {
	ID         string    `json:"id"`
	DocType    string    `json:"doc_type"`
	Title      string    `json:"title"`
	Status     string    `json:"status,omitempty"`
	Version    string    `json:"version,omitempty"`
	Author     string    `json:"author,omitempty"`
	IndexedAt  time.Time `json:"indexed_at"`
	ChunkCount int       `json:"chunk_count"`
}

// DocumentsResponse represents the documents list response
type DocumentsResponse struct {
	Documents  []DocumentSummary `json:"documents"`
	Pagination Pagination        `json:"pagination"`
}

// ChunkDetail represents a chunk in document detail
type ChunkDetail struct {
	ID           int64  `json:"id"`
	ChunkIndex   int    `json:"chunk_index"`
	SectionPath  string `json:"section_path"`
	SectionLevel int    `json:"section_level"`
	Content      string `json:"content"`
	TokenCount   int    `json:"token_count"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	HasCodeBlock bool   `json:"has_code_block"`
	HasTable     bool   `json:"has_table"`
}

// RelatedDoc represents a related document
type RelatedDoc struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	DocType string `json:"doc_type"`
}

// DocumentDetail represents full document details
type DocumentDetail struct {
	ID          string     `json:"id"`
	DocType     string     `json:"doc_type"`
	Title       string     `json:"title"`
	FilePath    string     `json:"file_path"`
	Status      string     `json:"status,omitempty"`
	Version     string     `json:"version,omitempty"`
	Author      string     `json:"author,omitempty"`
	CreatedDate *time.Time `json:"created_date,omitempty"`
	LastUpdated *time.Time `json:"last_updated,omitempty"`
	IndexedAt   time.Time  `json:"indexed_at"`
	Metadata    string     `json:"metadata,omitempty"`
}

// DocumentResponse represents single document response
type DocumentResponse struct {
	Document    DocumentDetail `json:"document"`
	Chunks      []ChunkDetail  `json:"chunks,omitempty"`
	RelatedDocs []RelatedDoc   `json:"related_docs,omitempty"`
}

// SearchResultChunk represents a chunk in search results
type SearchResultChunk struct {
	ID          int64  `json:"id"`
	SectionPath string `json:"section_path"`
	Content     string `json:"content"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
}

// SearchResult represents a single search result
type SearchResult struct {
	DocID       string            `json:"doc_id"`
	DocTitle    string            `json:"doc_title"`
	DocType     string            `json:"doc_type"`
	Chunk       SearchResultChunk `json:"chunk"`
	Score       float32           `json:"score"`
	RelatedDocs []string          `json:"related_docs,omitempty"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Query        string         `json:"query"`
	Results      []SearchResult `json:"results"`
	TotalResults int            `json:"total_results"`
	SearchTimeMs int64          `json:"search_time_ms"`
}

// ErrorResponse represents an error
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// UploadResponse represents the response from document upload
type UploadResponse struct {
	Results []UploadResult `json:"results"`
	Summary UploadSummary  `json:"summary"`
}

// UploadResult represents the result of uploading a single file
type UploadResult struct {
	Filename   string `json:"filename"`
	DocID      string `json:"doc_id"`
	Status     string `json:"status"` // "success", "error", "replaced"
	ChunkCount int    `json:"chunk_count"`
	Error      string `json:"error,omitempty"`
}

// UploadSummary represents summary statistics for an upload operation
type UploadSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Replaced  int `json:"replaced"`
}
