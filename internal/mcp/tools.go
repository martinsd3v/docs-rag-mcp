package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/trxio/docs-rag-mcp/internal/embeddings"
	"github.com/trxio/docs-rag-mcp/internal/vector"
)

// ToolsHandler manages MCP tools and their execution
type ToolsHandler struct {
	store           *vector.Store
	embeddingClient embeddings.EmbeddingClient
}

// NewToolsHandler creates a new tools handler
func NewToolsHandler(store *vector.Store, embClient embeddings.EmbeddingClient) *ToolsHandler {
	return &ToolsHandler{
		store:           store,
		embeddingClient: embClient,
	}
}

// RegisterTools registers all tools with the MCP server
func (h *ToolsHandler) RegisterTools(server *Server) {
	// Tool 1: search_docs
	server.RegisterTool(Tool{
		Name:        "search_docs",
		Description: "Search through indexed documentation (RFCs, ADRs, BDRs, Guidelines) using semantic search. Returns relevant chunks with similarity scores.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"query": {
					Type:        "string",
					Description: "The search query - describe what you're looking for",
				},
				"doc_types": {
					Type:        "array",
					Description: "Filter by document types (RFC, ADR, BDR, Guideline, Roadmap). Empty means all types.",
					Items:       &Items{Type: "string"},
				},
				"top_k": {
					Type:        "integer",
					Description: "Number of results to return (default: 5, max: 20)",
					Default:     5,
				},
				"min_score": {
					Type:        "number",
					Description: "Minimum similarity score 0-1 (default: 0.3)",
					Default:     0.3,
				},
			},
			Required: []string{"query"},
		},
	}, h.searchDocs)

	// Tool 2: get_relevant_context
	server.RegisterTool(Tool{
		Name:        "get_relevant_context",
		Description: "Get relevant documentation context for a specific task or implementation. Combines search results with related documents.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"task": {
					Type:        "string",
					Description: "Description of the task you're working on",
				},
				"max_chunks": {
					Type:        "integer",
					Description: "Maximum number of chunks to return (default: 10)",
					Default:     10,
				},
				"expand_graph": {
					Type:        "boolean",
					Description: "Include related documents from the reference graph (default: true)",
					Default:     true,
				},
			},
			Required: []string{"task"},
		},
	}, h.getRelevantContext)

	// Tool 3: get_document
	server.RegisterTool(Tool{
		Name:        "get_document",
		Description: "Get a specific document by ID with its metadata and optionally its chunks.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"doc_id": {
					Type:        "string",
					Description: "Document ID (e.g., RFC-001, ADR-001)",
				},
				"include_chunks": {
					Type:        "boolean",
					Description: "Include document chunks (default: false)",
					Default:     false,
				},
				"include_related": {
					Type:        "boolean",
					Description: "Include related document IDs (default: true)",
					Default:     true,
				},
			},
			Required: []string{"doc_id"},
		},
	}, h.getDocument)

	// Tool 4: search_by_section
	server.RegisterTool(Tool{
		Name:        "search_by_section",
		Description: "Search within specific sections of documents (e.g., 'Implementation', 'Examples', 'Guidelines').",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"query": {
					Type:        "string",
					Description: "The search query",
				},
				"section_names": {
					Type:        "array",
					Description: "Section names to search in (e.g., ['Implementation', 'Examples'])",
					Items:       &Items{Type: "string"},
				},
				"doc_types": {
					Type:        "array",
					Description: "Filter by document types",
					Items:       &Items{Type: "string"},
				},
				"top_k": {
					Type:        "integer",
					Description: "Number of results (default: 5)",
					Default:     5,
				},
			},
			Required: []string{"query"},
		},
	}, h.searchBySection)

	// Tool 5: find_related_docs
	server.RegisterTool(Tool{
		Name:        "find_related_docs",
		Description: "Find documents related to a given document through cross-references and the document graph.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"doc_id": {
					Type:        "string",
					Description: "Document ID to find relations for",
				},
				"max_depth": {
					Type:        "integer",
					Description: "Maximum depth to traverse in the graph (default: 2)",
					Default:     2,
				},
				"include_incoming": {
					Type:        "boolean",
					Description: "Include documents that reference this one (default: true)",
					Default:     true,
				},
			},
			Required: []string{"doc_id"},
		},
	}, h.findRelatedDocs)
}

// Tool implementations

// SearchDocsArgs represents arguments for search_docs
type SearchDocsArgs struct {
	Query    string   `json:"query"`
	DocTypes []string `json:"doc_types"`
	TopK     int      `json:"top_k"`
	MinScore float32  `json:"min_score"`
}

