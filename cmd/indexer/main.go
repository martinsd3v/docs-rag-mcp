package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/trxio/docs-rag-mcp/internal/indexer"
)

var (
	// Global flags
	dbPath    string
	verbose   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "docs-rag-indexer",
		Short: "RAG indexer for technical documentation",
		Long:  `A CLI tool for indexing technical documentation (RFCs, ADRs, etc.) for RAG-based search.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "./data/docs.db", "Database path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add commands
	rootCmd.AddCommand(indexCmd())
	rootCmd.AddCommand(reindexCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(testQueryCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func indexCmd() *cobra.Command {
	var (
		docsPath   string
		openaiKey  string
		baseURL    string
		model      string
		batchSize  int
		dryRun     bool
		docTypes   []string
		useOllama  bool
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index documents from a directory",
		Long:  `Scans a directory for markdown files and indexes them for RAG search.`,
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

			if openaiKey == "" && !dryRun && !useOllama {
				return fmt.Errorf("OpenAI API key required (use --openai-key, OPENAI_API_KEY env var, or --ollama for offline mode)")
			}

			config := &indexer.Config{
				DocsPath:  docsPath,
				DBPath:    dbPath,
				OpenAIKey: openaiKey,
				BaseURL:   baseURL,
				Model:     model,
				BatchSize: batchSize,
				DryRun:    dryRun,
				Verbose:   verbose,
				DocTypes:  docTypes,
				UseOllama: useOllama,
			}

			idx, err := indexer.NewIndexer(config)
			if err != nil {
				return fmt.Errorf("failed to create indexer: %w", err)
			}
			defer idx.Close()

			// Handle interrupts
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				fmt.Println("\nInterrupted, stopping...")
				cancel()
			}()

			// Run indexing
			stats, err := idx.Index(ctx)
			if err != nil {
				return fmt.Errorf("indexing failed: %w", err)
			}

			// Print results
			fmt.Println("\n=== Indexing Complete ===")
			fmt.Printf("Documents processed: %d\n", stats.DocsProcessed)
			fmt.Printf("Documents skipped:   %d\n", stats.DocsSkipped)
			fmt.Printf("Chunks created:      %d\n", stats.ChunksCreated)
			fmt.Printf("Embeddings generated: %d\n", stats.EmbeddingsGenerated)
			fmt.Printf("Cross-references:    %d\n", stats.CrossRefsExtracted)
			fmt.Printf("Cache hits:          %d\n", stats.CacheHits)
			fmt.Printf("Cache misses:        %d\n", stats.CacheMisses)
			fmt.Printf("Duration:            %s\n", stats.Duration)

			if len(stats.Errors) > 0 {
				fmt.Printf("\nErrors (%d):\n", len(stats.Errors))
				for _, e := range stats.Errors {
					fmt.Printf("  - %s\n", e)
				}
			}

			if dryRun {
				fmt.Println("\n(Dry run - no changes made)")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&docsPath, "path", "p", "", "Path to documents directory (required)")
	cmd.Flags().StringVar(&openaiKey, "openai-key", "", "OpenAI API key (or use OPENAI_API_KEY env var)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Custom API base URL (or use OPENAI_BASE_URL env var)")
	cmd.Flags().StringVar(&model, "model", "text-embedding-3-small", "Embedding model")
	cmd.Flags().IntVar(&batchSize, "batch-size", 100, "Batch size for embedding generation")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Parse files without storing or generating embeddings")
	cmd.Flags().StringSliceVar(&docTypes, "doc-types", []string{"RFC", "ADR", "BDR", "Guideline", "Roadmap"}, "Document types to index")
	cmd.Flags().BoolVar(&useOllama, "ollama", false, "Use Ollama for offline embeddings (default model: nomic-embed-text)")

	cmd.MarkFlagRequired("path")

	return cmd
}

func reindexCmd() *cobra.Command {
	var (
		openaiKey   string
		checkModified bool
		forceDocs   []string
	)

	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "Re-index modified documents",
		Long:  `Re-indexes documents that have been modified since last indexing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get API key from flag or environment
			if openaiKey == "" {
				openaiKey = os.Getenv("OPENAI_API_KEY")
			}

			if openaiKey == "" {
				return fmt.Errorf("OpenAI API key required")
			}

			// Get existing documents from database
			config := &indexer.Config{
				DBPath:       dbPath,
				OpenAIKey:    openaiKey,
				Verbose:      verbose,
				ForceReindex: forceDocs,
			}

			idx, err := indexer.NewIndexer(config)
			if err != nil {
				return fmt.Errorf("failed to create indexer: %w", err)
			}
			defer idx.Close()

			status, err := idx.GetStatus()
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			fmt.Printf("Database: %s\n", dbPath)
			fmt.Printf("Documents: %v\n", status["stats"])

			if checkModified {
				fmt.Println("\nChecking for modifications...")
				// TODO: Implement modification check
			}

			if len(forceDocs) > 0 {
				fmt.Printf("\nForce reindexing: %s\n", strings.Join(forceDocs, ", "))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&openaiKey, "openai-key", "", "OpenAI API key")
	cmd.Flags().BoolVar(&checkModified, "check-modified", false, "Check for modified files")
	cmd.Flags().StringSliceVar(&forceDocs, "force-docs", nil, "Force reindex specific documents (e.g., RFC-001,RFC-002)")

	return cmd
}

