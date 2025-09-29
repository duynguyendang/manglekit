// Package retrieval provides BM25 retrieval functionality.
package retrieval

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/go-nlp/bm25"
	"github.com/go-nlp/tfidf"
	"gopkg.in/yaml.v3"
	"ndduy.dev/manglekit/internal/types"
)

const (
	k1      = 2.0
	b_param = 0.75
)

// doc is a custom type that implements the tfidf.Document interface.
type doc []int

func (d doc) IDs() []int { return d }

// BM25 implements the BM25Retriever interface.
type BM25 struct {
	tf     *tfidf.TFIDF
	docs   []*ai.Document
	vocab  map[string]int
	tfDocs []tfidf.Document
}

// NewBM25 creates a new BM25 retriever.
func NewBM25(ctx context.Context, path string) (*BM25, error) {
	docs, err := loadDocuments(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load documents for BM25: %w", err)
	}

	// Build vocabulary and tokenized docs
	vocab := make(map[string]int)
	var tokenizedDocs [][]string
	idCounter := 0
	for _, d := range docs {
		content := ""
		if len(d.Content) > 0 {
			content = d.Content[0].Text
		}
		tokens := strings.Fields(strings.ToLower(content))
		tokenizedDocs = append(tokenizedDocs, tokens)
		for _, token := range tokens {
			if _, ok := vocab[token]; !ok {
				vocab[token] = idCounter
				idCounter++
			}
		}
	}

	// Convert tokenized docs to tfidf.Document
	var tfDocs []tfidf.Document
	for _, tokens := range tokenizedDocs {
		var ids []int
		for _, token := range tokens {
			ids = append(ids, vocab[token])
		}
		tfDocs = append(tfDocs, doc(ids))
	}

	tf := tfidf.New()
	for _, d := range tfDocs {
		tf.Add(d)
	}
	tf.CalculateIDF()

	return &BM25{
		tf:     tf,
		docs:   docs,
		vocab:  vocab,
		tfDocs: tfDocs,
	}, nil
}

// Retrieve performs a search using the BM25 algorithm.
func (b *BM25) Retrieve(ctx context.Context, query string, filters map[string]string, cfg types.BM25Config) ([]string, error) {
	queryTokens := strings.Fields(strings.ToLower(query))
	var queryIDs []int
	for _, token := range queryTokens {
		if id, ok := b.vocab[token]; ok {
			queryIDs = append(queryIDs, id)
		}
	}

	docScores := bm25.BM25(b.tf, doc(queryIDs), b.tfDocs, k1, b_param)

	sort.Sort(sort.Reverse(docScores))

	var results []string
	for _, score := range docScores {
		if len(results) >= cfg.TopK {
			break
		}
		doc := b.docs[score.ID]
		if matches(doc, filters) {
			if len(doc.Content) > 0 {
				results = append(results, doc.Content[0].Text)
			}
		}
	}
	return results, nil
}

func matches(doc *ai.Document, filters map[string]string) bool {
	if len(filters) == 0 {
		return true
	}
	for key, value := range filters {
		if metaValue, ok := doc.Metadata[key]; ok {
			if metaValueStr, ok := metaValue.(string); ok {
				if metaValueStr == value {
					continue // This filter key matches.
				}
			}
		}
		// If the key doesn't exist in metadata, or the value doesn't match, the doc is a mismatch.
		return false
	}
	// All filters matched.
	return true
}

func parseFrontMatter(fileContent []byte) (map[string]any, string) {
	const separator = "---\n"

	if !bytes.HasPrefix(fileContent, []byte(separator)) {
		return nil, string(fileContent)
	}

	parts := bytes.SplitN(fileContent, []byte(separator), 3)
	if len(parts) < 3 {
		return nil, string(fileContent)
	}

	var metadata map[string]any
	if err := yaml.Unmarshal(parts[1], &metadata); err != nil {
		fmt.Fprintf(os.Stderr, "could not parse front matter: %v, file content will be used as is", err)
		return nil, string(fileContent)
	}
	return metadata, strings.TrimSpace(string(parts[2]))
}

// loadDocuments reads all markdown files from a directory and returns them as strings.
func loadDocuments(path string) ([]*ai.Document, error) {
	var documents []*ai.Document
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			metadata, contentStr := parseFrontMatter(content)
			documents = append(documents, ai.DocumentFromText(contentStr, metadata))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return documents, nil
}