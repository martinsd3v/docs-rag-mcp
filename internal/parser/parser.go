package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Document represents a parsed Markdown document
type Document struct {
	ID           string            `json:"id"`            // RFC-001, ADR-001, etc
	Type         string            `json:"type"`          // RFC, ADR, BDR, Guideline, Roadmap
	Title        string            `json:"title"`         // Document title
	FilePath     string            `json:"file_path"`     // Absolute path to file
	Status       string            `json:"status"`        // Draft, Approved, etc
	Version      string            `json:"version"`       // v1.0, v2.1, etc
	CreatedDate  *time.Time        `json:"created_date"`  // When document was created
	LastUpdated  *time.Time        `json:"last_updated"`  // When document was last modified
	Author       string            `json:"author"`        // Document author
	Metadata     map[string]string `json:"metadata"`      // Extra metadata
	Content      string            `json:"content"`       // Full Markdown content
	AST          ast.Node          `json:"-"`             // Goldmark AST
	Sections     []Section         `json:"sections"`      // Parsed sections
	CodeBlocks   []CodeBlock       `json:"code_blocks"`   // Code blocks
	Tables       []Table           `json:"tables"`        // Tables
	Links        []Link            `json:"links"`         // Links to other documents
}

// Section represents a document section with hierarchy
type Section struct {
	Level       int      `json:"level"`        // 1 (H1), 2 (H2), 3 (H3), etc
	Title       string   `json:"title"`        // Section title
	Path        string   `json:"path"`         // "Executive Summary > Problem & Motivation"
	Content     string   `json:"content"`      // Section content (without subsections)
	StartLine   int      `json:"start_line"`   // Starting line in original file
	EndLine     int      `json:"end_line"`     // Ending line in original file
	Parent      *Section `json:"-"`            // Parent section
	Children    []Section `json:"children"`    // Subsections
	HasCode     bool     `json:"has_code"`     // Contains code blocks
	HasTable    bool     `json:"has_table"`    // Contains tables
}

// CodeBlock represents a code block in the document
type CodeBlock struct {
	Language  string `json:"language"`   // Programming language
	Content   string `json:"content"`    // Code content
	StartLine int    `json:"start_line"` // Starting line
	EndLine   int    `json:"end_line"`   // Ending line
}

// Table represents a table in the document
type Table struct {
	Headers   []string   `json:"headers"`    // Table headers
	Rows      [][]string `json:"rows"`       // Table rows
	StartLine int        `json:"start_line"` // Starting line
	EndLine   int        `json:"end_line"`   // Ending line
}

// Link represents a link to another document
type Link struct {
	Text      string `json:"text"`       // Link text
	Target    string `json:"target"`     // Link target (file path or URL)
	DocID     string `json:"doc_id"`     // Extracted document ID (RFC-001, etc)
	Context   string `json:"context"`    // Surrounding text
	LineNumber int   `json:"line_number"` // Line number
}

// Parser handles Markdown parsing using goldmark
type Parser struct {
	markdown goldmark.Markdown
}

// NewParser creates a new Markdown parser
func NewParser() *Parser {
	return &Parser{
		markdown: goldmark.New(
			goldmark.WithExtensions(),
		),
	}
}

// ParseFile parses a Markdown file and returns a Document
func (p *Parser) ParseFile(filePath string, content []byte) (*Document, error) {
	doc := &Document{
		FilePath: filePath,
		Content:  string(content),
		Metadata: make(map[string]string),
		Sections: make([]Section, 0),
		CodeBlocks: make([]CodeBlock, 0),
		Tables: make([]Table, 0),
		Links: make([]Link, 0),
	}

	// Parse Markdown to AST
	reader := text.NewReader(content)
	node := p.markdown.Parser().Parse(reader)
	doc.AST = node

	// Extract metadata from front matter or first section
	if err := p.extractMetadata(doc, content); err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Parse sections with hierarchy
	if err := p.parseSections(doc, node, content); err != nil {
		return nil, fmt.Errorf("failed to parse sections: %w", err)
	}

	// Extract code blocks
	p.extractCodeBlocks(doc, node, content)

	// Extract tables
	p.extractTables(doc, node, content)

	// Extract links and cross-references
	p.extractLinks(doc, node, content)

	return doc, nil
}

