package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trxio/docs-rag-mcp/internal/embeddings"
	"github.com/trxio/docs-rag-mcp/internal/parser"
	"github.com/trxio/docs-rag-mcp/internal/vector"
)

// Config holds indexer configuration
type Config struct {
	DocsPath     string   `json:"docs_path"`
	DBPath       string   `json:"db_path"`
	OpenAIKey    string   `json:"openai_key"`
	BaseURL      string   `json:"base_url"`
	Model        string   `json:"model"`
	BatchSize    int      `json:"batch_size"`
	DryRun       bool     `json:"dry_run"`
	Verbose      bool     `json:"verbose"`
	DocTypes     []string `json:"doc_types"`
	ForceReindex []string `json:"force_reindex"`
	UseOllama    bool     `json:"use_ollama"` // Use Ollama for offline embeddings
}

// DefaultConfig returns default indexer configuration
func DefaultConfig() *Config {
	return &Config{
		Model:     "text-embedding-3-small",
		BatchSize: 100,
		DocTypes:  []string{"RFC", "ADR", "BDR", "Guideline", "Roadmap"},
	}
}

// Stats holds indexing statistics
type Stats struct {
	DocsProcessed       int           `json:"docs_processed"`
	DocsSkipped         int           `json:"docs_skipped"`
	ChunksCreated       int           `json:"chunks_created"`
	EmbeddingsGenerated int           `json:"embeddings_generated"`
	CrossRefsExtracted  int           `json:"cross_refs_extracted"`
	CacheHits           int           `json:"cache_hits"`
	CacheMisses         int           `json:"cache_misses"`
	Errors              []string      `json:"errors"`
	Duration            time.Duration `json:"duration"`
}

// Indexer handles document indexing
type Indexer struct {
	config          *Config
	store           *vector.Store
	embeddingClient embeddings.EmbeddingClient
	parser          *parser.Parser
	chunker         *parser.Chunker
	stats           Stats
	log             func(format string, args ...interface{})
}

