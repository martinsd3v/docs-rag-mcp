package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunker_ChunkDocument(t *testing.T) {
	// Create a test document
	doc := &Document{
		ID:      "RFC-001",
		Content: createLargeTestContent(),
		Sections: []Section{
			{
				Level:     1,
				Title:     "Title",
				Path:      "Title",
				StartLine: 0,
				EndLine:   50,
			},
			{
				Level:     2,
				Title:     "Section 1",
				Path:      "Title > Section 1",
				StartLine: 10,
				EndLine:   30,
			},
			{
				Level:     2,
				Title:     "Section 2",
				Path:      "Title > Section 2",
				StartLine: 31,
				EndLine:   50,
			},
		},
	}

	config := &ChunkerConfig{
		MinSize:           100,
		MaxSize:           500,
		Overlap:           50,
		RespectBoundaries: true,
	}

	chunker := NewChunker(config)
	chunks, err := chunker.ChunkDocument(doc)

	require.NoError(t, err)
	assert.Greater(t, len(chunks), 0)

	// Verify chunks have sequential indexes
	for i, chunk := range chunks {
		assert.Equal(t, i, chunk.Index)
		assert.Equal(t, "RFC-001", chunk.DocID)
		assert.NotEmpty(t, chunk.ContentHash)
		assert.Greater(t, chunk.TokenCount, 0)
	}
}

func TestChunker_SmallSection(t *testing.T) {
	content := "# Title\n\nThis is a small section with just a few words."
	lines := len(strings.Split(content, "\n"))

	doc := &Document{
		ID:      "RFC-001",
		Content: content,
		Sections: []Section{
			{
				Level:     1,
				Title:     "Title",
				Path:      "Title",
				StartLine: 0,
				EndLine:   lines,
			},
		},
	}

	chunker := NewChunker(DefaultChunkerConfig())
	chunks, err := chunker.ChunkDocument(doc)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(chunks), 1)
	if len(chunks) > 0 {
		// Content should be present (either the actual text or just context header)
		assert.NotEmpty(t, chunks[0].Content)
	}
}

func TestChunker_LargeSectionSplit(t *testing.T) {
	// Create a large section that needs splitting
	largeContent := strings.Repeat("This is a paragraph with some content.\n\n", 100)

	doc := &Document{
		ID:      "RFC-001",
		Content: "# Title\n\n" + largeContent,
		Sections: []Section{
			{
				Level:     1,
				Title:     "Title",
				Path:      "Title",
				StartLine: 0,
				EndLine:   202,
			},
		},
	}

	config := &ChunkerConfig{
		MinSize:           100,
		MaxSize:           500,
		Overlap:           50,
		RespectBoundaries: true,
	}

	chunker := NewChunker(config)
	chunks, err := chunker.ChunkDocument(doc)

	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1, "Large section should be split into multiple chunks")

	// Verify all chunks are within size limits
	for _, chunk := range chunks {
		assert.LessOrEqual(t, chunk.TokenCount, config.MaxSize*2, "Chunk too large")
	}
}

func TestChunker_WithCodeBlock(t *testing.T) {
	doc := &Document{
		ID: "RFC-001",
		Content: "# Code Example\n\n" +
			"```go\n" +
			"func main() {\n" +
			"    fmt.Println(\"Hello\")\n" +
			"}\n" +
			"```",
		Sections: []Section{
			{
				Level:     1,
				Title:     "Code Example",
				Path:      "Code Example",
				StartLine: 0,
				EndLine:   7,
			},
		},
		CodeBlocks: []CodeBlock{
			{
				Language:  "go",
				Content:   "func main() {\n    fmt.Println(\"Hello\")\n}",
				StartLine: 2,
				EndLine:   6,
			},
		},
	}

	chunker := NewChunker(DefaultChunkerConfig())
	chunks, err := chunker.ChunkDocument(doc)

	require.NoError(t, err)
	assert.Greater(t, len(chunks), 0)
	assert.True(t, chunks[0].HasCodeBlock, "Chunk should detect code block")
}

