-- docs-rag-mcp Database Schema
-- SQLite 3 with sqlite-vec extension

-- Enable WAL mode for better concurrency
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;  -- 64MB
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 268435456; -- 256MB

-- ============================================================================
-- Documents Table
-- Stores metadata for each document (RFC, ADR, BDR, Guideline, Roadmap)
-- ============================================================================
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,                    -- RFC-001, ADR-001, etc
    doc_type TEXT NOT NULL,                 -- RFC, ADR, BDR, Guideline, Roadmap
    title TEXT NOT NULL,                    -- Document title
    file_path TEXT NOT NULL UNIQUE,         -- Absolute path to file
    status TEXT,                            -- Draft, Approved, Deprecated, etc
    version TEXT,                           -- v1.0, v2.1, etc
    created_date TIMESTAMP,                 -- When document was created
    last_updated TIMESTAMP,                 -- When document was last modified
    author TEXT,                            -- Document author
    file_hash TEXT NOT NULL,                -- SHA256 hash for cache invalidation
    metadata_json TEXT,                     -- JSON with extra metadata
    indexed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CHECK (doc_type IN ('RFC', 'ADR', 'BDR', 'Guideline', 'Roadmap'))
);

-- Index for common queries
CREATE INDEX IF NOT EXISTS idx_documents_type ON documents(doc_type);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_hash ON documents(file_hash);
CREATE INDEX IF NOT EXISTS idx_documents_indexed_at ON documents(indexed_at DESC);

-- ============================================================================
-- Chunks Table
-- Stores semantic chunks of documents with metadata
-- ============================================================================
CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    doc_id TEXT NOT NULL,                   -- Foreign key to documents.id
    chunk_index INTEGER NOT NULL,           -- Order within document (0, 1, 2, ...)
    section_path TEXT,                      -- "Executive Summary > Problem & Motivation"
    section_level INTEGER,                  -- 1 (H1), 2 (H2), 3 (H3)
    content TEXT NOT NULL,                  -- Chunk content with overlap
    content_hash TEXT NOT NULL,             -- SHA256 for deduplication
    token_count INTEGER NOT NULL,           -- Approximate token count
    start_line INTEGER,                     -- Starting line in original file
    end_line INTEGER,                       -- Ending line in original file
    has_code_block BOOLEAN DEFAULT 0,       -- Contains code blocks
    has_table BOOLEAN DEFAULT 0,            -- Contains tables
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (doc_id) REFERENCES documents(id) ON DELETE CASCADE
);

-- Indexes for chunk queries
CREATE INDEX IF NOT EXISTS idx_chunks_doc_id ON chunks(doc_id);
CREATE INDEX IF NOT EXISTS idx_chunks_doc_chunk ON chunks(doc_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_chunks_content_hash ON chunks(content_hash);
CREATE INDEX IF NOT EXISTS idx_chunks_section_level ON chunks(section_level);
CREATE INDEX IF NOT EXISTS idx_chunks_has_code ON chunks(has_code_block) WHERE has_code_block = 1;
CREATE INDEX IF NOT EXISTS idx_chunks_has_table ON chunks(has_table) WHERE has_table = 1;

-- ============================================================================
-- Chunk Embeddings Table (sqlite-vec)
-- Stores vector embeddings for semantic search
-- ============================================================================
-- Note: This table will be created using sqlite-vec extension
-- The actual creation will be done in Go code using:
-- CREATE VIRTUAL TABLE chunk_embeddings USING vec0(
--     chunk_id INTEGER PRIMARY KEY,
--     embedding FLOAT[1536]
-- );

-- For now, we'll use a regular table as placeholder
-- This will be replaced when sqlite-vec is integrated
CREATE TABLE IF NOT EXISTS chunk_embeddings (
    chunk_id INTEGER PRIMARY KEY,
    embedding BLOB NOT NULL,                -- Serialized float32 array (1536 dimensions)
    model TEXT NOT NULL DEFAULT 'text-embedding-3-small',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
);

-- ============================================================================
-- Cross References Table
-- Stores document relationships and references
-- ============================================================================
CREATE TABLE IF NOT EXISTS cross_references (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_doc_id TEXT NOT NULL,            -- Document containing the reference
    target_doc_id TEXT NOT NULL,            -- Document being referenced
    reference_type TEXT NOT NULL,           -- link, related_rfcs, see_also, mention
    context TEXT,                           -- Text around the reference
    line_number INTEGER,                    -- Line number in source document
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_doc_id) REFERENCES documents(id) ON DELETE CASCADE,
    FOREIGN KEY (target_doc_id) REFERENCES documents(id) ON DELETE CASCADE,
    CHECK (reference_type IN ('link', 'related_rfcs', 'see_also', 'mention'))
);