func statusCmd() *cobra.Command {
	var showStats bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show indexing status",
		Long:  `Shows statistics about indexed documents and database status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &indexer.Config{
				DBPath:  dbPath,
				Verbose: verbose,
			}

			idx, err := indexer.NewIndexer(config)
			if err != nil {
				return fmt.Errorf("failed to create indexer: %w", err)
			}
			defer idx.Close()

			status, err := idx.GetStatus()
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			fmt.Printf("Database: %s\n\n", dbPath)

			if showStats {
				// Show detailed stats
				stats := status["stats"].(map[string]int64)
				fmt.Println("=== Statistics ===")
				fmt.Printf("Documents:        %d\n", stats["documents"])
				fmt.Printf("Chunks:           %d\n", stats["chunks"])
				fmt.Printf("Embeddings:       %d\n", stats["embeddings"])
				fmt.Printf("Cross-references: %d\n", stats["cross_references"])
				fmt.Printf("Cache entries:    %d\n", stats["cache_entries"])

				fmt.Println("\n=== By Type ===")
				byType := status["by_type"].(map[string]int)
				for docType, count := range byType {
					fmt.Printf("%-12s %d\n", docType+":", count)
				}
			} else {
				// Show summary
				stats := status["stats"].(map[string]int64)
				fmt.Printf("Documents: %d\n", stats["documents"])
				fmt.Printf("Chunks:    %d\n", stats["chunks"])
				fmt.Println("\nUse --show-stats for detailed statistics")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showStats, "show-stats", false, "Show detailed statistics")

	return cmd
}

func testQueryCmd() *cobra.Command {
	var (
		openaiKey string
		baseURL   string
		topK      int
		jsonOut   bool
		useOllama bool
	)

	cmd := &cobra.Command{
		Use:   "test-query [query]",
		Short: "Test a search query",
		Long:  `Tests a search query against the indexed documents.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

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
				if openaiKey == "" {
					openaiKey = "ollama"
				}
			} else {
				// Get base URL from flag or environment (only for non-Ollama mode)
				if baseURL == "" {
					baseURL = os.Getenv("OPENAI_BASE_URL")
				}
			}

			if openaiKey == "" && !useOllama {
				return fmt.Errorf("OpenAI API key required (or use --ollama for offline mode)")
			}

			config := &indexer.Config{
				DBPath:    dbPath,
				OpenAIKey: openaiKey,
				BaseURL:   baseURL,
				Verbose:   verbose,
				UseOllama: useOllama,
			}

			idx, err := indexer.NewIndexer(config)
			if err != nil {
				return fmt.Errorf("failed to create indexer: %w", err)
			}
			defer idx.Close()

			ctx := context.Background()
			results, err := idx.TestQuery(ctx, query, topK)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}

			if jsonOut {
				// JSON output
				output, _ := json.MarshalIndent(results, "", "  ")
				fmt.Println(string(output))
			} else {
				// Human-readable output
				fmt.Printf("Query: %s\n", query)
				fmt.Printf("Results: %d\n\n", len(results))

				for i, r := range results {
					fmt.Printf("%d. %s: %s (score: %.3f)\n", i+1, r.DocType, r.DocTitle, r.Score)
					fmt.Printf("   Section: %s\n", r.Chunk.SectionPath)

					// Truncate content for display
					content := r.Chunk.Content
					if len(content) > 200 {
						content = content[:200] + "..."
					}
					// Replace newlines for compact display
					content = strings.ReplaceAll(content, "\n", " ")
					fmt.Printf("   Content: %s\n", content)

					if len(r.RelatedDocs) > 0 {
						fmt.Printf("   Related: %s\n", strings.Join(r.RelatedDocs, ", "))
					}
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&openaiKey, "openai-key", "", "OpenAI API key")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Custom API base URL")
	cmd.Flags().IntVar(&topK, "top-k", 5, "Number of results to return")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output results as JSON")
	cmd.Flags().BoolVar(&useOllama, "ollama", false, "Use Ollama for offline embeddings")

	return cmd
}
