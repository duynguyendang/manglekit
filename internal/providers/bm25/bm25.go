package bm25

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/firebase/genkit/go/ai"
	"github.com/go-nlp/bm25"
	"github.com/go-nlp/tfidf"
	"gopkg.in/yaml.v3"
)

const (
	k1      = 2.0
	b_param = 0.75
)

// BM25Options provides a type-safe way to configure the BM25 retriever, which
// performs keyword-based search.
type BM25Options struct {
	// Path is the file system path to a directory of documents that will be
	// indexed by the retriever for keyword search.
	Path string `yaml:"path" path:"resolve"`
	// TopK specifies the default number of documents to return if a different
	// limit is not specified in the retrieval request.
	TopK int `yaml:"topK"`
	// Logger is the logger to be used by the retriever. If nil, a default
	// logger will be used.
	Logger core.Logger `yaml:"-"`
}

func (o BM25Options) ProviderName() string { return "bm25" }
func (o BM25Options) ProviderKind() core.Kind   { return core.KindRetriever }

func Register(r *manglekit.Registry) {
	manglekit.Register(r, BM25Options{},
		func(ctx context.Context, deps diapi.RetrieverDeps, cfg BM25Options) (core.Retriever, error) {
			return New(cfg)
		},
	)
}

// tfidfDoc is a wrapper for ai.Document to implement the tfidf.Document interface.
type tfidfDoc struct {
	*ai.Document
	tokenIDs []int
	index    int
}

func (d tfidfDoc) IDs() []int { return d.tokenIDs }
func (d tfidfDoc) ID() int    { return d.index }

// BM25 implements the `retrieve.Retriever` interface, providing a classic sparse,
// keyword-based search using the Okapi BM25 algorithm. It operates by indexing a
// directory of documents in-memory, calculating term frequencies, and scoring
// documents based on the presence and frequency of query terms. This retriever
// is ideal for exact keyword matching and is often used as a component in a
// hybrid search system alongside a dense retriever.
type BM25 struct {
	tf     *tfidf.TFIDF
	docs   []*ai.Document
	vocab  map[string]int
	tfDocs []tfidf.Document
	topK   int
}

// New is the constructor for the BM25 retriever. It is the function registered
// with the MangleKit registry for the "bm25" provider name.
//
// The constructor performs the following steps:
//  1. Validates that a document path is provided in the options.
//  2. Loads all `.md` documents from the specified directory path.
//  3. Parses YAML front matter from each document to extract metadata.
//  4. Builds an in-memory vocabulary and tokenizes the documents.
//  5. Creates a TF-IDF model from the tokenized documents, which is a prerequisite
//     for the BM25 scoring algorithm.
//
// opts are the configuration options, requiring at least a `Path` to the
// directory of documents to be indexed.
// It returns an initialized `core.Retriever` or an error if the document
// path is invalid or document loading/parsing fails.
func New(opts BM25Options) (core.Retriever, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("path option is required for bm25 retriever")
	}
	topK := opts.TopK
	if topK == 0 {
		topK = 10 // default value
	}
	logger := opts.Logger
	if logger == nil {
		logger = obslogger.NewStdLogger()
	}

	docs, err := loadDocuments(opts.Path, logger)
	if err != nil {
		logger.Errorf("failed to load documents for BM25: %v", err)
		return nil, fmt.Errorf("failed to load documents for BM25: %w", err)
	}
	logger.Debugf("loaded %d documents from %s", len(docs), opts.Path)

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

// Retrieve performs a search against the in-memory document index using the BM25
// algorithm. It tokenizes the query, finds matching documents containing the
// query terms, scores them based on relevance, and returns the top K results
// sorted in descending order of score.
// This method satisfies the `retrieve.Retriever` interface.
//
// ctx is the context for the API call.
// req contains the query string and the number of results to return (`TopK`).
// It returns a `core.RetrieveResult` containing the ranked documents or an error if
// the retrieval fails.
func (b *BM25) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	queryTokens := strings.Fields(strings.ToLower(req.Query))
	var queryIDs []int
	for _, token := range queryTokens {
		if id, ok := b.vocab[token]; ok {
			queryIDs = append(queryIDs, id)
		}
	}

	docScores := bm25.BM25(b.tf, tfidfDoc{tokenIDs: queryIDs}, b.tfDocs, k1, b_param)
	sort.Sort(sort.Reverse(docScores))

	limit := req.TopK
	if limit <= 0 {
		limit = b.topK
	}

	var results []core.Doc
	for _, score := range docScores {
		if limit > 0 && len(results) >= limit {
			break
		}
		if score.Score <= 0 {
			continue
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
	return core.RetrieveResult{Docs: results}, nil
}

func parseFrontMatter(fileContent []byte, logger core.Logger) (map[string]any, []byte) {
	const separator = "---\n"
	if !bytes.HasPrefix(fileContent, []byte(separator)) {
		return nil, fileContent
	}
	parts := bytes.SplitN(fileContent, []byte(separator), 3)
	if len(parts) < 3 {
		return nil, fileContent
	}
	var metadata map[string]any
	if err := yaml.Unmarshal(parts[1], &metadata); err != nil {
		if logger != nil {
			logger.Warnf("could not parse front matter: %v, file content will be used as is", err)
		}
		return nil, fileContent
	}
	return metadata, parts[2]
}

func loadDocuments(path string, logger core.Logger) ([]*ai.Document, error) {
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
			metadata, contentBytes := parseFrontMatter(fileContent, logger)
			contentStr := string(contentBytes)
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
