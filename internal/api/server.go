package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/trxio/docs-rag-mcp/internal/api/models"
	"github.com/trxio/docs-rag-mcp/internal/embeddings"
	"github.com/trxio/docs-rag-mcp/internal/vector"
)

const Version = "0.1.0"

// Config holds server configuration
type Config struct {
	Port        int      `json:"port"`
	DBPath      string   `json:"db_path"`
	OpenAIKey   string   `json:"openai_key"`
	BaseURL     string   `json:"base_url"`
	Model       string   `json:"model"`
	UseOllama   bool     `json:"use_ollama"`
	CORSOrigins []string `json:"cors_origins"`
	StaticDir   string   `json:"static_dir"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:        8080,
		DBPath:      "./data/docs.db",
		Model:       "text-embedding-3-small",
		CORSOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		StaticDir:   "./web/dist",
	}
}

// Server represents the HTTP API server
type Server struct {
	cfg             *Config
	store           *vector.Store
	embeddingClient embeddings.EmbeddingClient
	router          chi.Router
	log             func(format string, args ...interface{})
}

// NewServer creates a new API server
func NewServer(cfg *Config, store *vector.Store, embClient embeddings.EmbeddingClient) *Server {
	s := &Server{
		cfg:             cfg,
		store:           store,
		embeddingClient: embClient,
		log: func(format string, args ...interface{}) {
			fmt.Printf("[API] "+format+"\n", args...)
		},
	}

	s.router = s.setupRoutes()
	return s
}

// SetLogger sets the logger function
func (s *Server) SetLogger(log func(format string, args ...interface{})) {
	s.log = log
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/stats", s.handleStats)
		r.Get("/documents", s.handleListDocuments)
		r.Get("/documents/{id}", s.handleGetDocument)
		r.Post("/search", s.handleSearch)
		r.Post("/upload", s.handleUpload)
	})

	// Static files (SPA fallback)
	if s.cfg.StaticDir != "" {
		fileServer := http.FileServer(http.Dir(s.cfg.StaticDir))
		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to serve the file, fallback to index.html for SPA routing
			http.StripPrefix("/", fileServer).ServeHTTP(w, r)
		}))
	}

	return r
}

// Router returns the chi router
func (s *Server) Router() chi.Router {
	return s.router
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.log("Starting server on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		s.log("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return server.ListenAndServe()
}

// Handler implementations

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{
		"database": "ok",
	}

	// Check database
	if _, err := s.store.GetStats(); err != nil {
		checks["database"] = "error: " + err.Error()
	}

	// Check embeddings
	if s.embeddingClient != nil {
		checks["embeddings"] = "ok"
	} else {
		checks["embeddings"] = "not configured"
	}

	status := "healthy"
	for _, v := range checks {
		if v != "ok" && v != "not configured" {
			status = "degraded"
			break
		}
	}

	resp := models.HealthResponse{
		Status:    status,
		Version:   Version,
		Timestamp: time.Now(),
		Checks:    checks,
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetStats()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get stats", err)
		return
	}

	// Get document types breakdown
	docTypes, err := s.store.GetStatsByDocType()
	if err != nil {
		docTypes = make(map[string]int64)
	}

	// Get last indexed time
	var lastIndexed *time.Time
	if t, err := s.store.GetLastIndexedTime(); err == nil && t != nil {
		lastIndexed = t
	}

	resp := models.StatsResponse{
		Documents:       stats["documents"],
		Chunks:          stats["chunks"],
		Embeddings:      stats["embeddings"],
		CrossReferences: stats["cross_references"],
		CacheEntries:    stats["cache_entries"],
		DocTypes:        docTypes,
		LastIndexed:     lastIndexed,
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	docType := r.URL.Query().Get("doc_type")
	status := r.URL.Query().Get("status")

	// Get documents
	docs, total, err := s.store.ListDocumentsPaginated(page, limit, docType, status)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list documents", err)
		return
	}

	// Convert to response format
	summaries := make([]models.DocumentSummary, len(docs))
	for i, doc := range docs {
		chunkCount, _ := s.store.GetChunkCountByDocID(doc.ID)
		summaries[i] = models.DocumentSummary{
			ID:         doc.ID,
			DocType:    doc.DocType,
			Title:      doc.Title,
			Status:     doc.Status,
			Version:    doc.Version,
			Author:     doc.Author,
			IndexedAt:  doc.IndexedAt,
			ChunkCount: chunkCount,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	resp := models.DocumentsResponse{
		Documents: summaries,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "document id is required", nil)
		return
	}

	includeChunks := r.URL.Query().Get("include_chunks") == "true"
	includeRelated := r.URL.Query().Get("include_related") != "false"

	// Get document
	doc, err := s.store.GetDocument(id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get document", err)
		return
	}
	if doc == nil {
		s.writeError(w, http.StatusNotFound, "document not found", nil)
		return
	}

	resp := models.DocumentResponse{
		Document: models.DocumentDetail{
			ID:          doc.ID,
			DocType:     doc.DocType,
			Title:       doc.Title,
			FilePath:    doc.FilePath,
			Status:      doc.Status,
			Version:     doc.Version,
			Author:      doc.Author,
			CreatedDate: doc.CreatedDate,
			LastUpdated: doc.LastUpdated,
			IndexedAt:   doc.IndexedAt,
			Metadata:    doc.Metadata,
		},
	}

	// Include chunks
	if includeChunks {
		chunks, err := s.store.GetChunksByDocID(id)
		if err == nil {
			resp.Chunks = make([]models.ChunkDetail, len(chunks))
			for i, c := range chunks {
				resp.Chunks[i] = models.ChunkDetail{
					ID:           c.ID,
					ChunkIndex:   c.ChunkIndex,
					SectionPath:  c.SectionPath,
					SectionLevel: c.SectionLevel,
					Content:      c.Content,
					TokenCount:   c.TokenCount,
					StartLine:    c.StartLine,
					EndLine:      c.EndLine,
					HasCodeBlock: c.HasCodeBlock,
					HasTable:     c.HasTable,
				}
			}
		}
	}

	// Include related docs
	if includeRelated {
		relatedIDs, err := s.store.GetRelatedDocuments(id, 2)
		if err == nil && len(relatedIDs) > 0 {
			resp.RelatedDocs = make([]models.RelatedDoc, 0, len(relatedIDs))
			for _, relID := range relatedIDs {
				relDoc, err := s.store.GetDocument(relID)
				if err == nil && relDoc != nil {
					resp.RelatedDocs = append(resp.RelatedDocs, models.RelatedDoc{
						ID:      relDoc.ID,
						Title:   relDoc.Title,
						DocType: relDoc.DocType,
					})
				}
			}
		}
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := req.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	req.ApplyDefaults()

	if s.embeddingClient == nil {
		s.writeError(w, http.StatusServiceUnavailable, "search not available - embedding client not configured", nil)
		return
	}

	start := time.Now()

	// Generate query embedding
	embResult, err := s.embeddingClient.Embed(r.Context(), req.Query)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to generate embedding", err)
		return
	}

	// Search
	results, err := s.store.SearchSimilar(embResult.Embedding, req.TopK, req.MinScore, req.DocTypes)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "search failed", err)
		return
	}

	// Convert to response format
	searchResults := make([]models.SearchResult, len(results))
	for i, r := range results {
		content := r.Chunk.Content
		if !req.IncludeContent && len(content) > 300 {
			content = content[:300] + "..."
		}

		searchResults[i] = models.SearchResult{
			DocID:    r.Chunk.DocID,
			DocTitle: r.DocTitle,
			DocType:  r.DocType,
			Chunk: models.SearchResultChunk{
				ID:          r.Chunk.ID,
				SectionPath: r.Chunk.SectionPath,
				Content:     content,
				StartLine:   r.Chunk.StartLine,
				EndLine:     r.Chunk.EndLine,
			},
			Score:       r.Score,
			RelatedDocs: r.RelatedDocs,
		}
	}

	resp := models.SearchResponse{
		Query:        req.Query,
		Results:      searchResults,
		TotalResults: len(results),
		SearchTimeMs: time.Since(start).Milliseconds(),
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// Helper methods

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string, err error) {
	resp := models.ErrorResponse{
		Error: message,
	}
	if err != nil {
		resp.Details = err.Error()
	}
	s.writeJSON(w, status, resp)
}