// extractMetadata extracts metadata from the document
// Looks for patterns like:
// - Status: Approved
// - Version: v1.0
// - Created: 2024-01-15
// - Last Updated: 2024-01-20
// - Author: John Doe
func (p *Parser) extractMetadata(doc *Document, content []byte) error {
	lines := bytes.Split(content, []byte("\n"))

	// Extract document ID from filename or title
	// Example: RFC-001-microservice-architecture.md -> RFC-001
	docIDPattern := regexp.MustCompile(`(RFC|ADR|BDR|GUIDELINE|ROADMAP)-(\d+)`)
	if matches := docIDPattern.FindStringSubmatch(doc.FilePath); len(matches) >= 3 {
		doc.ID = fmt.Sprintf("%s-%s", matches[1], matches[2])
		doc.Type = matches[1]
	}

	// Parse metadata fields (usually in first 20 lines)
	maxLines := 20
	if len(lines) < maxLines {
		maxLines = len(lines)
	}

	metadataPatterns := map[string]*regexp.Regexp{
		"status":       regexp.MustCompile(`(?i)^\*?\*?Status\*?\*?:\s*(.+)$`),
		"version":      regexp.MustCompile(`(?i)^\*?\*?Version\*?\*?:\s*(.+)$`),
		"created":      regexp.MustCompile(`(?i)^\*?\*?Created\*?\*?:\s*(.+)$`),
		"last_updated": regexp.MustCompile(`(?i)^\*?\*?(Last\s+Updated|Updated)\*?\*?:\s*(.+)$`),
		"author":       regexp.MustCompile(`(?i)^\*?\*?Author\*?\*?:\s*(.+)$`),
	}

	for i := 0; i < maxLines; i++ {
		line := string(lines[i])

		// Extract title from first H1
		if strings.HasPrefix(line, "# ") {
			doc.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}

		// Extract metadata fields
		for key, pattern := range metadataPatterns {
			if matches := pattern.FindStringSubmatch(line); len(matches) >= 2 {
				value := strings.TrimSpace(matches[len(matches)-1])
				// Remove any remaining asterisks from bold formatting
				value = strings.Trim(value, "*")
				value = strings.TrimSpace(value)
				doc.Metadata[key] = value

				// Map to specific fields
				switch key {
				case "status":
					doc.Status = value
				case "version":
					doc.Version = value
				case "created":
					if t, err := parseDate(value); err == nil {
						doc.CreatedDate = &t
					}
				case "last_updated":
					if t, err := parseDate(value); err == nil {
						doc.LastUpdated = &t
					}
				case "author":
					doc.Author = value
				}
			}
		}
	}

	// If no title found, use filename
	if doc.Title == "" {
		parts := strings.Split(doc.FilePath, "/")
		filename := parts[len(parts)-1]
		doc.Title = strings.TrimSuffix(filename, ".md")
	}

	return nil
}

// parseSections builds a hierarchical structure of sections
func (p *Parser) parseSections(doc *Document, node ast.Node, content []byte) error {
	var sectionStack []*Section // Stack to track hierarchy

	// Pre-compute line number mapping (byte offset -> line number)
	lineOffsets := computeLineOffsets(content)

	err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if heading, ok := n.(*ast.Heading); ok {
			level := heading.Level
			title := extractHeadingText(n, content)

			// Get byte offset and convert to line number
			byteOffset := 0
			if n.Lines() != nil && n.Lines().Len() > 0 {
				byteOffset = n.Lines().At(0).Start
			}
			lineNumber := byteOffsetToLineNumber(lineOffsets, byteOffset)

			section := Section{
				Level:     level,
				Title:     title,
				StartLine: lineNumber,
				Children:  make([]Section, 0),
			}

			// Build section path based on hierarchy
			if level == 1 {
				section.Path = title
				sectionStack = []*Section{&section}
			} else {
				// Find parent section
				for len(sectionStack) > 0 && sectionStack[len(sectionStack)-1].Level >= level {
					sectionStack = sectionStack[:len(sectionStack)-1]
				}

				if len(sectionStack) > 0 {
					parent := sectionStack[len(sectionStack)-1]
					section.Parent = parent
					section.Path = parent.Path + " > " + title
					parent.Children = append(parent.Children, section)
				} else {
					section.Path = title
				}

				sectionStack = append(sectionStack, &section)
			}

			// Set EndLine of previous section
			if len(doc.Sections) > 0 {
				doc.Sections[len(doc.Sections)-1].EndLine = lineNumber - 1
			}

			doc.Sections = append(doc.Sections, section)
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return err
	}

	// Set end line for last section
	if len(doc.Sections) > 0 {
		lines := bytes.Split(content, []byte("\n"))
		doc.Sections[len(doc.Sections)-1].EndLine = len(lines) - 1
	}

	return nil
}

