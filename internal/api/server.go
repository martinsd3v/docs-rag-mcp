package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/trxio/docs-rag-mcp/internal/api/models"
	"github.com/trxio/docs-rag-mcp/internal/embeddings"
	"github.com/trxio/docs-rag-mcp/internal/mcp"
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
	store           *vector.Store           // Can be nil if no project is active
	embeddingClient embeddings.EmbeddingClient // Can be nil if no project is active
	router          chi.Router
	log             func(format string, args ...interface{})
	projectManager  *ProjectManager
	activeProjectID string
	activeProject   *Project
	mcpHandler      *mcp.MCPHTTPHandler // MCP over HTTP/SSE handler
	mu              sync.RWMutex // Protects store/embeddingClient during hot-swap
}

// NewServer creates a new API server
func NewServer(cfg *Config, store *vector.Store, embClient embeddings.EmbeddingClient, pm *ProjectManager) *Server {
	s := &Server{
		cfg:             cfg,
		store:           store,
		embeddingClient: embClient,
		projectManager:  pm,
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
		// Project management routes (always available)
		r.Get("/projects", s.handleListProjects)
		r.Post("/projects", s.handleCreateProject)
		r.Put("/projects/{id}", s.handleUpdateProject)
		r.Delete("/projects/{id}", s.handleDeleteProject)
		r.Post("/projects/{id}/start", s.handleStartProject)
		r.Post("/projects/stop", s.handleStopProject)
		r.Get("/projects/active", s.handleGetActiveProject)

		// Health check (always available)
		r.Get("/health", s.handleHealth)

		// Routes that require an active project
		r.Group(func(r chi.Router) {
			r.Use(s.requireActiveProject)
			r.Get("/stats", s.handleStats)
			r.Get("/documents", s.handleListDocuments)
			r.Get("/documents/{id}", s.handleGetDocument)
			r.Delete("/documents/{id}", s.handleDeleteDocument)
			r.Post("/search", s.handleSearch)
			r.Post("/upload", s.handleUpload)
		})
	})

	// MCP routes (require active project)
	r.Route("/mcp", func(r chi.Router) {
		r.Use(s.requireActiveProject)
		r.Get("/sse", s.handleMCPSSE)
		r.Post("/message", s.handleMCPMessage)
	})

	// Static files (SPA fallback)
	if s.cfg.StaticDir != "" {
		staticDir := s.cfg.StaticDir
		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the requested path
			path := r.URL.Path

			// Try to serve the static file
			filePath := staticDir + path
			if _, err := os.Stat(filePath); err == nil {
				http.ServeFile(w, r, filePath)
				return
			}

			// For assets directory, return 404 if not found
			if len(path) > 8 && path[:8] == "/assets/" {
				http.NotFound(w, r)
				return
			}

			// Fallback to index.html for SPA routing
			http.ServeFile(w, r, staticDir+"/index.html")
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
		WriteTimeout: 0, // Disabled for SSE long-lived connections
		IdleTimeout:  0, // Disabled - SSE manages keepalive internally
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
	checks := map[string]string{}

	s.mu.RLock()
	hasStore := s.store != nil
	hasEmbeddings := s.embeddingClient != nil
	s.mu.RUnlock()

	// Check database
	if hasStore {
		if _, err := s.store.GetStats(); err != nil {
			checks["database"] = "error: " + err.Error()
		} else {
			checks["database"] = "ok"
		}
	} else {
		checks["database"] = "no project active"
	}

	// Check embeddings
	if hasEmbeddings {
		checks["embeddings"] = "ok"
	} else {
		checks["embeddings"] = "no project active"
	}

	status := "healthy"
	for _, v := range checks {
		if v != "ok" && v != "not configured" && v != "no project active" {
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

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "document id is required", nil)
		return
	}

	if err := s.store.DeleteDocument(id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete document", err)
		return
	}

	s.log("Deleted document %s", id)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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

// MCP HTTP/SSE handlers

// handleMCPSSE handles GET /mcp/sse - establishes SSE connection for MCP
func (s *Server) handleMCPSSE(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	handler := s.mcpHandler
	s.mu.RUnlock()

	if handler == nil {
		s.writeError(w, http.StatusServiceUnavailable, "MCP handler not initialized", nil)
		return
	}

	handler.HandleSSE(w, r)
}

// handleMCPMessage handles POST /mcp/message - receives MCP JSON-RPC messages
func (s *Server) handleMCPMessage(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	handler := s.mcpHandler
	s.mu.RUnlock()

	if handler == nil {
		s.writeError(w, http.StatusServiceUnavailable, "MCP handler not initialized", nil)
		return
	}

	handler.HandleMessage(w, r)
}

// requireActiveProject is middleware that ensures a project is active
func (s *Server) requireActiveProject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		hasProject := s.store != nil
		s.mu.RUnlock()

		if !hasProject {
			s.writeError(w, http.StatusServiceUnavailable, "no project active - select a project first", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// startProject initializes the store and embedding client for a project
func (s *Server) startProject(project *Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If there's already a project running, stop it first
	if s.store != nil {
		s.store.Close()
		s.store = nil
		s.embeddingClient = nil
	}
	if s.mcpHandler != nil {
		s.mcpHandler.Close()
		s.mcpHandler = nil
	}

	// Open the project's database
	store, err := vector.NewStore(project.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Detect provider and create embedding client
	provider := embeddings.DetectProvider(project.Host, project.Token)

	apiKey := project.Token
	if apiKey == "" {
		apiKey = "ollama" // Default for Ollama
	}

	unifiedConfig := &embeddings.UnifiedConfig{
		Provider:  provider,
		APIKey:    apiKey,
		BaseURL:   project.Host,
		Model:     project.Model,
		BatchSize: 100,
	}

	embClient, err := embeddings.NewEmbeddingClient(unifiedConfig)
	if err != nil {
		store.Close()
		return fmt.Errorf("failed to create embedding client: %w", err)
	}

	// Create MCP HTTP handler with tools
	toolsHandler := mcp.NewToolsHandler(store, embClient)
	mcpHandler := mcp.NewMCPHTTPHandler(toolsHandler)
	mcpHandler.SetLogger(s.log)

	// Update server state
	s.store = store
	s.embeddingClient = embClient
	s.activeProjectID = project.ID
	s.activeProject = project
	s.mcpHandler = mcpHandler

	s.log("Started project: %s (host: %s, model: %s)", project.Name, project.Host, project.Model)
	s.log("MCP server available at /mcp/sse")

	return nil
}
