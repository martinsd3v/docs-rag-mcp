package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ChunkerConfig represents chunking configuration for a project
type ChunkerConfig struct {
	MinSize           int  `json:"min_size"`            // 200-1000 tokens
	MaxSize           int  `json:"max_size"`            // 500-2000 tokens
	Overlap           int  `json:"overlap"`             // 0-500 tokens
	RespectBoundaries bool `json:"respect_boundaries"`  // true recommended
}

// Project represents a RAG project configuration
type Project struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Host        string          `json:"host"`        // Ex: http://localhost:11434
	Token       string          `json:"token"`       // API key (empty for Ollama)
	Model       string          `json:"model"`       // Ex: nomic-embed-text
	DBPath      string          `json:"db_path"`     // data/projects/{id}.db
	CreatedAt   time.Time       `json:"created_at"`
	ChunkConfig *ChunkerConfig  `json:"chunk_config,omitempty"` // Optional chunking configuration
}

// ProjectStore represents the JSON store for projects
type ProjectStore struct {
	Projects []Project `json:"projects"`
}

// ProjectManager handles CRUD operations for projects
type ProjectManager struct {
	dataDir  string
	filePath string
	mu       sync.RWMutex
}

// NewProjectManager creates a new project manager
func NewProjectManager(dataDir string) *ProjectManager {
	return &ProjectManager{
		dataDir:  dataDir,
		filePath: filepath.Join(dataDir, "projects.json"),
	}
}

// ensureDataDir ensures the data directory exists
func (pm *ProjectManager) ensureDataDir() error {
	projectsDir := filepath.Join(pm.dataDir, "projects")
	return os.MkdirAll(projectsDir, 0755)
}

// load reads the project store from disk
func (pm *ProjectManager) load() (*ProjectStore, error) {
	data, err := os.ReadFile(pm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectStore{Projects: []Project{}}, nil
		}
		return nil, fmt.Errorf("failed to read projects file: %w", err)
	}

	var store ProjectStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse projects file: %w", err)
	}

	return &store, nil
}

// save writes the project store to disk
func (pm *ProjectManager) save(store *ProjectStore) error {
	if err := pm.ensureDataDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal projects: %w", err)
	}

	if err := os.WriteFile(pm.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write projects file: %w", err)
	}

	return nil
}

// List returns all projects
func (pm *ProjectManager) List() ([]Project, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	store, err := pm.load()
	if err != nil {
		return nil, err
	}

	return store.Projects, nil
}

// Get returns a project by ID
func (pm *ProjectManager) Get(id string) (*Project, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	store, err := pm.load()
	if err != nil {
		return nil, err
	}

	for _, p := range store.Projects {
		if p.ID == id {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("project not found: %s", id)
}

// Create creates a new project
func (pm *ProjectManager) Create(name, host, token, model string) (*Project, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	store, err := pm.load()
	if err != nil {
		return nil, err
	}

	// Generate ID
	id := uuid.New().String()[:8]

	// Create project
	project := Project{
		ID:        id,
		Name:      name,
		Host:      host,
		Token:     token,
		Model:     model,
		DBPath:    filepath.Join(pm.dataDir, "projects", id+".db"),
		CreatedAt: time.Now(),
	}

	store.Projects = append(store.Projects, project)

	if err := pm.save(store); err != nil {
		return nil, err
	}

	return &project, nil
}

// Update updates an existing project
func (pm *ProjectManager) Update(id string, name, host, token, model string, chunkConfig *ChunkerConfig) (*Project, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	store, err := pm.load()
	if err != nil {
		return nil, err
	}

	// Find and update project
	for i, p := range store.Projects {
		if p.ID == id {
			store.Projects[i].Name = name
			store.Projects[i].Host = host
			store.Projects[i].Token = token
			store.Projects[i].Model = model
			store.Projects[i].ChunkConfig = chunkConfig

			if err := pm.save(store); err != nil {
				return nil, err
			}

			return &store.Projects[i], nil
		}
	}

	return nil, fmt.Errorf("project not found: %s", id)
}

// Delete removes a project and its database
func (pm *ProjectManager) Delete(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	store, err := pm.load()
	if err != nil {
		return err
	}

	var project *Project
	var newProjects []Project
	for _, p := range store.Projects {
		if p.ID == id {
			project = &p
		} else {
			newProjects = append(newProjects, p)
		}
	}

	if project == nil {
		return fmt.Errorf("project not found: %s", id)
	}

	// Delete database file if it exists
	if _, err := os.Stat(project.DBPath); err == nil {
		if err := os.Remove(project.DBPath); err != nil {
			return fmt.Errorf("failed to delete database: %w", err)
		}
	}

	store.Projects = newProjects
	return pm.save(store)
}

// MaskToken returns a masked version of the token for display
func MaskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
