package inmemory_vector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/duynguyendang/manglekit/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// chunk represents a semantic chunk with associated metadata.
type chunk struct {
	text    string
	heading string
}

// markdownLoader handles loading and chunking markdown files for RAG.
// It uses goldmark to parse markdown structure and respects heading hierarchy.
type markdownLoader struct {
	chunkSize    int
	chunkOverlap int
	md           goldmark.Markdown
}

// newMarkdownLoader creates a new markdown loader with the given chunk parameters.
func newMarkdownLoader(chunkSize, chunkOverlap int) *markdownLoader {
	if chunkSize <= 0 {
		chunkSize = 500 // default
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 4 // ensure overlap is less than chunk size
	}

	return &markdownLoader{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
		md:           goldmark.New(),
	}
}

// loadMarkdownFiles loads and chunks markdown files from the provided paths.
// Returns a slice of core.Doc with ID, Source, Text populated.
// Each semantic section (respecting heading hierarchy) becomes a separate document.
func (ml *markdownLoader) loadMarkdownFiles(filePaths []string) ([]core.Doc, error) {
	var docs []core.Doc
	docIDCounter := 0

	for _, filePath := range filePaths {
		// Resolve absolute path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %q: %w", filePath, err)
		}

		// Check file exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("markdown file not found: %q", absPath)
		}

		// Read file
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read markdown file %q: %w", absPath, err)
		}

		text := string(content)

		// Parse markdown with goldmark and chunk respecting structure
		chunks := ml.chunkMarkdownWithStructure(text)

		// Create documents from chunks
		for _, chunk := range chunks {
			docIDCounter++
			doc := core.Doc{
				ID:     fmt.Sprintf("%s#chunk-%d", filepath.Base(absPath), docIDCounter),
				Source: absPath,
				URI:    "file://" + absPath,
				Text:   chunk.text,
				Meta: map[string]any{
					"file_path":   absPath,
					"chunk_index": docIDCounter,
					"heading":     chunk.heading,
				},
			}
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// chunkMarkdownWithStructure parses markdown using goldmark and creates semantic chunks
// that respect the markdown structure (headings, paragraphs, lists, code blocks, etc.).
func (ml *markdownLoader) chunkMarkdownWithStructure(md string) []chunk {
	// Parse markdown
	source := []byte(md)
	doc := ml.md.Parser().Parse(text.NewReader(source))

	var chunks []chunk
	var currentChunk strings.Builder
	var currentHeading string

	// Walk AST nodes
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			// If we have accumulated content, save it as a chunk
			if currentChunk.Len() > 0 {
				c := chunk{
					text:    strings.TrimSpace(currentChunk.String()),
					heading: currentHeading,
				}
				if len(c.text) > 0 {
					chunks = append(chunks, c)
				}
				currentChunk.Reset()
			}

			// Update heading context
			headingText := extractNodeText(md, node)
			currentHeading = headingText
			currentChunk.WriteString(headingText)
			currentChunk.WriteString("\n")

		case *ast.Paragraph:
			content := extractNodeText(md, node)
			if len(content) > 0 {
				// Check if adding this would exceed chunk size
				testChunk := currentChunk.String() + content + "\n"
				if len(testChunk) > ml.chunkSize && currentChunk.Len() > 0 {
					// Save current chunk
					c := chunk{
						text:    strings.TrimSpace(currentChunk.String()),
						heading: currentHeading,
					}
					if len(c.text) > 0 {
						chunks = append(chunks, c)
					}

					// Start new chunk with overlap
					overlap := ml.getOverlapText(currentChunk.String())
					currentChunk.Reset()
					if overlap != "" {
						currentChunk.WriteString(overlap)
						currentChunk.WriteString("\n")
					}
					// Add heading context back
					if currentHeading != "" {
						currentChunk.WriteString(currentHeading)
						currentChunk.WriteString("\n")
					}
				}

				currentChunk.WriteString(content)
				currentChunk.WriteString("\n")
			}

		case *ast.List:
			content := extractNodeText(md, node)
			if len(content) > 0 {
				testChunk := currentChunk.String() + content + "\n"
				if len(testChunk) > ml.chunkSize && currentChunk.Len() > 0 {
					c := chunk{
						text:    strings.TrimSpace(currentChunk.String()),
						heading: currentHeading,
					}
					if len(c.text) > 0 {
						chunks = append(chunks, c)
					}

					overlap := ml.getOverlapText(currentChunk.String())
					currentChunk.Reset()
					if overlap != "" {
						currentChunk.WriteString(overlap)
						currentChunk.WriteString("\n")
					}
					if currentHeading != "" {
						currentChunk.WriteString(currentHeading)
						currentChunk.WriteString("\n")
					}
				}

				currentChunk.WriteString(content)
				currentChunk.WriteString("\n")
			}

		case *ast.CodeBlock, *ast.FencedCodeBlock:
			content := extractNodeText(md, node)
			if len(content) > 0 {
				// Code blocks are often significant, preserve them
				testChunk := currentChunk.String() + "\n```\n" + content + "\n```\n"
				if len(testChunk) > ml.chunkSize && currentChunk.Len() > 0 {
					c := chunk{
						text:    strings.TrimSpace(currentChunk.String()),
						heading: currentHeading,
					}
					if len(c.text) > 0 {
						chunks = append(chunks, c)
					}

					overlap := ml.getOverlapText(currentChunk.String())
					currentChunk.Reset()
					if overlap != "" {
						currentChunk.WriteString(overlap)
						currentChunk.WriteString("\n")
					}
					if currentHeading != "" {
						currentChunk.WriteString(currentHeading)
						currentChunk.WriteString("\n")
					}
				}

				currentChunk.WriteString("```\n")
				currentChunk.WriteString(content)
				currentChunk.WriteString("\n```\n")
			}

		case *ast.Blockquote:
			content := extractNodeText(md, node)
			if len(content) > 0 {
				testChunk := currentChunk.String() + "> " + content + "\n"
				if len(testChunk) > ml.chunkSize && currentChunk.Len() > 0 {
					c := chunk{
						text:    strings.TrimSpace(currentChunk.String()),
						heading: currentHeading,
					}
					if len(c.text) > 0 {
						chunks = append(chunks, c)
					}

					overlap := ml.getOverlapText(currentChunk.String())
					currentChunk.Reset()
					if overlap != "" {
						currentChunk.WriteString(overlap)
						currentChunk.WriteString("\n")
					}
					if currentHeading != "" {
						currentChunk.WriteString(currentHeading)
						currentChunk.WriteString("\n")
					}
				}

				currentChunk.WriteString("> ")
				currentChunk.WriteString(content)
				currentChunk.WriteString("\n")
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		// If walk fails, return empty chunks - error is logged at higher level
		return []chunk{}
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		c := chunk{
			text:    strings.TrimSpace(currentChunk.String()),
			heading: currentHeading,
		}
		if len(c.text) > 0 {
			chunks = append(chunks, c)
		}
	}

	// Validate and sanitize chunks
	return validateAndSanitizeMarkdownChunks(chunks)
}

// extractNodeText extracts text content from a goldmark AST node.
// It converts markdown nodes to plain text for chunking purposes.
func extractNodeText(sourceText string, node ast.Node) string {
	sourceBytes := []byte(sourceText)
	var result string

	// Use goldmark's built-in text extraction by walking the tree
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Extract text segments from known node types
		switch n.(type) {
		case *ast.Text:
			text, ok := n.(*ast.Text)
			if ok {
				// Text nodes have a Segment that points to the source
				seg := text.Segment
				result += string(seg.Value(sourceBytes))
			}
		}
		return ast.WalkContinue, nil
	})

	result = strings.TrimSpace(result)
	return result
}

