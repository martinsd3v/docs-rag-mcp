package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/trxio/docs-rag-mcp/internal/api/models"
	"github.com/trxio/docs-rag-mcp/internal/parser"
	"github.com/trxio/docs-rag-mcp/internal/vector"
)

// handleUpload processes multiple .md file uploads
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.writeError(w, http.StatusBadRequest, "files too large or invalid form", err)
		return
	}

	// Get all uploaded files
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		s.writeError(w, http.StatusBadRequest, "no files uploaded", nil)
		return
	}

	// Process each file
	results := make([]models.UploadResult, 0, len(files))
	summary := models.UploadSummary{
		Total: len(files),
	}

	for _, fileHeader := range files {
		result := s.processUploadedFile(r, fileHeader)
		results = append(results, result)

		switch result.Status {
		case "success":
			summary.Succeeded++
		case "replaced":
			summary.Replaced++
			summary.Succeeded++
		case "error":
			summary.Failed++
		}
	}

	resp := models.UploadResponse{
		Results: results,
		Summary: summary,
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// processUploadedFile processes a single uploaded file
func (s *Server) processUploadedFile(r *http.Request, fh *multipart.FileHeader) models.UploadResult {
	result := models.UploadResult{
		Filename: fh.Filename,
		Status:   "error",
	}

	// Open file
	file, err := fh.Open()
	if err != nil {
		result.Error = "failed to open file: " + err.Error()
		return result
	}
	defer file.Close()

	// Read content
	content, err := io.ReadAll(file)
	if err != nil {
		result.Error = "failed to read file: " + err.Error()
		return result
	}

	// Parse document
	p := parser.NewParser()
	doc, err := p.ParseFile(fh.Filename, content)
	if err != nil {
		result.Error = "failed to parse markdown: " + err.Error()
		return result
	}

	result.DocID = doc.ID

	// Get chunker config from active project
	chunkerConfig := parser.DefaultChunkerConfig()
	s.mu.RLock()
	if s.activeProject != nil && s.activeProject.ChunkConfig != nil {
		chunkerConfig = &parser.ChunkerConfig{
			MinSize:           s.activeProject.ChunkConfig.MinSize,
			MaxSize:           s.activeProject.ChunkConfig.MaxSize,
			Overlap:           s.activeProject.ChunkConfig.Overlap,
			RespectBoundaries: s.activeProject.ChunkConfig.RespectBoundaries,
		}
	}
	s.mu.RUnlock()

	// Chunk document
	chunker := parser.NewChunker(chunkerConfig)
	chunks, err := chunker.ChunkDocument(doc)
	if err != nil {
		result.Error = "failed to chunk document: " + err.Error()
		return result
	}

	result.ChunkCount = len(chunks)

	// Check if document exists (for replacement)
	existingDoc, _ := s.store.GetDocument(doc.ID)
	wasReplaced := existingDoc != nil

	// Delete existing document if exists (automatic replacement)
	if wasReplaced {
		if err := s.store.DeleteDocument(doc.ID); err != nil {
			result.Error = "failed to delete existing document: " + err.Error()
			return result
		}
	}

	// Create document metadata JSON
	metadataJSON, _ := json.Marshal(doc.Metadata)

	// Insert document
	storeDoc := &vector.Document{
		ID:          doc.ID,
		DocType:     doc.Type,
		Title:       doc.Title,
		FilePath:    fh.Filename, // Use filename as path since file is not persisted
		Status:      doc.Status,
		Version:     doc.Version,
		CreatedDate: doc.CreatedDate,
		LastUpdated: doc.LastUpdated,
		Author:      doc.Author,
		FileHash:    "", // No hash since file is not persisted
		Metadata:    string(metadataJSON),
	}

	if err := s.store.InsertDocument(storeDoc); err != nil {
		result.Error = "failed to insert document: " + err.Error()
		return result
	}

	// Insert chunks and generate embeddings
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

		chunkID, err := s.store.InsertChunk(storeChunk)
		if err != nil {
			result.Error = "failed to insert chunk: " + err.Error()
			// Clean up the document we just inserted
			s.store.DeleteDocument(doc.ID)
			return result
		}

		chunkIDs[i] = chunkID
		chunkContents[i] = chunk.Content
	}

	// Generate and store embeddings - required for RAG functionality
	if s.embeddingClient != nil && len(chunkContents) > 0 {
		embResult, err := s.embeddingClient.EmbedBatch(r.Context(), chunkContents)
		if err != nil {
			s.log("Error: failed to generate embeddings for %s: %v", doc.ID, err)
			// Delete document since embeddings are required for RAG
			s.store.DeleteDocument(doc.ID)
			result.Error = fmt.Sprintf("Falha ao gerar embeddings: %v. Documento não foi indexado.", err)
			return result
		}

		// Store embeddings
		embeddingErrors := 0
		for i, emb := range embResult.Results {
			if err := s.store.InsertChunkEmbedding(chunkIDs[i], emb.Embedding, s.cfg.Model); err != nil {
				s.log("Error: failed to store embedding for chunk %d: %v", i, err)
				embeddingErrors++
			}
		}

		// If any embedding failed to store, delete document and fail
		if embeddingErrors > 0 {
			s.log("Error: %d embeddings failed to store for %s", embeddingErrors, doc.ID)
			s.store.DeleteDocument(doc.ID)
			result.Error = fmt.Sprintf("Falha ao armazenar %d embeddings. Documento não foi indexado.", embeddingErrors)
			return result
		}
	} else if s.embeddingClient == nil {
		// No embedding client configured - this is a fatal error
		s.log("Error: no embedding client configured for %s", doc.ID)
		s.store.DeleteDocument(doc.ID)
		result.Error = "Sistema de embeddings não está configurado. Documento não foi indexado."
		return result
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
			// Ignore errors for cross-references
			s.store.InsertCrossReference(ref)
		}
	}

	if wasReplaced {
		result.Status = "replaced"
	} else {
		result.Status = "success"
	}

	s.log("Uploaded document %s with %d chunks", doc.ID, len(chunks))

	return result
}