-- Indexes for graph queries
CREATE INDEX IF NOT EXISTS idx_cross_refs_source ON cross_references(source_doc_id);
CREATE INDEX IF NOT EXISTS idx_cross_refs_target ON cross_references(target_doc_id);
CREATE INDEX IF NOT EXISTS idx_cross_refs_type ON cross_references(reference_type);
CREATE INDEX IF NOT EXISTS idx_cross_refs_both ON cross_references(source_doc_id, target_doc_id);

-- ============================================================================
-- Embedding Cache Table
-- Caches embeddings to avoid re-generating for unchanged content
-- ============================================================================
CREATE TABLE IF NOT EXISTS embedding_cache (
    content_hash TEXT PRIMARY KEY,          -- SHA256 of content
    embedding BLOB NOT NULL,                -- Serialized float32 array
    model TEXT NOT NULL,                    -- OpenAI model used
    token_count INTEGER NOT NULL,           -- Number of tokens in content
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    use_count INTEGER DEFAULT 1             -- How many times this cache was hit
);

-- Index for cache cleanup (LRU)
CREATE INDEX IF NOT EXISTS idx_cache_last_used ON embedding_cache(last_used);
CREATE INDEX IF NOT EXISTS idx_cache_model ON embedding_cache(model);

-- ============================================================================
-- Indexing Stats Table
-- Tracks indexing operations for monitoring and debugging
-- ============================================================================
CREATE TABLE IF NOT EXISTS indexing_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation TEXT NOT NULL,                -- index, reindex, update
    docs_processed INTEGER NOT NULL,
    chunks_created INTEGER NOT NULL,
    embeddings_generated INTEGER NOT NULL,
    cache_hits INTEGER NOT NULL,
    cache_misses INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    errors TEXT,                            -- JSON array of errors
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_stats_completed ON indexing_stats(completed_at DESC);

-- ============================================================================
-- Views for Common Queries
-- ============================================================================

-- View: Document with chunk count
CREATE VIEW IF NOT EXISTS v_documents_with_stats AS
SELECT
    d.*,
    COUNT(c.id) as chunk_count,
    SUM(c.token_count) as total_tokens,
    COUNT(CASE WHEN c.has_code_block = 1 THEN 1 END) as chunks_with_code,
    COUNT(CASE WHEN c.has_table = 1 THEN 1 END) as chunks_with_tables
FROM documents d
LEFT JOIN chunks c ON d.id = c.doc_id
GROUP BY d.id;

-- View: Cross-reference graph statistics
CREATE VIEW IF NOT EXISTS v_document_graph_stats AS
SELECT
    d.id,
    d.title,
    d.doc_type,
    COUNT(DISTINCT cr_out.target_doc_id) as outgoing_refs,
    COUNT(DISTINCT cr_in.source_doc_id) as incoming_refs,
    COUNT(DISTINCT cr_out.target_doc_id) + COUNT(DISTINCT cr_in.source_doc_id) as total_connections
FROM documents d
LEFT JOIN cross_references cr_out ON d.id = cr_out.source_doc_id
LEFT JOIN cross_references cr_in ON d.id = cr_in.target_doc_id
GROUP BY d.id;

-- View: Recent indexing activity
CREATE VIEW IF NOT EXISTS v_recent_indexing AS
SELECT
    operation,
    docs_processed,
    chunks_created,
    embeddings_generated,
    cache_hits,
    cache_misses,
    ROUND(cache_hits * 100.0 / NULLIF(cache_hits + cache_misses, 0), 2) as cache_hit_rate,
    duration_ms,
    datetime(started_at) as started,
    datetime(completed_at) as completed
FROM indexing_stats
ORDER BY completed_at DESC
LIMIT 10;

-- ============================================================================
-- Triggers
-- ============================================================================

-- Trigger: Update embedding_cache.last_used when cache is hit
CREATE TRIGGER IF NOT EXISTS tr_cache_update_last_used
AFTER SELECT ON embedding_cache
BEGIN
    UPDATE embedding_cache
    SET last_used = CURRENT_TIMESTAMP,
        use_count = use_count + 1
    WHERE content_hash = NEW.content_hash;
END;

-- ============================================================================
-- Initial Data / Metadata
-- ============================================================================

-- Store schema version for migrations
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT INTO schema_version (version, description)
VALUES (1, 'Initial schema with documents, chunks, embeddings, cross_references, and cache');

-- ============================================================================
-- Cleanup Functions (will be called periodically via Go)
-- ============================================================================

-- Clean old cache entries (keep last 30 days or top 10K by use)
-- This will be executed from Go code:
-- DELETE FROM embedding_cache
-- WHERE last_used < datetime('now', '-30 days')
-- AND content_hash NOT IN (
--     SELECT content_hash FROM embedding_cache
--     ORDER BY use_count DESC LIMIT 10000
-- );
