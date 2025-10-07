package bm25

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/go-nlp/bm25"
	"github.com/go-nlp/tfidf"
	"gopkg.in/yaml.v3"
)

const (
	k1      = 2.0
	b_param = 0.75
)

func init() {
	manglekit.RegisterRetriever("bm25", New)
}

// tfidfDoc is a wrapper for ai.Document to implement the tfidf.Document interface.
type tfidfDoc struct {
	*ai.Document
	tokenIDs []int
	index    int
}

func (d tfidfDoc) IDs() []int { return d.tokenIDs }
func (d tfidfDoc) ID() int    { return d.index }

// BM25 implements the retrieve.Retriever interface providing a sparse, keyword-based
// search using the Okapi BM25 algorithm. It indexes a directory of markdown
// documents and performs term-based retrieval.
type BM25 struct {
	tf     *tfidf.TFIDF
	docs   []*ai.Document
	vocab  map[string]int
	tfDocs []tfidf.Document
	topK   int
}

// New creates a new BM25 retriever. It is the constructor function registered
// with the MangleKit registry for the "bm25" retriever.
//
// The function performs the following steps:
// 1. Validates that a document path is provided in the options.
// 2. Loads all .md documents from the specified path.
// 3. Builds a vocabulary and tokenizes the documents.
// 4. Creates a TF-IDF model from the documents, which is used by the BM25 algorithm.
//
// opts are the configuration options, requiring at least a 'Path' to the documents.
// It returns an initialized BM25 retriever or an error if setup fails.
func New(opts retrieve.BM25Options) (retrieve.Retriever, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("path option is required for bm25 retriever")
	}
	topK := opts.TopK
	if topK == 0 {
		topK = 10 // default value
	}

	docs, err := loadDocuments(opts.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load documents for BM25: %w", err)
	}

	vocab := make(map[string]int)
	var tfDocs []tfidf.Document
	idCounter := 0
	for i, d := range docs {
		content := ""
		if len(d.Content) > 0 {
			content = d.Content[0].Text
		}
		tokens := strings.Fields(strings.ToLower(content))
		var tokenIDs []int
		for _, token := range tokens {
			if _, ok := vocab[token]; !ok {
				vocab[token] = idCounter
				idCounter++
			}
			tokenIDs = append(tokenIDs, vocab[token])
		}
		tfDocs = append(tfDocs, tfidfDoc{Document: d, tokenIDs: tokenIDs, index: i})
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
		topK:   topK,
	}, nil
}

// Retrieve performs a search against the indexed documents using the BM25 algorithm.
// It tokenizes the query, finds matching documents, scores them, and returns
// the top K results sorted by relevance.
// This method satisfies the core.Retriever interface.
//
// req contains the query string and the number of results to return (TopK).
// It returns a retrieve.Result containing the ranked documents or an error.
func (b *BM25) Retrieve(req retrieve.Request) (retrieve.Result, error) {
	queryTokens := strings.Fields(strings.ToLower(req.Query))
	var queryIDs []int
	for _, token := range queryTokens {
		if id, ok := b.vocab[token]; ok {
			queryIDs = append(queryIDs, id)
		}
	}

	docScores := bm25.BM25(b.tf, tfidfDoc{tokenIDs: queryIDs}, b.tfDocs, k1, b_param)
	sort.Sort(sort.Reverse(docScores))

	var results []core.Doc
	for _, score := range docScores {
		if len(results) >= req.TopK {
			break
		}
		doc := b.docs[score.ID]
		if len(doc.Content) > 0 {
			var docID string
			if id, ok := doc.Metadata["doc_id"].(string); ok {
				docID = id
			} else {
				docID = fmt.Sprintf("doc-%d", score.ID) // Fallback
			}
			source, _ := doc.Metadata["source"].(string)
			// The score is not part of retrieve.Doc, but we can add it to metadata
			// if needed by downstream components like the reranker.
			doc.Metadata["score"] = float64(score.Score)
			results = append(results, core.Doc{
				ID:     docID,
				Text:   doc.Content[0].Text,
				Source: source,
				URI:    source,
				Meta:   doc.Metadata,
			})
		}
	}
	return retrieve.Result{Docs: results}, nil
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
	return metadata, string(parts[2])
}

func loadDocuments(path string) ([]*ai.Document, error) {
	var documents []*ai.Document
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			metadata, contentStr := parseFrontMatter(fileContent)
			trimmedContent := strings.TrimSpace(contentStr)
			if trimmedContent == "" {
				return nil
			}
			if metadata == nil {
				metadata = make(map[string]any)
			}
			metadata["doc_id"] = info.Name()
			metadata["source"] = filePath // Add source for citation
			doc := ai.DocumentFromText(trimmedContent, metadata)
			documents = append(documents, doc)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return documents, nil
}