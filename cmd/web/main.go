package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/trxio/docs-rag-mcp/internal/api"
)

func main() {
	var (
		port      int
		staticDir string
		dataDir   string
	)

	rootCmd := &cobra.Command{
		Use:   "docs-rag-web",
		Short: "Web UI server for RAG documentation search",
		Long: `A web server that provides a modern dashboard UI for browsing,
searching, and exploring indexed technical documentation.

The server starts without any project active. Use the web UI to create
and select projects. Each project has its own database and embedding
configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(port, staticDir, dataDir)
		},
	}

	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
	rootCmd.Flags().StringVar(&staticDir, "static-dir", "./web/dist", "Static files directory for frontend")
	rootCmd.Flags().StringVar(&dataDir, "data-dir", "./data", "Data directory for projects and databases")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServer(port int, staticDir, dataDir string) error {
	log := func(format string, args ...interface{}) {
		fmt.Printf("[WEB] "+format+"\n", args...)
	}

	log("Starting web server...")
	log("Data directory: %s", dataDir)
	log("Static files: %s", staticDir)
	log("Port: %d", port)

	// Create project manager
	pm := api.NewProjectManager(dataDir)

	// Create API config
	cfg := &api.Config{
		Port:        port,
		CORSOrigins: []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8080"},
		StaticDir:   staticDir,
	}

	// Create server without store or embedding client (no project active)
	server := api.NewServer(cfg, nil, nil, pm)
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
	log("No project active - select a project in the UI to start")

	// Run server
	return server.Start(ctx)
}
