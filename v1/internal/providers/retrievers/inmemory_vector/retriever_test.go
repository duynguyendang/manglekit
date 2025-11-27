package inmemory_vector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit/v1/core"
)

func TestMarkdownLoader_LoadMarkdownFiles(t *testing.T) {
	tmpDir := t.TempDir()

	mdContent1 := `# Introduction
This is the introduction section.

## Background
Some background information here.

## Methods
Method details go here.
`

	mdContent2 := `# API Documentation
## Endpoints
### GET /users
Returns a list of users.

### POST /users
Create a new user.
`

	mdFile1 := filepath.Join(tmpDir, "doc1.md")
	mdFile2 := filepath.Join(tmpDir, "doc2.md")

	if err := os.WriteFile(mdFile1, []byte(mdContent1), 0o644); err != nil {
		t.Fatalf("failed to create test markdown file: %v", err)
	}
	if err := os.WriteFile(mdFile2, []byte(mdContent2), 0o644); err != nil {
		t.Fatalf("failed to create test markdown file: %v", err)
	}

	loader := newMarkdownLoader(500, 100)
	docs, err := loader.loadMarkdownFiles([]string{mdFile1, mdFile2})
	if err != nil {
		t.Fatalf("failed to load markdown files: %v", err)
	}

	if len(docs) == 0 {
		t.Fatalf("expected non-zero number of documents, got %d", len(docs))
	}

	for _, doc := range docs {
		if doc.ID == "" {
			t.Error("document ID should not be empty")
		}
		if doc.Source == "" {
			t.Error("document Source should not be empty")
		}
		if doc.URI == "" {
			t.Error("document URI should not be empty")
		}
		if doc.Text == "" {
			t.Error("document Text should not be empty")
		}
		if doc.Meta == nil {
			t.Error("document Meta should not be nil")
		}
	}

	t.Logf("Successfully loaded %d documents from markdown files", len(docs))
}

func TestMarkdownLoader_ChunkMarkdown(t *testing.T) {
	loader := newMarkdownLoader(200, 50)

	text := `# Section 1
This is a longer paragraph that will be split into chunks to test the chunking logic.
The chunks should respect heading boundaries when possible.

## Subsection 1.1
Another chunk here with more content.
More lines to make this chunk substantial.

# Section 2
Final section content.
`

	chunks := loader.chunkMarkdownWithStructure(text)

	if len(chunks) == 0 {
		t.Error("expected non-empty chunks")
	}

	for i, c := range chunks {
		if c.text == "" {
			t.Errorf("chunk %d is empty", i)
		}
	}

	t.Logf("Chunked text into %d chunks", len(chunks))
}

func TestOptionsValidateLoadingMode(t *testing.T) {
	tests := []struct {
		name    string
		opts    InMemoryVectorOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_with_documents",
			opts: InMemoryVectorOptions{
				Documents: []core.Doc{{ID: "doc1", Text: "content"}},
			},
			wantErr: false,
		},
		{
			name: "valid_with_embedder",
			opts: InMemoryVectorOptions{
				Embedder: "openai",
			},
			wantErr: false,
		},
		{
			name: "valid_with_markdown",
			opts: InMemoryVectorOptions{
				MarkdownFiles: []string{"doc.md"},
				Embedder:      "openai",
			},
			wantErr: false,
		},
		{
			name:    "invalid_no_loading_source",
			opts:    InMemoryVectorOptions{},
			wantErr: true,
		},
		{
			name: "invalid_markdown_without_embedder",
			opts: InMemoryVectorOptions{
				MarkdownFiles: []string{"doc.md"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.validateLoadingMode()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLoadingMode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical_vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal_vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{0.0, 1.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "opposite_vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{-1.0, 0.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "mismatched_length",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "zero_vector",
			a:        []float32{0.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			epsilon := 1e-6
			if (result-tt.expected) > epsilon || (tt.expected-result) > epsilon {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateAndSanitizeMarkdownChunks(t *testing.T) {
	tests := []struct {
		name     string
		input    []chunk
		expected int
	}{
		{
			name:     "empty_slice",
			input:    []chunk{},
			expected: 0,
		},
		{
			name:     "single_valid_chunk",
			input:    []chunk{{text: "valid content", heading: ""}},
			expected: 1,
		},
		{
			name:     "chunks_with_empty_strings",
			input:    []chunk{{text: "content", heading: ""}, {text: "", heading: ""}, {text: "more content", heading: ""}, {text: "   ", heading: ""}},
			expected: 2,
		},
		{
			name:     "multiple_valid_chunks",
			input:    []chunk{{text: "chunk1", heading: ""}, {text: "chunk2", heading: ""}, {text: "chunk3", heading: ""}},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateAndSanitizeMarkdownChunks(tt.input)
			if len(result) != tt.expected {
				t.Errorf("got %d chunks, expected %d", len(result), tt.expected)
			}
		})
	}
}

func TestOptionsProviderName(t *testing.T) {
	opts := InMemoryVectorOptions{}
	if opts.ProviderName() != "inmemory-vector" {
		t.Errorf("ProviderName() = %q, want inmemory-vector", opts.ProviderName())
	}
}

func TestOptionsProviderKind(t *testing.T) {
	opts := InMemoryVectorOptions{}
	if opts.ProviderKind() != core.KindRetriever {
		t.Errorf("ProviderKind() = %v, want %v", opts.ProviderKind(), core.KindRetriever)
	}
}

func TestLoadMarkdownFiles_FileNotFound(t *testing.T) {
	loader := newMarkdownLoader(500, 100)
	docs, err := loader.loadMarkdownFiles([]string{"/nonexistent/path/file.md"})

	if err == nil {
		t.Error("expected error for non-existent file")
	}
	if len(docs) != 0 {
		t.Error("expected no documents for non-existent file")
	}
}

func TestInMemoryVectorRetriever_CosineSimilarityEdgeCases(t *testing.T) {
	v1 := []float32{1.0 / 1.732, 1.0 / 1.732, 1.0 / 1.732}
	v2 := []float32{1.0 / 1.732, 1.0 / 1.732, 1.0 / 1.732}

	sim := cosineSimilarity(v1, v2)
	if sim < 0.999 {
		t.Errorf("expected similarity close to 1.0 for identical normalized vectors, got %f", sim)
	}
}
