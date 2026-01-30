package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SSESession represents an active SSE connection
type SSESession struct {
	ID        string
	Writer    http.ResponseWriter
	Flusher   http.Flusher
	Created   time.Time
	Messages  chan *Response
	Done      chan struct{}
	Ctx       context.Context
	CancelFn  context.CancelFunc
}

// MCPHTTPHandler manages MCP connections over HTTP/SSE
type MCPHTTPHandler struct {
	toolsHandler *ToolsHandler
	sessions     map[string]*SSESession
	mu           sync.RWMutex
	log          func(format string, args ...interface{})

	// MCP server components
	tools    map[string]Tool
	handlers map[string]ToolHandler
}

// NewMCPHTTPHandler creates a new HTTP/SSE handler for MCP
func NewMCPHTTPHandler(toolsHandler *ToolsHandler) *MCPHTTPHandler {
	h := &MCPHTTPHandler{
		toolsHandler: toolsHandler,
		sessions:     make(map[string]*SSESession),
		tools:        make(map[string]Tool),
		handlers:     make(map[string]ToolHandler),
		log: func(format string, args ...interface{}) {
			fmt.Printf("[MCP-HTTP] "+format+"\n", args...)
		},
	}

	// Register tools from the toolsHandler
	h.registerTools()

	return h
}

// SetLogger sets a custom logger
func (h *MCPHTTPHandler) SetLogger(log func(format string, args ...interface{})) {
	h.log = log
}

// registerTools registers all tools (mirrors ToolsHandler.RegisterTools logic)
func (h *MCPHTTPHandler) registerTools() {
	// Tool 1: search_docs
	h.RegisterTool(Tool{
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
	}, h.toolsHandler.SearchDocs)

	// Tool 2: get_relevant_context
	h.RegisterTool(Tool{
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
	}, h.toolsHandler.GetRelevantContext)

	// Tool 3: get_document
	h.RegisterTool(Tool{
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
	}, h.toolsHandler.GetDocument)

	// Tool 4: search_by_section
	h.RegisterTool(Tool{
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
	}, h.toolsHandler.SearchBySection)

	// Tool 5: find_related_docs
	h.RegisterTool(Tool{
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
	}, h.toolsHandler.FindRelatedDocs)
}

// RegisterTool registers a tool with its handler
func (h *MCPHTTPHandler) RegisterTool(tool Tool, handler ToolHandler) {
	h.tools[tool.Name] = tool
	h.handlers[tool.Name] = handler
}