// extractCodeBlocks finds all code blocks in the document
func (p *Parser) extractCodeBlocks(doc *Document, node ast.Node, content []byte) {
	// Pre-compute line number mapping
	lineOffsets := computeLineOffsets(content)

	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if codeBlock, ok := n.(*ast.FencedCodeBlock); ok {
			// Skip code blocks with no line information
			if codeBlock.Lines() == nil || codeBlock.Lines().Len() == 0 {
				return ast.WalkContinue, nil
			}

			language := string(codeBlock.Language(content))
			code := extractNodeContent(codeBlock, content)

			// Convert byte offsets to line numbers
			startOffset := codeBlock.Lines().At(0).Start
			endOffset := codeBlock.Lines().At(codeBlock.Lines().Len() - 1).Stop

			block := CodeBlock{
				Language:  language,
				Content:   code,
				StartLine: byteOffsetToLineNumber(lineOffsets, startOffset),
				EndLine:   byteOffsetToLineNumber(lineOffsets, endOffset),
			}

			doc.CodeBlocks = append(doc.CodeBlocks, block)

			// Mark sections containing code
			for i := range doc.Sections {
				if doc.Sections[i].StartLine <= block.StartLine &&
				   doc.Sections[i].EndLine >= block.EndLine {
					doc.Sections[i].HasCode = true
				}
			}
		}

		return ast.WalkContinue, nil
	})
}

// extractTables finds all tables in the document using regex
func (p *Parser) extractTables(doc *Document, node ast.Node, content []byte) {
	lines := bytes.Split(content, []byte("\n"))

	// Pattern to match table separator line (e.g., |---|---|)
	separatorPattern := regexp.MustCompile(`^\s*\|[\s\-:]+\|`)

	inTable := false
	tableStart := 0
	var tableLines []string

	for lineNum, line := range lines {
		lineStr := string(line)

		// Check if this is a table separator
		if separatorPattern.MatchString(lineStr) {
			if !inTable {
				inTable = true
				tableStart = lineNum - 1 // Header is one line before separator
				if tableStart < 0 {
					tableStart = 0
				}
				// Add header line if it exists
				if tableStart < len(lines) {
					tableLines = append(tableLines, string(lines[tableStart]))
				}
			}
			tableLines = append(tableLines, lineStr)
		} else if inTable {
			// Check if line looks like a table row
			if strings.HasPrefix(strings.TrimSpace(lineStr), "|") && strings.HasSuffix(strings.TrimSpace(lineStr), "|") {
				tableLines = append(tableLines, lineStr)
			} else {
				// End of table
				if len(tableLines) >= 2 {
					tbl := p.parseTableLines(tableLines, tableStart, lineNum-1)
					doc.Tables = append(doc.Tables, tbl)

					// Mark sections containing tables
					for i := range doc.Sections {
						if doc.Sections[i].StartLine <= tbl.StartLine &&
						   doc.Sections[i].EndLine >= tbl.EndLine {
							doc.Sections[i].HasTable = true
						}
					}
				}
				inTable = false
				tableLines = nil
			}
		}
	}

	// Handle table at end of file
	if inTable && len(tableLines) >= 2 {
		tbl := p.parseTableLines(tableLines, tableStart, len(lines)-1)
		doc.Tables = append(doc.Tables, tbl)

		for i := range doc.Sections {
			if doc.Sections[i].StartLine <= tbl.StartLine &&
			   doc.Sections[i].EndLine >= tbl.EndLine {
				doc.Sections[i].HasTable = true
			}
		}
	}
}

