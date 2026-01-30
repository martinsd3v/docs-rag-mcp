package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestParser_ParseFile(t *testing.T) {
	content := []byte(`# RFC-001: Test Document

**Status:** Approved
**Version:** v1.0
**Created:** 2024-01-15
**Last Updated:** 2024-01-20
**Author:** Test Author

## Executive Summary

This is the executive summary.

### Problem & Motivation

This is a subsection with some content.

` + "```go" + `
func main() {
	fmt.Println("Hello")
}
` + "```" + `

## Implementation

| Feature | Status |
|---------|--------|
| Auth    | Done   |
| API     | TODO   |

See [RFC-002](./RFC-002-test.md) for details.

`)

	parser := NewParser()
	doc, err := parser.ParseFile("/path/to/RFC-001-test.md", content)

	require.NoError(t, err)
	assert.Equal(t, "RFC-001", doc.ID)
	assert.Equal(t, "RFC", doc.Type)
	assert.Equal(t, "RFC-001: Test Document", doc.Title)
	assert.Equal(t, "Approved", doc.Status)
	assert.Equal(t, "v1.0", doc.Version)
	assert.Equal(t, "Test Author", doc.Author)
	assert.NotNil(t, doc.CreatedDate)
	assert.NotNil(t, doc.LastUpdated)

	// Check sections
	assert.GreaterOrEqual(t, len(doc.Sections), 3)

	// Check code blocks
	assert.Equal(t, 1, len(doc.CodeBlocks))
	assert.Equal(t, "go", doc.CodeBlocks[0].Language)
	assert.Contains(t, doc.CodeBlocks[0].Content, "func main()")

	// Check tables
	assert.Equal(t, 1, len(doc.Tables))
	assert.Equal(t, []string{"Feature", "Status"}, doc.Tables[0].Headers)
	assert.Equal(t, 2, len(doc.Tables[0].Rows))

	// Check links
	assert.GreaterOrEqual(t, len(doc.Links), 1)
	hasRFC002 := false
	for _, link := range doc.Links {
		if link.DocID == "RFC-002" {
			hasRFC002 = true
		}
	}
	assert.True(t, hasRFC002)
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "Standard format",
			content: `# RFC-001: Test
**Status:** Approved
**Version:** v1.0
**Author:** John Doe`,
			expected: map[string]string{
				"status":  "Approved",
				"version": "v1.0",
				"author":  "John Doe",
			},
		},
		{
			name: "Without bold",
			content: `# ADR-001: Test
Status: Draft
Version: v2.0`,
			expected: map[string]string{
				"status":  "Draft",
				"version": "v2.0",
			},
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{
				FilePath: "/path/to/file.md",
				Metadata: make(map[string]string),
			}
			err := parser.extractMetadata(doc, []byte(tt.content))
			require.NoError(t, err)

			for key, expectedValue := range tt.expected {
				actualValue, ok := doc.Metadata[key]
				assert.True(t, ok, "Key %s not found", key)
				assert.Equal(t, expectedValue, actualValue)
			}
		})
	}
}

func TestParseSections(t *testing.T) {
	content := []byte(`# Title

## Section 1

Content 1

### Subsection 1.1

Content 1.1

## Section 2

Content 2
`)

	parser := NewParser()
	doc := &Document{
		Metadata: make(map[string]string),
	}

	md := goldmark.New()
	reader := text.NewReader(content)
	node := md.Parser().Parse(reader)
	err := parser.parseSections(doc, node, content)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(doc.Sections), 4)

	// Check hierarchy
	var section1 *Section
	for i := range doc.Sections {
		if doc.Sections[i].Title == "Section 1" {
			section1 = &doc.Sections[i]
			break
		}
	}
	require.NotNil(t, section1)
	assert.Equal(t, 2, section1.Level)

	// Check path
	hasSubsection := false
	for i := range doc.Sections {
		if doc.Sections[i].Title == "Subsection 1.1" {
			assert.Contains(t, doc.Sections[i].Path, "Section 1")
			assert.Contains(t, doc.Sections[i].Path, "Subsection 1.1")
			hasSubsection = true
		}
	}
	assert.True(t, hasSubsection)
}

func TestExtractCodeBlocks(t *testing.T) {
	content := []byte("```go\nfunc test() {}\n```\n\n```python\ndef test():\n    pass\n```")

	parser := NewParser()
	doc := &Document{
		Metadata: make(map[string]string),
		Sections: make([]Section, 0),
	}

	md := goldmark.New()
	reader := text.NewReader(content)
	node := md.Parser().Parse(reader)
	parser.extractCodeBlocks(doc, node, content)

	assert.Equal(t, 2, len(doc.CodeBlocks))
	assert.Equal(t, "go", doc.CodeBlocks[0].Language)
	assert.Equal(t, "python", doc.CodeBlocks[1].Language)
}

func TestExtractLinks(t *testing.T) {
	content := []byte(`
See [RFC-001](./RFC-001.md) for details.
Also check [ADR-002](../adr/ADR-002.md).
[External link](https://example.com)
`)

	parser := NewParser()
	doc := &Document{
		Metadata: make(map[string]string),
	}

	md := goldmark.New()
	reader := text.NewReader(content)
	node := md.Parser().Parse(reader)
	parser.extractLinks(doc, node, content)

	assert.GreaterOrEqual(t, len(doc.Links), 3)

	// Check doc IDs extracted
	docIDs := make([]string, 0)
	for _, link := range doc.Links {
		if link.DocID != "" {
			docIDs = append(docIDs, link.DocID)
		}
	}

	assert.Contains(t, docIDs, "RFC-001")
	assert.Contains(t, docIDs, "ADR-002")
}