func (h *ToolsHandler) searchDocs(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var params SearchDocsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Defaults
	if params.TopK <= 0 {
		params.TopK = 5
	}
	if params.TopK > 20 {
		params.TopK = 20
	}
	if params.MinScore <= 0 {
		params.MinScore = 0.3
	}

	start := time.Now()

	// Generate query embedding
	embResult, err := h.embeddingClient.Embed(ctx, params.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search
	results, err := h.store.SearchSimilar(embResult.Embedding, params.TopK, params.MinScore, params.DocTypes)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d results for: \"%s\"\n\n", len(results), params.Query))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("## %d. %s: %s (score: %.2f)\n", i+1, r.DocType, r.DocTitle, r.Score))
		sb.WriteString(fmt.Sprintf("**Section:** %s\n", r.Chunk.SectionPath))

		if len(r.RelatedDocs) > 0 {
			sb.WriteString(fmt.Sprintf("**Related:** %s\n", strings.Join(r.RelatedDocs, ", ")))
		}

		sb.WriteString("\n```\n")
		content := r.Chunk.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		sb.WriteString(content)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString(fmt.Sprintf("---\n*Search took %dms*\n", time.Since(start).Milliseconds()))

	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
	}, nil
}

// GetRelevantContextArgs represents arguments for get_relevant_context
type GetRelevantContextArgs struct {
	Task        string `json:"task"`
	MaxChunks   int    `json:"max_chunks"`
	ExpandGraph bool   `json:"expand_graph"`
}

