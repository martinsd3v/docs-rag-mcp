package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ProjectResponse represents a project in API responses (with masked token)
type ProjectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	HasToken  bool   `json:"has_token"`
	Model     string `json:"model"`
	DBPath    string `json:"db_path"`
	CreatedAt string `json:"created_at"`
}

// ProjectListResponse represents the list projects response
type ProjectListResponse struct {
	Projects        []ProjectResponse `json:"projects"`
	ActiveProjectID string            `json:"active_project_id"`
}

// ActiveProjectResponse represents the active project response
type ActiveProjectResponse struct {
	Project *ProjectResponse `json:"project"`
	Status  string           `json:"status"` // "running" or "stopped"
}

// CreateProjectRequest represents the create project request
type CreateProjectRequest struct {
	Name  string `json:"name"`
	Host  string `json:"host"`
	Token string `json:"token,omitempty"`
	Model string `json:"model"`
}

// UpdateProjectRequest represents the update project request
type UpdateProjectRequest struct {
	Name        string          `json:"name"`
	Host        string          `json:"host"`
	Token       string          `json:"token,omitempty"`
	Model       string          `json:"model"`
	ChunkConfig *ChunkerConfig  `json:"chunk_config,omitempty"`
}

// toProjectResponse converts a Project to ProjectResponse
func toProjectResponse(p *Project) ProjectResponse {
	return ProjectResponse{
		ID:        p.ID,
		Name:      p.Name,
		Host:      p.Host,
		HasToken:  p.Token != "",
		Model:     p.Model,
		DBPath:    p.DBPath,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// handleListProjects returns all projects
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.projectManager.List()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list projects", err)
		return
	}

	resp := ProjectListResponse{
		Projects:        make([]ProjectResponse, len(projects)),
		ActiveProjectID: s.activeProjectID,
	}

	for i, p := range projects {
		resp.Projects[i] = toProjectResponse(&p)
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleCreateProject creates a new project
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required", nil)
		return
	}
	if req.Host == "" {
		s.writeError(w, http.StatusBadRequest, "host is required", nil)
		return
	}
	if req.Model == "" {
		s.writeError(w, http.StatusBadRequest, "model is required", nil)
		return
	}

	project, err := s.projectManager.Create(req.Name, req.Host, req.Token, req.Model)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create project", err)
		return
	}

	resp := toProjectResponse(project)
	s.writeJSON(w, http.StatusCreated, resp)
}

// handleDeleteProject deletes a project
func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "project id is required", nil)
		return
	}

	// Check if project is currently active
	s.mu.RLock()
	activeID := s.activeProjectID
	s.mu.RUnlock()

	if id == activeID {
		s.writeError(w, http.StatusConflict, "cannot delete active project - stop it first", nil)
		return
	}

	if err := s.projectManager.Delete(id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete project", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleUpdateProject updates an existing project
func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "project id is required", nil)
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validation
	if req.Name == "" || req.Host == "" || req.Model == "" {
		s.writeError(w, http.StatusBadRequest, "name, host, and model are required", nil)
		return
	}

	// Validate chunk config if provided
	if req.ChunkConfig != nil {
		if err := validateChunkConfig(req.ChunkConfig); err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
	}

	// Check if project is currently active
	s.mu.RLock()
	isActive := s.activeProjectID == id
	s.mu.RUnlock()

	if isActive {
		s.writeError(w, http.StatusConflict, "cannot update active project - stop it first", nil)
		return
	}

	project, err := s.projectManager.Update(id, req.Name, req.Host, req.Token, req.Model, req.ChunkConfig)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update project", err)
		return
	}

	resp := toProjectResponse(project)
	s.writeJSON(w, http.StatusOK, resp)
}

// validateChunkConfig validates chunking configuration
func validateChunkConfig(cfg *ChunkerConfig) error {
	if cfg.MinSize < 200 || cfg.MinSize > 1000 {
		return fmt.Errorf("min_size must be between 200 and 1000 tokens")
	}
	if cfg.MaxSize < 500 || cfg.MaxSize > 2000 {
		return fmt.Errorf("max_size must be between 500 and 2000 tokens")
	}
	if cfg.MinSize >= cfg.MaxSize {
		return fmt.Errorf("min_size must be less than max_size")
	}
	if cfg.Overlap < 0 || cfg.Overlap > 500 {
		return fmt.Errorf("overlap must be between 0 and 500 tokens")
	}
	if cfg.Overlap >= cfg.MinSize {
		return fmt.Errorf("overlap must be less than min_size")
	}
	return nil
}

// handleGetActiveProject returns the currently active project
func (s *Server) handleGetActiveProject(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	activeProject := s.activeProject
	s.mu.RUnlock()

	if activeProject == nil {
		s.writeJSON(w, http.StatusOK, ActiveProjectResponse{
			Project: nil,
			Status:  "stopped",
		})
		return
	}

	resp := toProjectResponse(activeProject)
	s.writeJSON(w, http.StatusOK, ActiveProjectResponse{
		Project: &resp,
		Status:  "running",
	})
}

// handleStartProject starts a project (initializes store and embedding client)
func (s *Server) handleStartProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "project id is required", nil)
		return
	}

	project, err := s.projectManager.Get(id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "project not found", err)
		return
	}

	if err := s.startProject(project); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to start project", err)
		return
	}

	resp := toProjectResponse(project)
	s.writeJSON(w, http.StatusOK, ActiveProjectResponse{
		Project: &resp,
		Status:  "running",
	})
}

// handleStopProject stops the active project
func (s *Server) handleStopProject(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.store == nil {
		s.writeError(w, http.StatusBadRequest, "no project running", nil)
		return
	}

	// Close MCP handler
	if s.mcpHandler != nil {
		s.mcpHandler.Close()
		s.mcpHandler = nil
	}

	// Close connections
	s.store.Close()
	s.store = nil
	s.embeddingClient = nil
	s.activeProjectID = ""
	s.activeProject = nil

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}
