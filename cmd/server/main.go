package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/trxio/docs-rag-mcp/internal/embeddings"
	"github.com/trxio/docs-rag-mcp/internal/mcp"
	"github.com/trxio/docs-rag-mcp/internal/vector"
)

func main() {
	var (
		dbPath    string
		openaiKey string
		baseURL   string
		model     string
		useOllama bool
	)

	rootCmd := &cobra.Command{
		Use:   "docs-rag-mcp-server",
		Short: "MCP server for RAG-based documentation search",
		Long: `An MCP (Model Context Protocol) server that provides semantic search
capabilities for technical documentation (RFCs, ADRs, etc.) to Claude Code.`,
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
					openaiKey = "ollama" // Placeholder, not used
				}
			} else {
				// Get base URL from flag or environment (only for non-Ollama mode)
				if baseURL == "" {
					baseURL = os.Getenv("OPENAI_BASE_URL")
				}
			}

			if openaiKey == "" && !useOllama {
				return fmt.Errorf("OpenAI API key required (use --openai-key, OPENAI_API_KEY env var, or --ollama for offline mode)")
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

			return runServer(dbPath, openaiKey, baseURL, model, useOllama)
		},
	}

	rootCmd.Flags().StringVar(&dbPath, "db", "", "Database path (or DB_PATH env var)")
	rootCmd.Flags().StringVar(&openaiKey, "openai-key", "", "OpenAI API key (or OPENAI_API_KEY env var)")
	rootCmd.Flags().StringVar(&baseURL, "base-url", "", "Custom API base URL (or OPENAI_BASE_URL env var)")
	rootCmd.Flags().StringVar(&model, "model", "text-embedding-3-small", "Embedding model")
	rootCmd.Flags().BoolVar(&useOllama, "ollama", false, "Use Ollama for offline embeddings (default model: nomic-embed-text)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServer(dbPath, openaiKey, baseURL, model string, useOllama bool) error {
	// Log to stderr (stdout is for MCP protocol)
	log := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, "[MCP] "+format+"\n", args...)
	}

	log("Starting MCP server...")
	log("Database: %s", dbPath)

	// Create vector store
	store, err := vector.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	// Detect provider or use explicit config
	provider := embeddings.ProviderOpenAI
	if useOllama {
		provider = embeddings.ProviderOllama
		log("Using Ollama for embeddings (model: %s)", model)
	} else {
		provider = embeddings.DetectProvider(baseURL, openaiKey)
		log("Using %s for embeddings (model: %s)", provider, model)
	}

	// Create embedding client using unified config
	unifiedConfig := &embeddings.UnifiedConfig{
		Provider:  provider,
		APIKey:    openaiKey,
		BaseURL:   baseURL,
		Model:     model,
		BatchSize: 100,
	}

	embClient, err := embeddings.NewEmbeddingClient(unifiedConfig)
	if err != nil {
		return fmt.Errorf("failed to create embedding client: %w", err)
	}

	// Create MCP server
	server := mcp.NewServer()
	server.SetLogger(log)

	// Register tools
	toolsHandler := mcp.NewToolsHandler(store, embClient)
	toolsHandler.RegisterTools(server)

	log("Tools registered, waiting for connections...")

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

	// Run server
	return server.Run(ctx)
}