// HandleSSE handles GET /mcp/sse - establishes SSE connection
func (h *MCPHTTPHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Check for SSE support
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionID := uuid.New().String()
	ctx, cancel := context.WithCancel(r.Context())

	session := &SSESession{
		ID:       sessionID,
		Writer:   w,
		Flusher:  flusher,
		Created:  time.Now(),
		Messages: make(chan *Response, 100),
		Done:     make(chan struct{}),
		Ctx:      ctx,
		CancelFn: cancel,
	}

	h.mu.Lock()
	h.sessions[sessionID] = session
	h.mu.Unlock()

	h.log("New SSE session: %s", sessionID)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Send endpoint event with the message URL (as per MCP spec)
	// The client needs to know where to POST messages
	messageURL := fmt.Sprintf("/mcp/message?sessionId=%s", sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", messageURL)
	flusher.Flush()

	// Keep connection alive and send messages
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			h.log("SSE session closed by client: %s", sessionID)
			h.removeSession(sessionID)
			return

		case <-session.Done:
			h.log("SSE session terminated: %s", sessionID)
			return

		case msg := <-session.Messages:
			data, err := json.Marshal(msg)
			if err != nil {
				h.log("Error marshaling message: %v", err)
				continue
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			flusher.Flush()

		case <-ticker.C:
			// Send keepalive comment
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// HandleMessage handles POST /mcp/message - receives JSON-RPC messages
func (h *MCPHTTPHandler) HandleMessage(w http.ResponseWriter, r *http.Request) {
	// Get session ID from query param
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId query parameter required", http.StatusBadRequest)
		return
	}

	// Get session
	h.mu.RLock()
	session, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Parse request
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorToSession(session, nil, ParseError, "Parse error", err.Error())
		w.WriteHeader(http.StatusAccepted)
		return
	}

	h.log("Received message: %s (session: %s)", req.Method, sessionID)

	// Handle request asynchronously, send response via SSE
	go h.handleRequest(session, &req)

	// Return 202 Accepted immediately
	w.WriteHeader(http.StatusAccepted)
}

// handleRequest processes a JSON-RPC request and sends response via SSE
func (h *MCPHTTPHandler) handleRequest(session *SSESession, req *Request) {
	h.log("Processing: %s", req.Method)

	switch req.Method {
	case "initialize":
		h.handleInitialize(session, req)
	case "initialized":
		// Notification, no response needed
		h.log("Client initialized")
	case "tools/list":
		h.handleToolsList(session, req)
	case "tools/call":
		h.handleToolCall(session, req)
	case "ping":
		h.sendResultToSession(session, req.ID, map[string]interface{}{})
	default:
		h.sendErrorToSession(session, req.ID, MethodNotFound, "Method not found", req.Method)
	}
}

// handleInitialize handles the initialize request
func (h *MCPHTTPHandler) handleInitialize(session *SSESession, req *Request) {
	var params InitializeParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			h.sendErrorToSession(session, req.ID, InvalidParams, "Invalid params", err.Error())
			return
		}
	}

	h.log("Client: %s %s", params.ClientInfo.Name, params.ClientInfo.Version)

	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    ServerName,
			Version: ServerVersion,
		},
	}

	h.sendResultToSession(session, req.ID, result)
}

// handleToolsList handles the tools/list request
func (h *MCPHTTPHandler) handleToolsList(session *SSESession, req *Request) {
	tools := make([]Tool, 0, len(h.tools))
	for _, tool := range h.tools {
		tools = append(tools, tool)
	}

	result := ToolsListResult{
		Tools: tools,
	}

	h.sendResultToSession(session, req.ID, result)
}

// handleToolCall handles the tools/call request
func (h *MCPHTTPHandler) handleToolCall(session *SSESession, req *Request) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		h.sendErrorToSession(session, req.ID, InvalidParams, "Invalid params", err.Error())
		return
	}

	handler, ok := h.handlers[params.Name]
	if !ok {
		h.sendErrorToSession(session, req.ID, InvalidParams, "Unknown tool", params.Name)
		return
	}

	h.log("Calling tool: %s", params.Name)

	result, err := handler(session.Ctx, params.Arguments)
	if err != nil {
		h.sendResultToSession(session, req.ID, &ToolCallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		})
		return
	}

	h.sendResultToSession(session, req.ID, result)
}

// sendResultToSession sends a successful response via SSE
func (h *MCPHTTPHandler) sendResultToSession(session *SSESession, id interface{}, result interface{}) {
	resp := &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	select {
	case session.Messages <- resp:
	case <-session.Ctx.Done():
		h.log("Session closed, cannot send result")
	default:
		h.log("Message buffer full, dropping message")
	}
}

// sendErrorToSession sends an error response via SSE
func (h *MCPHTTPHandler) sendErrorToSession(session *SSESession, id interface{}, code int, message string, data interface{}) {
	resp := &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	select {
	case session.Messages <- resp:
	case <-session.Ctx.Done():
		h.log("Session closed, cannot send error")
	default:
		h.log("Message buffer full, dropping error")
	}
}

// removeSession removes a session from the handler
func (h *MCPHTTPHandler) removeSession(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if session, ok := h.sessions[sessionID]; ok {
		session.CancelFn()
		close(session.Done)
		delete(h.sessions, sessionID)
	}
}

// Close closes all sessions and cleans up
func (h *MCPHTTPHandler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, session := range h.sessions {
		session.CancelFn()
		close(session.Done)
		delete(h.sessions, id)
	}
}

// ActiveSessions returns the number of active sessions
func (h *MCPHTTPHandler) ActiveSessions() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}
