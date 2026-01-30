package parser

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Chunk represents a semantic chunk of a document
type Chunk struct {
	Index         int    `json:"index"`          // Order within document (0, 1, 2, ...)
	DocID         string `json:"doc_id"`         // Document ID
	SectionPath   string `json:"section_path"`   // "Executive Summary > Problem"
	SectionLevel  int    `json:"section_level"`  // Heading level
	Content       string `json:"content"`        // Chunk content with context
	ContentHash   string `json:"content_hash"`   // SHA256 for deduplication
	TokenCount    int    `json:"token_count"`    // Approximate token count
	StartLine     int    `json:"start_line"`     // Starting line in file
	EndLine       int    `json:"end_line"`       // Ending line in file
	HasCodeBlock  bool   `json:"has_code_block"` // Contains code blocks
	HasTable      bool   `json:"has_table"`      // Contains tables
}

// ChunkerConfig holds configuration for chunking
type ChunkerConfig struct {
	MinSize           int  `json:"min_size"`            // Minimum chunk size in tokens
	MaxSize           int  `json:"max_size"`            // Maximum chunk size in tokens
	Overlap           int  `json:"overlap"`             // Overlap in tokens
	RespectBoundaries bool `json:"respect_boundaries"`  // Don't break code blocks, tables
}

// DefaultChunkerConfig returns default chunking configuration
func DefaultChunkerConfig() *ChunkerConfig {
	return &ChunkerConfig{
		MinSize:           500,
		MaxSize:           1000,
		Overlap:           150,
		RespectBoundaries: true,
	}
}

// Chunker handles semantic chunking of documents
type Chunker struct {
	config *ChunkerConfig
}

// NewChunker creates a new chunker with given configuration
func NewChunker(config *ChunkerConfig) *Chunker {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	return &Chunker{
		config: config,
	}
}

// ChunkDocument splits a document into semantic chunks
func (c *Chunker) ChunkDocument(doc *Document) ([]Chunk, error) {
	chunks := make([]Chunk, 0)
	chunkIndex := 0

	// If document has no sections, create one big chunk
	if len(doc.Sections) == 0 {
		chunk := c.createChunkFromContent(
			doc.ID,
			"",
			0,
			doc.Content,
			0,
			len(strings.Split(doc.Content, "\n")),
			chunkIndex,
		)
		chunks = append(chunks, chunk)
		return chunks, nil
	}

	// Process each section directly
	// Each section becomes one or more chunks based on size
	for i := range doc.Sections {
		section := &doc.Sections[i]
		sectionChunks := c.chunkSection(doc, section, &chunkIndex)
		chunks = append(chunks, sectionChunks...)
	}

	// Add overlap between chunks
	chunks = c.addOverlap(chunks)

	return chunks, nil
}

// chunkSection creates chunks for a single section
func (c *Chunker) chunkSection(doc *Document, section *Section, chunkIndex *int) []Chunk {
	chunks := make([]Chunk, 0)

	// Extract section content
	content := c.extractSectionContent(doc, section)

	// Skip empty sections
	if strings.TrimSpace(content) == "" {
		return chunks
	}

	tokens := c.countTokens(content)

	// Determine if section has code blocks or tables
	hasCode, hasTable := c.detectSpecialContent(doc, section)

	// If section is small enough, create single chunk
	if tokens <= c.config.MaxSize {
		chunk := Chunk{
			Index:        *chunkIndex,
			DocID:        doc.ID,
			SectionPath:  section.Path,
			SectionLevel: section.Level,
			Content:      c.formatChunkContent(section.Path, content),
			ContentHash:  c.hashContent(content),
			TokenCount:   tokens,
			StartLine:    section.StartLine,
			EndLine:      section.EndLine,
			HasCodeBlock: hasCode,
			HasTable:     hasTable,
		}
		chunks = append(chunks, chunk)
		*chunkIndex++
	} else {
		// Section is too large, split by paragraphs
		paraChunks := c.splitByParagraphs(doc, section, content, chunkIndex)
		chunks = append(chunks, paraChunks...)
	}

	return chunks
}

// extractSectionContent extracts content for a section
func (c *Chunker) extractSectionContent(doc *Document, section *Section) string {
	lines := strings.Split(doc.Content, "\n")

	// Use section's StartLine and EndLine directly
	startIdx := section.StartLine
	endIdx := section.EndLine + 1 // +1 because endIdx is exclusive in slice

	// Bounds checking
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	if startIdx >= endIdx {
		return ""
	}

	// Extract content
	content := strings.Join(lines[startIdx:endIdx], "\n")

	// Remove the heading line itself
	content = regexp.MustCompile(`^#{1,6}\s+.+\n?`).ReplaceAllString(content, "")

	return strings.TrimSpace(content)
}