// validateAndSanitizeChunks ensures chunks are valid UTF-8 and non-empty.
func validateAndSanitizeMarkdownChunks(chunks []chunk) []chunk {
	var validated []chunk
	for _, c := range chunks {
		text := strings.TrimSpace(c.text)
		if len(text) == 0 {
			continue
		}
		// Ensure valid UTF-8
		if !utf8.ValidString(text) {
			text = strings.ToValidUTF8(text, "")
		}
		validated = append(validated, chunk{
			text:    text,
			heading: c.heading,
		})
	}
	return validated
}

// getOverlapText extracts the last portion of text for overlap.
// It tries to extract the last chunkOverlap characters while respecting word boundaries.
func (ml *markdownLoader) getOverlapText(text string) string {
	if ml.chunkOverlap <= 0 || len(text) <= ml.chunkOverlap {
		return ""
	}

	// Get last chunkOverlap characters
	runes := []rune(text)
	if len(runes) <= ml.chunkOverlap {
		return text
	}

	overlap := string(runes[len(runes)-ml.chunkOverlap:])

	// Try to find last newline or space for cleaner boundaries
	if idx := strings.LastIndexAny(overlap, "\n "); idx > 0 {
		overlap = overlap[idx+1:]
	}

	return overlap
}

// mergeDocuments combines markdown docs with existing docs.
// If both exist, markdown docs are processed and embedder is required.
func (opts *InMemoryVectorOptions) validateLoadingMode() error {
	docsCount := len(opts.Documents)
	mdCount := len(opts.MarkdownFiles)
	hasEmbedder := opts.Embedder != ""

	// At least one mode must be specified: Documents, MarkdownFiles, OR Embedder
	if docsCount == 0 && mdCount == 0 && !hasEmbedder {
		return fmt.Errorf("either Documents, MarkdownFiles, or Embedder must be provided")
	}

	// If markdown files are provided, embedder is required
	if mdCount > 0 && !hasEmbedder {
		return fmt.Errorf("Embedder is required when MarkdownFiles are provided")
	}

	return nil
}