func (h *ToolsHandler) getRelevantContext(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var params GetRelevantContextArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Defaults
	if params.MaxChunks <= 0 {
		params.MaxChunks = 10
	}

	// Generate query embedding
	embResult, err := h.embeddingClient.Embed(ctx, params.Task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Initial search
	results, err := h.store.SearchSimilar(embResult.Embedding, params.MaxChunks/2, 0.25, nil)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Expand with related documents if requested
	relatedDocs := make(map[string]bool)
	if params.ExpandGraph {
		for _, r := range results {
			for _, relID := range r.RelatedDocs {
				relatedDocs[relID] = true
			}
		}
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Relevant Context for: %s\n\n", params.Task))

	// Primary results
	sb.WriteString("## Primary Documents\n\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("### %s: %s\n", r.DocType, r.DocTitle))
		sb.WriteString(fmt.Sprintf("*Relevance: %.0f%%*\n\n", r.Score*100))
		sb.WriteString(r.Chunk.Content)
		sb.WriteString("\n\n---\n\n")
	}

	// Related documents
	if len(relatedDocs) > 0 {
		sb.WriteString("## Related Documents\n\n")
		sb.WriteString("The following documents are referenced by the primary results:\n\n")
		for docID := range relatedDocs {
			doc, err := h.store.GetDocument(docID)
			if err == nil && doc != nil {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", doc.ID, doc.Title))
			}
		}
		sb.WriteString("\n")
	}

	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
	}, nil
}

// GetDocumentArgs represents arguments for get_document
type GetDocumentArgs struct {
	DocID          string `json:"doc_id"`
	IncludeChunks  bool   `json:"include_chunks"`
	IncludeRelated bool   `json:"include_related"`
}

func (h *ToolsHandler) getDocument(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var params GetDocumentArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Get document
	doc, err := h.store.GetDocument(params.DocID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	if doc == nil {
		return &ToolCallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Document not found: %s", params.DocID)}},
			IsError: true,
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s: %s\n\n", doc.ID, doc.Title))
	sb.WriteString(fmt.Sprintf("**Type:** %s\n", doc.DocType))
	if doc.Status != "" {
		sb.WriteString(fmt.Sprintf("**Status:** %s\n", doc.Status))
	}
	if doc.Version != "" {
		sb.WriteString(fmt.Sprintf("**Version:** %s\n", doc.Version))
	}
	if doc.Author != "" {
		sb.WriteString(fmt.Sprintf("**Author:** %s\n", doc.Author))
	}
	sb.WriteString(fmt.Sprintf("**File:** %s\n", doc.FilePath))
	sb.WriteString("\n")

	// Include related documents
	if params.IncludeRelated {
		related, err := h.store.GetRelatedDocuments(params.DocID, 2)
		if err == nil && len(related) > 0 {
			sb.WriteString("## Related Documents\n\n")
			for _, relID := range related {
				relDoc, _ := h.store.GetDocument(relID)
				if relDoc != nil {
					sb.WriteString(fmt.Sprintf("- **%s**: %s\n", relDoc.ID, relDoc.Title))
				} else {
					sb.WriteString(fmt.Sprintf("- %s\n", relID))
				}
			}
			sb.WriteString("\n")
		}
	}

	// Include chunks
	if params.IncludeChunks {
		chunks, err := h.store.GetChunksByDocID(params.DocID)
		if err == nil && len(chunks) > 0 {
			sb.WriteString("## Content Chunks\n\n")
			for i, chunk := range chunks {
				sb.WriteString(fmt.Sprintf("### Chunk %d: %s\n", i+1, chunk.SectionPath))
				sb.WriteString(fmt.Sprintf("*Lines %d-%d, %d tokens*\n\n", chunk.StartLine, chunk.EndLine, chunk.TokenCount))
				sb.WriteString(chunk.Content)
				sb.WriteString("\n\n---\n\n")
			}
		}
	}

	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
	}, nil
}

// SearchBySectionArgs represents arguments for search_by_section
type SearchBySectionArgs struct {
	Query        string   `json:"query"`
	SectionNames []string `json:"section_names"`
	DocTypes     []string `json:"doc_types"`
	TopK         int      `json:"top_k"`
}

func (h *ToolsHandler) searchBySection(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var params SearchBySectionArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Defaults
	if params.TopK <= 0 {
		params.TopK = 5
	}

	// Generate query embedding
	embResult, err := h.embeddingClient.Embed(ctx, params.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Get more results and filter by section
	results, err := h.store.SearchSimilar(embResult.Embedding, params.TopK*3, 0.2, params.DocTypes)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Filter by section names if provided
	filtered := make([]vector.SearchResult, 0)
	if len(params.SectionNames) > 0 {
		for _, r := range results {
			for _, sectionName := range params.SectionNames {
				if strings.Contains(strings.ToLower(r.Chunk.SectionPath), strings.ToLower(sectionName)) {
					filtered = append(filtered, r)
					break
				}
			}
		}
	} else {
		filtered = results
	}

	// Limit to topK
	if len(filtered) > params.TopK {
		filtered = filtered[:params.TopK]
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d results in sections: %v\n\n", len(filtered), params.SectionNames))

	for i, r := range filtered {
		sb.WriteString(fmt.Sprintf("## %d. %s: %s (score: %.2f)\n", i+1, r.DocType, r.DocTitle, r.Score))
		sb.WriteString(fmt.Sprintf("**Section:** %s\n\n", r.Chunk.SectionPath))
		sb.WriteString(r.Chunk.Content)
		sb.WriteString("\n\n---\n\n")
	}

	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
	}, nil
}

// FindRelatedDocsArgs represents arguments for find_related_docs
type FindRelatedDocsArgs struct {
	DocID           string `json:"doc_id"`
	MaxDepth        int    `json:"max_depth"`
	IncludeIncoming bool   `json:"include_incoming"`
}

func (h *ToolsHandler) findRelatedDocs(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var params FindRelatedDocsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Defaults
	if params.MaxDepth <= 0 {
		params.MaxDepth = 2
	}

	// Get source document
	doc, err := h.store.GetDocument(params.DocID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	if doc == nil {
		return &ToolCallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Document not found: %s", params.DocID)}},
			IsError: true,
		}, nil
	}

	// Get related documents
	related, err := h.store.GetRelatedDocuments(params.DocID, params.MaxDepth)
	if err != nil {
		return nil, fmt.Errorf("failed to get related documents: %w", err)
	}

	// Get cross-references
	refs, err := h.store.GetCrossReferences(params.DocID)
	if err != nil {
		refs = []vector.CrossReference{}
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Related Documents for %s\n\n", params.DocID))
	sb.WriteString(fmt.Sprintf("**Source:** %s: %s\n\n", doc.ID, doc.Title))

	// Direct references (outgoing)
	if len(refs) > 0 {
		sb.WriteString("## Direct References\n\n")
		for _, ref := range refs {
			refDoc, _ := h.store.GetDocument(ref.TargetDocID)
			if refDoc != nil {
				sb.WriteString(fmt.Sprintf("- **%s**: %s (%s)\n", ref.TargetDocID, refDoc.Title, ref.ReferenceType))
			} else {
				sb.WriteString(fmt.Sprintf("- **%s** (%s)\n", ref.TargetDocID, ref.ReferenceType))
			}
		}
		sb.WriteString("\n")
	}

	// All related (including through graph)
	if len(related) > 0 {
		sb.WriteString(fmt.Sprintf("## All Related (depth %d)\n\n", params.MaxDepth))
		for _, relID := range related {
			relDoc, _ := h.store.GetDocument(relID)
			if relDoc != nil {
				sb.WriteString(fmt.Sprintf("- **%s**: %s (%s)\n", relDoc.ID, relDoc.Title, relDoc.DocType))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", relID))
			}
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("No related documents found.\n")
	}

	return &ToolCallResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
	}, nil
}
