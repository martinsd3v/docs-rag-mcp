package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/trxio/docs-rag-mcp/internal/api"
	"github.com/trxio/docs-rag-mcp/internal/embeddings"
	"github.com/trxio/docs-rag-mcp/internal/vector"
)

func main() {
	var (
		port      int
		dbPath    string
		openaiKey string
		baseURL   string
		model     string
		useOllama bool
		staticDir string
	)

	rootCmd := &cobra.Command{
		Use:   "docs-rag-web",
		Short: "Web UI server for RAG documentation search",
		Long: `A web server that provides a modern dashboard UI for browsing,
searching, and exploring indexed technical documentation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get API key from flag or environment
			if openaiKey == "" {
				openaiKey = os.Getenv("OPENAI_API_KEY")
			}

			// Check for Ollama env var
			if os.Getenv("USE_OLLAMA") == "true" || os.Getenv("USE_OLLAMA") == "1" {
				useOllama = true
			}

			// If using Ollama, set defaults (ignore OPENAI_BASE_URL)
			if useOllama {
				if baseURL == "" {
					baseURL = "http://localhost:11434"
				}
				if model == "" || model == "text-embedding-3-small" {
					model = "nomic-embed-text"
				}
				if openaiKey == "" {
					openaiKey = "ollama"
				}
			} else {
				// Get base URL from flag or environment
				if baseURL == "" {
					baseURL = os.Getenv("OPENAI_BASE_URL")
				}
			}

			// Get DB path from flag or environment
			if dbPath == "" {
				dbPath = os.Getenv("DB_PATH")
			}
			if dbPath == "" {
				dbPath = "./data/docs.db"
			}

			// Check if database exists
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				return fmt.Errorf("database not found: %s (run indexer first)", dbPath)
			}

			return runServer(port, dbPath, openaiKey, baseURL, model, useOllama, staticDir)
		},
	}

	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
	rootCmd.Flags().StringVar(&dbPath, "db", "", "Database path (or DB_PATH env var)")
	rootCmd.Flags().StringVar(&openaiKey, "openai-key", "", "OpenAI API key (or OPENAI_API_KEY env var)")
	rootCmd.Flags().StringVar(&baseURL, "base-url", "", "Custom API base URL (or OPENAI_BASE_URL env var)")
	rootCmd.Flags().StringVar(&model, "model", "text-embedding-3-small", "Embedding model")
	rootCmd.Flags().BoolVar(&useOllama, "ollama", false, "Use Ollama for offline embeddings")
	rootCmd.Flags().StringVar(&staticDir, "static-dir", "./web/dist", "Static files directory for frontend")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServer(port int, dbPath, openaiKey, baseURL, model string, useOllama bool, staticDir string) error {
	log := func(format string, args ...interface{}) {
		fmt.Printf("[WEB] "+format+"\n", args...)
	}

	log("Starting web server...")
	log("Database: %s", dbPath)
	log("Port: %d", port)

	// Create vector store
	store, err := vector.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Create embedding client
	var embClient embeddings.EmbeddingClient
	if openaiKey != "" {
		provider := embeddings.ProviderOpenAI
		if useOllama {
			provider = embeddings.ProviderOllama
			log("Using Ollama for embeddings (model: %s)", model)
		} else {
			provider = embeddings.DetectProvider(baseURL, openaiKey)
			log("Using %s for embeddings (model: %s)", provider, model)
		}

		unifiedConfig := &embeddings.UnifiedConfig{
			Provider:  provider,
			APIKey:    openaiKey,
			BaseURL:   baseURL,
			Model:     model,
			BatchSize: 100,
		}

		embClient, err = embeddings.NewEmbeddingClient(unifiedConfig)
		if err != nil {
			log("Warning: failed to create embedding client: %v", err)
			log("Search functionality will be disabled")
		}
	} else {
		log("Warning: no API key provided, search will be disabled")
	}

	// Create API config
	cfg := &api.Config{
		Port:        port,
		DBPath:      dbPath,
		OpenAIKey:   openaiKey,
		BaseURL:     baseURL,
		Model:       model,
		UseOllama:   useOllama,
		CORSOrigins: []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8080"},
		StaticDir:   staticDir,
	}

	// Create server
	server := api.NewServer(cfg, store, embClient)
	server.SetLogger(log)

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log("Received shutdown signal")
		cancel()
	}()

	log("Server starting at http://localhost:%d", port)
	log("API endpoints at http://localhost:%d/api", port)

	// Run server
	return server.Start(ctx)
}
