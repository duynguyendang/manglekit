// Package retrieval provides BM25 retrieval functionality.
package retrieval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-nlp/bm25"
	"github.com/go-nlp/tfidf"
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
	tf      *tfidf.TFIDF
	docs    []string
	vocab   map[string]int
	tfDocs  []tfidf.Document
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
		tokens := strings.Fields(strings.ToLower(d))
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
func (b *BM25) Retrieve(ctx context.Context, query string, cfg types.BM25Config) ([]string, error) {
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
	for i := 0; i < len(docScores) && i < cfg.TopK; i++ {
		results = append(results, b.docs[docScores[i].ID])
	}

	return results, nil
}

// loadDocuments reads all markdown files from a directory and returns them as strings.
func loadDocuments(path string) ([]string, error) {
	var documents []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			documents = append(documents, string(content))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return documents, nil
}