func TestChunker_WithTable(t *testing.T) {
	doc := &Document{
		ID: "RFC-001",
		Content: "# Table Example\n\n" +
			"| Column 1 | Column 2 |\n" +
			"|----------|----------|\n" +
			"| Value 1  | Value 2  |",
		Sections: []Section{
			{
				Level:     1,
				Title:     "Table Example",
				Path:      "Table Example",
				StartLine: 0,
				EndLine:   5,
			},
		},
		Tables: []Table{
			{
				Headers:   []string{"Column 1", "Column 2"},
				Rows:      [][]string{{"Value 1", "Value 2"}},
				StartLine: 2,
				EndLine:   4,
			},
		},
	}

	chunker := NewChunker(DefaultChunkerConfig())
	chunks, err := chunker.ChunkDocument(doc)

	require.NoError(t, err)
	assert.Greater(t, len(chunks), 0)
	assert.True(t, chunks[0].HasTable, "Chunk should detect table")
}

func TestChunker_OverlapBetweenChunks(t *testing.T) {
	largeContent := ""
	for i := 0; i < 50; i++ {
		largeContent += "Paragraph " + strings.Repeat("word ", 20) + "\n\n"
	}

	doc := &Document{
		ID:      "RFC-001",
		Content: "# Title\n\n" + largeContent,
		Sections: []Section{
			{
				Level:     1,
				Title:     "Title",
				Path:      "Title",
				StartLine: 0,
				EndLine:   100,
			},
		},
	}

	config := &ChunkerConfig{
		MinSize:           200,
		MaxSize:           500,
		Overlap:           100,
		RespectBoundaries: true,
	}

	chunker := NewChunker(config)
	chunks, err := chunker.ChunkDocument(doc)

	require.NoError(t, err)

	if len(chunks) > 1 {
		// Check that second chunk contains overlap separator
		assert.Contains(t, chunks[1].Content, "---", "Should have overlap separator")
	}
}

func TestChunker_CountTokens(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	tests := []struct {
		name     string
		text     string
		minCount int
		maxCount int
	}{
		{
			name:     "Empty",
			text:     "",
			minCount: 0,
			maxCount: 0,
		},
		{
			name:     "Short text",
			text:     "Hello world",
			minCount: 1,
			maxCount: 5,
		},
		{
			name:     "Medium text",
			text:     strings.Repeat("word ", 100),
			minCount: 80,
			maxCount: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := chunker.countTokens(tt.text)
			assert.GreaterOrEqual(t, count, tt.minCount)
			assert.LessOrEqual(t, count, tt.maxCount)
		})
	}
}

func TestChunker_FormatChunkContent(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	tests := []struct {
		name        string
		sectionPath string
		content     string
		expected    string
	}{
		{
			name:        "With path",
			sectionPath: "Executive Summary > Problem",
			content:     "Some content here",
			expected:    "Context: Executive Summary > Problem\n\nSome content here",
		},
		{
			name:        "Without path",
			sectionPath: "",
			content:     "Some content here",
			expected:    "Some content here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunker.formatChunkContent(tt.sectionPath, tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChunker_HashContent(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	content1 := "Same content"
	content2 := "Same content"
	content3 := "Different content"

	hash1 := chunker.hashContent(content1)
	hash2 := chunker.hashContent(content2)
	hash3 := chunker.hashContent(content3)

	assert.Equal(t, hash1, hash2, "Same content should produce same hash")
	assert.NotEqual(t, hash1, hash3, "Different content should produce different hash")
	assert.Len(t, hash1, 64, "SHA256 hash should be 64 characters")
}

func TestChunker_DetectCodeBlock(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	assert.True(t, chunker.containsCodeBlock("```go\ncode\n```"))
	assert.True(t, chunker.containsCodeBlock("Some text\n```python\ncode\n```\nmore text"))
	assert.False(t, chunker.containsCodeBlock("No code blocks here"))
}

func TestChunker_DetectTable(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	assert.True(t, chunker.containsTable("| Col |\n|-----|\n| Val |"))
	assert.True(t, chunker.containsTable("|---|---|"))
	assert.False(t, chunker.containsTable("No tables here"))
}

// Helper function to create large test content
func createLargeTestContent() string {
	var sb strings.Builder

	sb.WriteString("# Main Title\n\n")
	sb.WriteString("Introduction paragraph.\n\n")

	sb.WriteString("## Section 1\n\n")
	for i := 0; i < 10; i++ {
		sb.WriteString("Paragraph in section 1. ")
	}
	sb.WriteString("\n\n")

	sb.WriteString("## Section 2\n\n")
	for i := 0; i < 10; i++ {
		sb.WriteString("Paragraph in section 2. ")
	}
	sb.WriteString("\n\n")

	return sb.String()
}