// parseTableLines parses table lines into a Table struct
func (p *Parser) parseTableLines(lines []string, startLine, endLine int) Table {
	headers := make([]string, 0)
	rows := make([][]string, 0)

	for i, line := range lines {
		// Skip separator line
		if strings.Contains(line, "---") || strings.Contains(line, ":--") || strings.Contains(line, "--:") {
			continue
		}

		// Parse cells
		cells := make([]string, 0)
		parts := strings.Split(line, "|")
		for _, part := range parts {
			cell := strings.TrimSpace(part)
			if cell != "" {
				cells = append(cells, cell)
			}
		}

		if len(cells) > 0 {
			if i == 0 {
				headers = cells
			} else {
				rows = append(rows, cells)
			}
		}
	}

	return Table{
		Headers:   headers,
		Rows:      rows,
		StartLine: startLine,
		EndLine:   endLine,
	}
}

// extractLinks finds all links and cross-references
func (p *Parser) extractLinks(doc *Document, node ast.Node, content []byte) {
	docIDPattern := regexp.MustCompile(`(RFC|ADR|BDR|GUIDELINE|ROADMAP)-(\d+)`)

	// Pre-compute line number mapping
	lineOffsets := computeLineOffsets(content)

	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if link, ok := n.(*ast.Link); ok {
			text := extractNodeContent(link, content)
			target := string(link.Destination)

			// Extract doc ID from link target or text
			var docID string
			if matches := docIDPattern.FindStringSubmatch(target); len(matches) >= 3 {
				docID = fmt.Sprintf("%s-%s", matches[1], matches[2])
			} else if matches := docIDPattern.FindStringSubmatch(text); len(matches) >= 3 {
				docID = fmt.Sprintf("%s-%s", matches[1], matches[2])
			}

			// Get line number from parent paragraph if possible
			lineNumber := 0
			if parent := link.Parent(); parent != nil && parent.Lines() != nil && parent.Lines().Len() > 0 {
				byteOffset := parent.Lines().At(0).Start
				lineNumber = byteOffsetToLineNumber(lineOffsets, byteOffset)
			}

			l := Link{
				Text:       text,
				Target:     target,
				DocID:      docID,
				LineNumber: lineNumber,
			}

			doc.Links = append(doc.Links, l)
		}

		return ast.WalkContinue, nil
	})
}

// Helper functions

func extractHeadingText(node ast.Node, content []byte) string {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		segment := child.Text(content)
		buf.Write(segment)
	}
	return strings.TrimSpace(buf.String())
}

func extractNodeContent(node ast.Node, content []byte) string {
	var buf bytes.Buffer

	// Check if it's a text node first (most common inline case)
	if textNode, ok := node.(*ast.Text); ok {
		buf.Write(textNode.Segment.Value(content))
		return strings.TrimSpace(buf.String())
	}

	// Check if it's a string node
	if stringNode, ok := node.(*ast.String); ok {
		buf.Write(stringNode.Value)
		return strings.TrimSpace(buf.String())
	}

	// For other inline nodes with children, recurse
	if node.HasChildren() && node.Kind() != ast.KindDocument {
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			buf.WriteString(extractNodeContent(child, content))
			if child.NextSibling() != nil {
				buf.WriteString(" ")
			}
		}
		return strings.TrimSpace(buf.String())
	}

	// For block nodes with lines
	if node.Lines() != nil && node.Lines().Len() > 0 {
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			buf.Write(line.Value(content))
			if i < node.Lines().Len()-1 {
				buf.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(buf.String())
}

func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"January 2, 2006",
		"Jan 2, 2006",
		"02-01-2006",
		"02/01/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// computeLineOffsets returns a slice where index i contains the byte offset
// where line i starts (0-indexed)
func computeLineOffsets(content []byte) []int {
	offsets := []int{0} // Line 0 starts at byte 0
	for i, b := range content {
		if b == '\n' && i+1 < len(content) {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

// byteOffsetToLineNumber converts a byte offset to a line number (0-indexed)
func byteOffsetToLineNumber(lineOffsets []int, byteOffset int) int {
	// Binary search for the line containing this offset
	low, high := 0, len(lineOffsets)-1
	for low < high {
		mid := (low + high + 1) / 2
		if lineOffsets[mid] <= byteOffset {
			low = mid
		} else {
			high = mid - 1
		}
	}
	return low
}