// NewIndexer creates a new indexer
func NewIndexer(config *Config) (*Indexer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate config - DocsPath only required for indexing
	if config.DBPath == "" {
		return nil, fmt.Errorf("database path is required")
	}

	// Create store
	store, err := vector.NewStore(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Create embedding client
	var embClient embeddings.EmbeddingClient
	if !config.DryRun {
		// Detect provider or use explicit config
		provider := embeddings.ProviderOpenAI
		if config.UseOllama {
			provider = embeddings.ProviderOllama
		} else {
			provider = embeddings.DetectProvider(config.BaseURL, config.OpenAIKey)
		}

		unifiedConfig := &embeddings.UnifiedConfig{
			Provider:  provider,
			APIKey:    config.OpenAIKey,
			BaseURL:   config.BaseURL,
			Model:     config.Model,
			BatchSize: config.BatchSize,
		}

		embClient, err = embeddings.NewEmbeddingClient(unifiedConfig)
		if err != nil {
			store.Close()
			return nil, fmt.Errorf("failed to create embedding client: %w", err)
		}
	}

	// Create parser and chunker
	mdParser := parser.NewParser()
	chunkerConfig := parser.DefaultChunkerConfig()
	chunker := parser.NewChunker(chunkerConfig)

	// Create logger
	log := func(format string, args ...interface{}) {
		if config.Verbose {
			fmt.Printf(format+"\n", args...)
		}
	}

	return &Indexer{
		config:          config,
		store:           store,
		embeddingClient: embClient,
		parser:          mdParser,
		chunker:         chunker,
		log:             log,
	}, nil
}

// Close closes the indexer resources
func (idx *Indexer) Close() error {
	return idx.store.Close()
}

// Index indexes all documents in the configured path
func (idx *Indexer) Index(ctx context.Context) (*Stats, error) {
	if idx.config.DocsPath == "" {
		return nil, fmt.Errorf("docs path is required for indexing")
	}

	startTime := time.Now()
	idx.stats = Stats{
		Errors: make([]string, 0),
	}

	idx.log("Starting indexing from: %s", idx.config.DocsPath)

	// Find all markdown files
	files, err := idx.findMarkdownFiles(idx.config.DocsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find files: %w", err)
	}

	idx.log("Found %d markdown files", len(files))

	// Process each file
	for _, filePath := range files {
		select {
		case <-ctx.Done():
			return &idx.stats, ctx.Err()
		default:
		}

		err := idx.processFile(ctx, filePath)
		if err != nil {
			errMsg := fmt.Sprintf("Error processing %s: %v", filePath, err)
			idx.stats.Errors = append(idx.stats.Errors, errMsg)
			idx.log("ERROR: %s", errMsg)
		}
	}

	idx.stats.Duration = time.Since(startTime)

	// Save indexing stats
	if !idx.config.DryRun {
		errJSON, _ := json.Marshal(idx.stats.Errors)
		idx.store.SaveIndexingStats(
			"index",
			idx.stats.DocsProcessed,
			idx.stats.ChunksCreated,
			idx.stats.EmbeddingsGenerated,
			idx.stats.CacheHits,
			idx.stats.CacheMisses,
			idx.stats.Duration.Milliseconds(),
			string(errJSON),
			startTime,
		)
	}

	return &idx.stats, nil
}

// findMarkdownFiles recursively finds all .md files
func (idx *Indexer) findMarkdownFiles(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .md files
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// processFile processes a single file
func (idx *Indexer) processFile(ctx context.Context, filePath string) error {
	idx.log("Processing: %s", filePath)

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate file hash
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	// Check if file already indexed with same hash
	existingDoc, err := idx.store.GetDocumentByPath(filePath)
	if err != nil {
		return fmt.Errorf("failed to check existing document: %w", err)
	}

	// Skip if hash matches and not in force reindex list
	if existingDoc != nil && existingDoc.FileHash == hash {
		if !idx.shouldForceReindex(existingDoc.ID) {
			idx.log("  Skipping (unchanged): %s", existingDoc.ID)
			idx.stats.DocsSkipped++
			return nil
		}
		idx.log("  Force reindexing: %s", existingDoc.ID)
	}

	// Parse document
	doc, err := idx.parser.ParseFile(filePath, content)
	if err != nil {
		return fmt.Errorf("failed to parse document: %w", err)
	}

	// Validate document type
	if !idx.isAllowedDocType(doc.Type) {
		idx.log("  Skipping (type not allowed): %s", doc.Type)
		idx.stats.DocsSkipped++
		return nil
	}

	idx.log("  Parsed: %s (%s)", doc.ID, doc.Type)

	// Chunk document
	chunks, err := idx.chunker.ChunkDocument(doc)
	if err != nil {
		return fmt.Errorf("failed to chunk document: %w", err)
	}

	idx.log("  Created %d chunks", len(chunks))

	if idx.config.DryRun {
		idx.stats.DocsProcessed++
		idx.stats.ChunksCreated += len(chunks)
		return nil
	}

	// Delete existing chunks if updating
	if existingDoc != nil {
		if err := idx.store.DeleteChunksByDocID(existingDoc.ID); err != nil {
			return fmt.Errorf("failed to delete existing chunks: %w", err)
		}
	}

	// Create document metadata JSON
	metadataJSON, _ := json.Marshal(doc.Metadata)

	// Insert document
	storeDoc := &vector.Document{
		ID:          doc.ID,
		DocType:     doc.Type,
		Title:       doc.Title,
		FilePath:    filePath,
		Status:      doc.Status,
		Version:     doc.Version,
		CreatedDate: doc.CreatedDate,
		LastUpdated: doc.LastUpdated,
		Author:      doc.Author,
		FileHash:    hash,
		Metadata:    string(metadataJSON),
	}

	if err := idx.store.InsertDocument(storeDoc); err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	// Insert chunks
	chunkIDs := make([]int64, len(chunks))
	chunkContents := make([]string, len(chunks))

	for i, chunk := range chunks {
		storeChunk := &vector.Chunk{
			DocID:        doc.ID,
			ChunkIndex:   chunk.Index,
			SectionPath:  chunk.SectionPath,
			SectionLevel: chunk.SectionLevel,
			Content:      chunk.Content,
			ContentHash:  chunk.ContentHash,
			TokenCount:   chunk.TokenCount,
			StartLine:    chunk.StartLine,
			EndLine:      chunk.EndLine,
			HasCodeBlock: chunk.HasCodeBlock,
			HasTable:     chunk.HasTable,
		}

		chunkID, err := idx.store.InsertChunk(storeChunk)
		if err != nil {
			return fmt.Errorf("failed to insert chunk %d: %w", i, err)
		}

		chunkIDs[i] = chunkID
		chunkContents[i] = chunk.Content
	}

	idx.stats.ChunksCreated += len(chunks)

	// Generate embeddings
	if idx.embeddingClient != nil && len(chunkContents) > 0 {
		idx.log("  Generating embeddings for %d chunks", len(chunkContents))

		result, err := idx.embeddingClient.EmbedBatch(ctx, chunkContents)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		// Store embeddings
		for i, embResult := range result.Results {
			if err := idx.store.InsertChunkEmbedding(chunkIDs[i], embResult.Embedding, idx.config.Model); err != nil {
				return fmt.Errorf("failed to store embedding %d: %w", i, err)
			}
		}

		idx.stats.EmbeddingsGenerated += len(result.Results)
		idx.stats.CacheHits += result.CacheHits
		idx.stats.CacheMisses += result.CacheMisses

		idx.log("  Generated %d embeddings (cache hits: %d)", len(result.Results), result.CacheHits)
	}

	// Extract and store cross-references
	for _, link := range doc.Links {
		if link.DocID != "" && link.DocID != doc.ID {
			ref := &vector.CrossReference{
				SourceDocID:   doc.ID,
				TargetDocID:   link.DocID,
				ReferenceType: "link",
				Context:       link.Text,
				LineNumber:    link.LineNumber,
			}

			if err := idx.store.InsertCrossReference(ref); err != nil {
				// Log but don't fail
				idx.log("  Warning: failed to insert cross-reference to %s: %v", link.DocID, err)
			} else {
				idx.stats.CrossRefsExtracted++
			}
		}
	}

	idx.stats.DocsProcessed++
	idx.log("  Done: %s", doc.ID)

	return nil
}

// shouldForceReindex checks if document should be force reindexed
func (idx *Indexer) shouldForceReindex(docID string) bool {
	for _, id := range idx.config.ForceReindex {
		if id == docID {
			return true
		}
	}
	return false
}

// isAllowedDocType checks if document type is in allowed list
func (idx *Indexer) isAllowedDocType(docType string) bool {
	if len(idx.config.DocTypes) == 0 {
		return true
	}

	for _, allowed := range idx.config.DocTypes {
		if strings.EqualFold(allowed, docType) {
			return true
		}
	}

	// Also allow empty type (will be categorized as unknown)
	return docType == ""
}

// GetStatus returns indexing status
func (idx *Indexer) GetStatus() (map[string]interface{}, error) {
	stats, err := idx.store.GetStats()
	if err != nil {
		return nil, err
	}

	docs, err := idx.store.ListDocuments()
	if err != nil {
		return nil, err
	}

	// Group by type
	byType := make(map[string]int)
	for _, doc := range docs {
		byType[doc.DocType]++
	}

	return map[string]interface{}{
		"database": idx.config.DBPath,
		"stats":    stats,
		"by_type":  byType,
	}, nil
}

// TestQuery performs a test search query
func (idx *Indexer) TestQuery(ctx context.Context, query string, topK int) ([]vector.SearchResult, error) {
	if idx.embeddingClient == nil {
		return nil, fmt.Errorf("embedding client not initialized (API key required)")
	}

	// Generate query embedding
	result, err := idx.embeddingClient.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search
	results, err := idx.store.SearchSimilar(result.Embedding, topK, 0.0, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	return results, nil
}
