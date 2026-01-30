package vector

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/trxio/docs-rag-mcp/internal/embeddings"
)

// Store wraps SQLite operations for document storage and vector search
type Store struct {
	db     *sql.DB
	dbPath string
}

// NewStore creates a new vector store
func NewStore(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database with foreign keys enabled for CASCADE support
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-64000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &Store{
		db:     db,
		dbPath: dbPath,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates database tables if they don't exist
func (s *Store) initSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			doc_type TEXT NOT NULL,
			title TEXT NOT NULL,
			file_path TEXT NOT NULL UNIQUE,
			status TEXT,
			version TEXT,
			created_date TIMESTAMP,
			last_updated TIMESTAMP,
			author TEXT,
			file_hash TEXT NOT NULL,
			metadata_json TEXT,
			indexed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_documents_type ON documents(doc_type);
		CREATE INDEX IF NOT EXISTS idx_documents_hash ON documents(file_hash);

		CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_id TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			section_path TEXT,
			section_level INTEGER,
			content TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			token_count INTEGER NOT NULL,
			start_line INTEGER,
			end_line INTEGER,
			has_code_block BOOLEAN DEFAULT 0,
			has_table BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (doc_id) REFERENCES documents(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_chunks_doc_id ON chunks(doc_id);
		CREATE INDEX IF NOT EXISTS idx_chunks_content_hash ON chunks(content_hash);

		CREATE TABLE IF NOT EXISTS chunk_embeddings (
			chunk_id INTEGER PRIMARY KEY,
			embedding BLOB NOT NULL,
			model TEXT NOT NULL DEFAULT 'text-embedding-3-small',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS cross_references (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_doc_id TEXT NOT NULL,
			target_doc_id TEXT NOT NULL,
			reference_type TEXT NOT NULL,
			context TEXT,
			line_number INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (source_doc_id) REFERENCES documents(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_cross_refs_source ON cross_references(source_doc_id);
		CREATE INDEX IF NOT EXISTS idx_cross_refs_target ON cross_references(target_doc_id);

		CREATE TABLE IF NOT EXISTS embedding_cache (
			content_hash TEXT PRIMARY KEY,
			embedding BLOB NOT NULL,
			model TEXT NOT NULL,
			token_count INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			use_count INTEGER DEFAULT 1
		);

		CREATE INDEX IF NOT EXISTS idx_cache_last_used ON embedding_cache(last_used);

		CREATE TABLE IF NOT EXISTS indexing_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			operation TEXT NOT NULL,
			docs_processed INTEGER NOT NULL,
			chunks_created INTEGER NOT NULL,
			embeddings_generated INTEGER NOT NULL,
			cache_hits INTEGER NOT NULL,
			cache_misses INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL,
			errors TEXT,
			started_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Document represents a document in the database
type Document struct {
	ID          string     `json:"id"`
	DocType     string     `json:"doc_type"`
	Title       string     `json:"title"`
	FilePath    string     `json:"file_path"`
	Status      string     `json:"status"`
	Version     string     `json:"version"`
	CreatedDate *time.Time `json:"created_date"`
	LastUpdated *time.Time `json:"last_updated"`
	Author      string     `json:"author"`
	FileHash    string     `json:"file_hash"`
	Metadata    string     `json:"metadata_json"`
	IndexedAt   time.Time  `json:"indexed_at"`
}

// Chunk represents a chunk in the database
type Chunk struct {
	ID           int64  `json:"id"`
	DocID        string `json:"doc_id"`
	ChunkIndex   int    `json:"chunk_index"`
	SectionPath  string `json:"section_path"`
	SectionLevel int    `json:"section_level"`
	Content      string `json:"content"`
	ContentHash  string `json:"content_hash"`
	TokenCount   int    `json:"token_count"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	HasCodeBlock bool   `json:"has_code_block"`
	HasTable     bool   `json:"has_table"`
}

// CrossReference represents a cross-reference between documents
type CrossReference struct {
	ID            int64  `json:"id"`
	SourceDocID   string `json:"source_doc_id"`
	TargetDocID   string `json:"target_doc_id"`
	ReferenceType string `json:"reference_type"`
	Context       string `json:"context"`
	LineNumber    int    `json:"line_number"`
}

// SearchResult represents a search result with score
type SearchResult struct {
	Chunk       Chunk    `json:"chunk"`
	DocTitle    string   `json:"doc_title"`
	DocType     string   `json:"doc_type"`
	Score       float32  `json:"score"`
	RelatedDocs []string `json:"related_docs"`
}

// InsertDocument inserts or updates a document
func (s *Store) InsertDocument(doc *Document) error {
	_, err := s.db.Exec(`
		INSERT INTO documents (id, doc_type, title, file_path, status, version, created_date, last_updated, author, file_hash, metadata_json, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			doc_type = excluded.doc_type,
			title = excluded.title,
			file_path = excluded.file_path,
			status = excluded.status,
			version = excluded.version,
			created_date = excluded.created_date,
			last_updated = excluded.last_updated,
			author = excluded.author,
			file_hash = excluded.file_hash,
			metadata_json = excluded.metadata_json,
			indexed_at = CURRENT_TIMESTAMP
	`, doc.ID, doc.DocType, doc.Title, doc.FilePath, doc.Status, doc.Version, doc.CreatedDate, doc.LastUpdated, doc.Author, doc.FileHash, doc.Metadata)

	return err
}

// GetDocument retrieves a document by ID
func (s *Store) GetDocument(id string) (*Document, error) {
	doc := &Document{}
	err := s.db.QueryRow(`
		SELECT id, doc_type, title, file_path, status, version, created_date, last_updated, author, file_hash, metadata_json, indexed_at
		FROM documents WHERE id = ?
	`, id).Scan(&doc.ID, &doc.DocType, &doc.Title, &doc.FilePath, &doc.Status, &doc.Version, &doc.CreatedDate, &doc.LastUpdated, &doc.Author, &doc.FileHash, &doc.Metadata, &doc.IndexedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// GetDocumentByPath retrieves a document by file path
func (s *Store) GetDocumentByPath(path string) (*Document, error) {
	doc := &Document{}
	err := s.db.QueryRow(`
		SELECT id, doc_type, title, file_path, status, version, created_date, last_updated, author, file_hash, metadata_json, indexed_at
		FROM documents WHERE file_path = ?
	`, path).Scan(&doc.ID, &doc.DocType, &doc.Title, &doc.FilePath, &doc.Status, &doc.Version, &doc.CreatedDate, &doc.LastUpdated, &doc.Author, &doc.FileHash, &doc.Metadata, &doc.IndexedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// GetDocumentByHash retrieves a document by file hash
func (s *Store) GetDocumentByHash(hash string) (*Document, error) {
	doc := &Document{}
	err := s.db.QueryRow(`
		SELECT id, doc_type, title, file_path, status, version, created_date, last_updated, author, file_hash, metadata_json, indexed_at
		FROM documents WHERE file_hash = ?
	`, hash).Scan(&doc.ID, &doc.DocType, &doc.Title, &doc.FilePath, &doc.Status, &doc.Version, &doc.CreatedDate, &doc.LastUpdated, &doc.Author, &doc.FileHash, &doc.Metadata, &doc.IndexedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// ListDocuments lists all documents
func (s *Store) ListDocuments() ([]Document, error) {
	rows, err := s.db.Query(`
		SELECT id, doc_type, title, file_path, status, version, created_date, last_updated, author, file_hash, metadata_json, indexed_at
		FROM documents ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	docs := make([]Document, 0)
	for rows.Next() {
		doc := Document{}
		err := rows.Scan(&doc.ID, &doc.DocType, &doc.Title, &doc.FilePath, &doc.Status, &doc.Version, &doc.CreatedDate, &doc.LastUpdated, &doc.Author, &doc.FileHash, &doc.Metadata, &doc.IndexedAt)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// DeleteDocument deletes a document and its related data
// With foreign keys enabled, chunks and chunk_embeddings are deleted via CASCADE
func (s *Store) DeleteDocument(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete cross-references (not CASCADE protected)
	_, err = tx.Exec(`DELETE FROM cross_references WHERE source_doc_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete document (CASCADE will handle chunks and chunk_embeddings)
	_, err = tx.Exec(`DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// InsertChunk inserts a chunk and returns its ID
func (s *Store) InsertChunk(chunk *Chunk) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO chunks (doc_id, chunk_index, section_path, section_level, content, content_hash, token_count, start_line, end_line, has_code_block, has_table)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, chunk.DocID, chunk.ChunkIndex, chunk.SectionPath, chunk.SectionLevel, chunk.Content, chunk.ContentHash, chunk.TokenCount, chunk.StartLine, chunk.EndLine, chunk.HasCodeBlock, chunk.HasTable)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// InsertChunkEmbedding inserts an embedding for a chunk
func (s *Store) InsertChunkEmbedding(chunkID int64, embedding []float32, model string) error {
	embeddingBytes := embeddings.EmbeddingToBytes(embedding)
	_, err := s.db.Exec(`
		INSERT INTO chunk_embeddings (chunk_id, embedding, model)
		VALUES (?, ?, ?)
		ON CONFLICT(chunk_id) DO UPDATE SET embedding = excluded.embedding, model = excluded.model
	`, chunkID, embeddingBytes, model)

	return err
}

// GetChunksByDocID retrieves all chunks for a document
func (s *Store) GetChunksByDocID(docID string) ([]Chunk, error) {
	rows, err := s.db.Query(`
		SELECT id, doc_id, chunk_index, section_path, section_level, content, content_hash, token_count, start_line, end_line, has_code_block, has_table
		FROM chunks WHERE doc_id = ? ORDER BY chunk_index
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chunks := make([]Chunk, 0)
	for rows.Next() {
		chunk := Chunk{}
		err := rows.Scan(&chunk.ID, &chunk.DocID, &chunk.ChunkIndex, &chunk.SectionPath, &chunk.SectionLevel, &chunk.Content, &chunk.ContentHash, &chunk.TokenCount, &chunk.StartLine, &chunk.EndLine, &chunk.HasCodeBlock, &chunk.HasTable)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// DeleteChunksByDocID deletes all chunks for a document
func (s *Store) DeleteChunksByDocID(docID string) error {
	_, err := s.db.Exec(`DELETE FROM chunks WHERE doc_id = ?`, docID)
	return err
}

// InsertCrossReference inserts a cross-reference
func (s *Store) InsertCrossReference(ref *CrossReference) error {
	_, err := s.db.Exec(`
		INSERT INTO cross_references (source_doc_id, target_doc_id, reference_type, context, line_number)
		VALUES (?, ?, ?, ?, ?)
	`, ref.SourceDocID, ref.TargetDocID, ref.ReferenceType, ref.Context, ref.LineNumber)

	return err
}

// GetCrossReferences retrieves all cross-references for a document
func (s *Store) GetCrossReferences(docID string) ([]CrossReference, error) {
	rows, err := s.db.Query(`
		SELECT id, source_doc_id, target_doc_id, reference_type, context, line_number
		FROM cross_references WHERE source_doc_id = ?
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refs := make([]CrossReference, 0)
	for rows.Next() {
		ref := CrossReference{}
		err := rows.Scan(&ref.ID, &ref.SourceDocID, &ref.TargetDocID, &ref.ReferenceType, &ref.Context, &ref.LineNumber)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// GetRelatedDocuments retrieves documents related to a given document
func (s *Store) GetRelatedDocuments(docID string, maxDepth int) ([]string, error) {
	if maxDepth <= 0 {
		maxDepth = 2
	}

	related := make(map[string]bool)
	toProcess := []string{docID}

	for depth := 0; depth < maxDepth && len(toProcess) > 0; depth++ {
		nextToProcess := []string{}

		for _, id := range toProcess {
			// Get outgoing references
			rows, err := s.db.Query(`
				SELECT DISTINCT target_doc_id FROM cross_references WHERE source_doc_id = ?
			`, id)
			if err != nil {
				return nil, err
			}

			for rows.Next() {
				var targetID string
				if err := rows.Scan(&targetID); err != nil {
					rows.Close()
					return nil, err
				}
				if !related[targetID] && targetID != docID {
					related[targetID] = true
					nextToProcess = append(nextToProcess, targetID)
				}
			}
			rows.Close()

			// Get incoming references
			rows, err = s.db.Query(`
				SELECT DISTINCT source_doc_id FROM cross_references WHERE target_doc_id = ?
			`, id)
			if err != nil {
				return nil, err
			}

			for rows.Next() {
				var sourceID string
				if err := rows.Scan(&sourceID); err != nil {
					rows.Close()
					return nil, err
				}
				if !related[sourceID] && sourceID != docID {
					related[sourceID] = true
					nextToProcess = append(nextToProcess, sourceID)
				}
			}
			rows.Close()
		}

		toProcess = nextToProcess
	}

	result := make([]string, 0, len(related))
	for id := range related {
		result = append(result, id)
	}

	return result, nil
}

// SearchSimilar performs vector similarity search
func (s *Store) SearchSimilar(queryEmbedding []float32, topK int, minScore float32, docTypes []string) ([]SearchResult, error) {
	// Build query
	query := `
		SELECT c.id, c.doc_id, c.chunk_index, c.section_path, c.section_level, c.content, c.content_hash, c.token_count, c.start_line, c.end_line, c.has_code_block, c.has_table,
		       d.title, d.doc_type, ce.embedding
		FROM chunks c
		JOIN documents d ON c.doc_id = d.id
		JOIN chunk_embeddings ce ON c.id = ce.chunk_id
	`

	args := make([]interface{}, 0)

	// Add doc type filter
	if len(docTypes) > 0 {
		query += " WHERE d.doc_type IN (?" + repeat(",?", len(docTypes)-1) + ")"
		for _, dt := range docTypes {
			args = append(args, dt)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Calculate similarities and sort
	type scoredResult struct {
		chunk    Chunk
		docTitle string
		docType  string
		score    float32
	}

	results := make([]scoredResult, 0)

	for rows.Next() {
		var chunk Chunk
		var docTitle, docType string
		var embeddingBytes []byte

		err := rows.Scan(&chunk.ID, &chunk.DocID, &chunk.ChunkIndex, &chunk.SectionPath, &chunk.SectionLevel, &chunk.Content, &chunk.ContentHash, &chunk.TokenCount, &chunk.StartLine, &chunk.EndLine, &chunk.HasCodeBlock, &chunk.HasTable, &docTitle, &docType, &embeddingBytes)
		if err != nil {
			return nil, err
		}

		// Calculate cosine similarity
		embedding := embeddings.BytesToEmbedding(embeddingBytes)
		score := embeddings.CosineSimilarity(queryEmbedding, embedding)

		if score >= minScore {
			results = append(results, scoredResult{
				chunk:    chunk,
				docTitle: docTitle,
				docType:  docType,
				score:    score,
			})
		}
	}

	// Sort by score descending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit to topK
	if len(results) > topK {
		results = results[:topK]
	}

	// Convert to SearchResult with related docs
	finalResults := make([]SearchResult, len(results))
	for i, r := range results {
		relatedDocs, err := s.GetRelatedDocuments(r.chunk.DocID, 1)
		if err != nil {
			relatedDocs = []string{}
		}

		finalResults[i] = SearchResult{
			Chunk:       r.chunk,
			DocTitle:    r.docTitle,
			DocType:     r.docType,
			Score:       r.score,
			RelatedDocs: relatedDocs,
		}
	}

	return finalResults, nil
}

// GetStats returns database statistics
func (s *Store) GetStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count documents
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["documents"] = count

	// Count chunks
	err = s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["chunks"] = count

	// Count embeddings
	err = s.db.QueryRow(`SELECT COUNT(*) FROM chunk_embeddings`).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["embeddings"] = count

	// Count cross-references
	err = s.db.QueryRow(`SELECT COUNT(*) FROM cross_references`).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["cross_references"] = count

	// Count cache entries
	err = s.db.QueryRow(`SELECT COUNT(*) FROM embedding_cache`).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["cache_entries"] = count

	return stats, nil
}

// SaveIndexingStats saves indexing operation statistics
func (s *Store) SaveIndexingStats(operation string, docsProcessed, chunksCreated, embeddingsGenerated, cacheHits, cacheMisses int, durationMs int64, errors string, startedAt time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO indexing_stats (operation, docs_processed, chunks_created, embeddings_generated, cache_hits, cache_misses, duration_ms, errors, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, operation, docsProcessed, chunksCreated, embeddingsGenerated, cacheHits, cacheMisses, durationMs, errors, startedAt)

	return err
}

// Helper function
func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// GetStatsByDocType returns document counts grouped by type
func (s *Store) GetStatsByDocType() (map[string]int64, error) {
	rows, err := s.db.Query(`
		SELECT doc_type, COUNT(*) as count
		FROM documents
		GROUP BY doc_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var docType string
		var count int64
		if err := rows.Scan(&docType, &count); err != nil {
			return nil, err
		}
		result[docType] = count
	}

	return result, nil
}

// GetLastIndexedTime returns the most recent indexing timestamp
func (s *Store) GetLastIndexedTime() (*time.Time, error) {
	var t time.Time
	err := s.db.QueryRow(`
		SELECT MAX(indexed_at) FROM documents
	`).Scan(&t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListDocumentsPaginated returns documents with pagination and filters
func (s *Store) ListDocumentsPaginated(page, limit int, docType, status string) ([]Document, int64, error) {
	// Build query
	baseQuery := `FROM documents WHERE 1=1`
	args := []interface{}{}

	if docType != "" {
		baseQuery += ` AND doc_type = ?`
		args = append(args, docType)
	}
	if status != "" {
		baseQuery += ` AND status = ?`
		args = append(args, status)
	}

	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) ` + baseQuery
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get documents
	offset := (page - 1) * limit
	selectQuery := `SELECT id, doc_type, title, file_path, status, version, created_date, last_updated, author, file_hash, metadata_json, indexed_at ` + baseQuery + ` ORDER BY indexed_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	docs := make([]Document, 0)
	for rows.Next() {
		var doc Document
		var createdDate, lastUpdated sql.NullTime
		var status, version, author, metadata sql.NullString

		err := rows.Scan(
			&doc.ID, &doc.DocType, &doc.Title, &doc.FilePath,
			&status, &version, &createdDate, &lastUpdated,
			&author, &doc.FileHash, &metadata, &doc.IndexedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if status.Valid {
			doc.Status = status.String
		}
		if version.Valid {
			doc.Version = version.String
		}
		if author.Valid {
			doc.Author = author.String
		}
		if metadata.Valid {
			doc.Metadata = metadata.String
		}
		if createdDate.Valid {
			doc.CreatedDate = &createdDate.Time
		}
		if lastUpdated.Valid {
			doc.LastUpdated = &lastUpdated.Time
		}

		docs = append(docs, doc)
	}

	return docs, total, nil
}

// GetChunkCountByDocID returns the number of chunks for a document
func (s *Store) GetChunkCountByDocID(docID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE doc_id = ?`, docID).Scan(&count)
	return count, err
}