// splitByParagraphs splits content into chunks by paragraphs
func (c *Chunker) splitByParagraphs(doc *Document, section *Section, content string, chunkIndex *int) []Chunk {
	chunks := make([]Chunk, 0)

	// Split by double newlines (paragraphs)
	paragraphs := strings.Split(content, "\n\n")

	currentChunk := ""
	currentTokens := 0
	startLine := section.StartLine

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		paraTokens := c.countTokens(para)

		// If adding this paragraph exceeds max size, save current chunk
		if currentTokens+paraTokens > c.config.MaxSize && currentTokens >= c.config.MinSize {
			if currentChunk != "" {
				chunk := c.createChunkFromContent(
					doc.ID,
					section.Path,
					section.Level,
					currentChunk,
					startLine,
					section.EndLine,
					*chunkIndex,
				)
				chunks = append(chunks, chunk)
				*chunkIndex++

				// Reset for next chunk
				currentChunk = ""
				currentTokens = 0
			}
		}

		// Add paragraph to current chunk
		if currentChunk != "" {
			currentChunk += "\n\n"
		}
		currentChunk += para
		currentTokens += paraTokens
	}

	// Save remaining content
	if currentChunk != "" {
		chunk := c.createChunkFromContent(
			doc.ID,
			section.Path,
			section.Level,
			currentChunk,
			startLine,
			section.EndLine,
			*chunkIndex,
		)
		chunks = append(chunks, chunk)
		*chunkIndex++
	}

	return chunks
}

// createChunkFromContent creates a chunk from raw content
func (c *Chunker) createChunkFromContent(docID, sectionPath string, level int, content string, startLine, endLine, index int) Chunk {
	formattedContent := c.formatChunkContent(sectionPath, content)

	return Chunk{
		Index:        index,
		DocID:        docID,
		SectionPath:  sectionPath,
		SectionLevel: level,
		Content:      formattedContent,
		ContentHash:  c.hashContent(content),
		TokenCount:   c.countTokens(formattedContent),
		StartLine:    startLine,
		EndLine:      endLine,
		HasCodeBlock: c.containsCodeBlock(content),
		HasTable:     c.containsTable(content),
	}
}

// formatChunkContent adds context header to chunk content
func (c *Chunker) formatChunkContent(sectionPath, content string) string {
	if sectionPath == "" {
		return content
	}

	return fmt.Sprintf("Context: %s\n\n%s", sectionPath, content)
}

// addOverlap adds overlap between consecutive chunks
// Skips overlap for chunks with code blocks or tables to preserve formatting
func (c *Chunker) addOverlap(chunks []Chunk) []Chunk {
	if len(chunks) <= 1 || c.config.Overlap == 0 {
		return chunks
	}

	for i := 1; i < len(chunks); i++ {
		prevChunk := chunks[i-1]

		// Skip overlap if previous chunk has code blocks or tables
		// This prevents breaking ASCII diagrams and code formatting
		if prevChunk.HasCodeBlock || prevChunk.HasTable {
			continue
		}

		// Extract overlap from previous chunk
		if prevChunk.TokenCount > c.config.Overlap {
			// Take last N tokens from previous chunk
			overlap := c.extractLastTokens(prevChunk.Content, c.config.Overlap)

			// Prepend to current chunk
			chunks[i].Content = overlap + "\n\n---\n\n" + chunks[i].Content
			chunks[i].TokenCount = c.countTokens(chunks[i].Content)

			// Update hash
			chunks[i].ContentHash = c.hashContent(chunks[i].Content)
		}
	}

	return chunks
}

// extractLastTokens extracts approximately the last N tokens from text
// Preserves line breaks and formatting
func (c *Chunker) extractLastTokens(text string, n int) string {
	// Count tokens in the text
	totalTokens := c.countTokens(text)

	// If text has fewer tokens than requested, return all
	if totalTokens <= n {
		return text
	}

	// Split by words but preserve structure
	lines := strings.Split(text, "\n")

	// Work backwards from the end to collect approximately N tokens
	collectedTokens := 0
	targetTokens := n
	var resultLines []string

	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		lineTokens := c.countTokens(line)

		// Add this line to result
		resultLines = append([]string{line}, resultLines...)
		collectedTokens += lineTokens

		// Stop when we have enough tokens
		if collectedTokens >= targetTokens {
			break
		}
	}

	return strings.Join(resultLines, "\n")
}

// detectSpecialContent checks if section contains code blocks or tables
func (c *Chunker) detectSpecialContent(doc *Document, section *Section) (hasCode, hasTable bool) {
	// Check code blocks
	for _, cb := range doc.CodeBlocks {
		if cb.StartLine >= section.StartLine && cb.EndLine <= section.EndLine {
			hasCode = true
			break
		}
	}

	// Check tables
	for _, tbl := range doc.Tables {
		if tbl.StartLine >= section.StartLine && tbl.EndLine <= section.EndLine {
			hasTable = true
			break
		}
	}

	return hasCode, hasTable
}

// countTokens estimates token count for text
// Uses a simple heuristic: ~4 characters per token
// This is rough but fast; for exact counts, use tiktoken
func (c *Chunker) countTokens(text string) int {
	// Remove extra whitespace
	text = strings.TrimSpace(text)

	// Count words
	words := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			words++
			inWord = true
		}
	}

	// Rough approximation: 1 token ≈ 0.75 words or 4 characters
	tokensByWords := int(float64(words) * 1.33)
	tokensByChars := len(text) / 4

	// Use average of both methods
	return (tokensByWords + tokensByChars) / 2
}

// containsCodeBlock checks if content contains code blocks
func (c *Chunker) containsCodeBlock(content string) bool {
	return strings.Contains(content, "```")
}

// containsTable checks if content contains markdown tables
func (c *Chunker) containsTable(content string) bool {
	// Simple heuristic: look for table separator lines
	tablePattern := regexp.MustCompile(`\|[\s\-:]+\|`)
	return tablePattern.MatchString(content)
}

// hashContent creates SHA256 hash of content for deduplication
func (c *Chunker) hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
