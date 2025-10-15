package localvec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"
	"gopkg.in/yaml.v3"
)

const (
	collectionName = "manglekit-localvec-collection"
)

func Register(r *manglekit.Registry) {
	r.Register("localvec", func(ctx context.Context, options any, deps manglekit.FactoryDeps) (any, error) {
		opts, ok := options.(core.LocalvecOptions)
		if !ok {
			return nil, fmt.Errorf("invalid options type, expected core.LocalvecOptions, got %T", options)
		}
		embedder, ok := deps["embedder"].(ai.Embedder)
		if !ok {
			return nil, fmt.Errorf("missing required dependency 'embedder' of type ai.Embedder")
		}
		return New(ctx, opts, embedder)
	})
	r.RegisterOptions("localvec", (*core.LocalvecOptions)(nil))
}

// LocalVecStore implements the core.VectorStore interface using Genkit's localvec.
type LocalVecStore struct {
	retriever ai.Retriever
	docStore  *localvec.DocStore
	generator *genkit.Genkit
	embedder  ai.Embedder
	cancel    context.CancelFunc
}

// Close shuts down the underlying Genkit generator.
func (l *LocalVecStore) Close(ctx context.Context) error {
	if l.cancel != nil {
		l.cancel()
	}
	return nil
}

// New creates a new LocalVecStore with explicit dependencies.
func New(ctx context.Context, opts core.LocalvecOptions, embedder ai.Embedder) (core.VectorStore, error) {
	if embedder == nil {
		return nil, fmt.Errorf("localvec: an embedder is required")
	}
	if opts.Path == "" {
		return nil, fmt.Errorf("localvec: 'path' parameter is required")
	}

	// Initialize Genkit internally. It needs a long-lived context that we manage
	// via the Close() method.
	gCtx, gCancel := context.WithCancel(ctx)
	g := genkit.Init(gCtx)

	docs, err := loadDocuments(opts.Path)
	if err != nil {
		gCancel()
		return nil, fmt.Errorf("localvec: failed to load documents: %w", err)
	}

	if err := localvec.Init(); err != nil {
		if !strings.Contains(err.Error(), "already initialized") {
			return nil, fmt.Errorf("localvec: failed to initialize localvec plugin: %w", err)
		}
	}

	retOpts := &ai.RetrieverOptions{Label: collectionName}
	docStore, retriever, err := localvec.DefineRetriever(g, collectionName, localvec.Config{Embedder: embedder}, retOpts)
	if err != nil {
		return nil, fmt.Errorf("localvec: failed to define retriever: %w", err)
	}

	// Index documents at startup using the context passed to the constructor.
	if len(docs) > 0 {
		if err := localvec.Index(ctx, docs, docStore); err != nil {
			gCancel()
			return nil, fmt.Errorf("localvec: failed to index documents: %w", err)
		}
	}

	return &LocalVecStore{
		retriever: retriever,
		docStore:  docStore,
		generator: g,
		embedder:  embedder,
		cancel:    gCancel,
	}, nil
}

// AddDocuments adds new documents to the local vector store.
func (l *LocalVecStore) AddDocuments(ctx context.Context, docs []core.Doc) error {
	aiDocs := make([]*ai.Document, len(docs))
	for i, d := range docs {
		aiDocs[i] = ai.DocumentFromText(d.Text, d.Meta)
	}

	if err := localvec.Index(ctx, aiDocs, l.docStore); err != nil {
		return fmt.Errorf("localvec: failed to index documents: %w", err)
	}
	return nil
}

// Search performs a vector search.
func (l *LocalVecStore) Search(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
	dRequest := ai.DocumentFromText(queryText, nil)
	retrieverOptions := &localvec.RetrieverOptions{K: topK}

	response, err := genkit.Retrieve(ctx, l.generator,
		ai.WithRetriever(l.retriever),
		ai.WithDocs(dRequest),
		ai.WithConfig(retrieverOptions),
	)
	if err != nil {
		return nil, fmt.Errorf("localvec: failed to retrieve documents: %w", err)
	}

	var results []core.Doc
	if response != nil {
		for _, doc := range response.Documents {
			if matches(doc, filter) {
				if len(doc.Content) > 0 {
					var docID string
					if id, ok := doc.Metadata["doc_id"].(string); ok {
						docID = id
					} else if source, ok := doc.Metadata["source"].(string); ok {
						base := filepath.Base(source)
						docID = strings.TrimSuffix(base, filepath.Ext(base))
					}
					source, _ := doc.Metadata["source"].(string)
					results = append(results, core.Doc{
						ID:     docID,
						Text:   doc.Content[0].Text,
						Source: source,
						URI:    source,
						Meta:   doc.Metadata,
					})
				}
			}
		}
	}
	return results, nil
}

// matches performs post-retrieval filtering.
func matches(doc *ai.Document, filters map[string]any) bool {
	if len(filters) == 0 {
		return true
	}
	for key, value := range filters {
		if metaValue, ok := doc.Metadata[key]; ok {
			if fmt.Sprintf("%v", metaValue) == fmt.Sprintf("%v", value) {
				continue
			}
		}
		return false
	}
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
		return nil, string(fileContent)
	}
	return metadata, strings.TrimSpace(string(parts[2]))
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
			if contentStr == "" {
				return nil
			}
			if metadata == nil {
				metadata = make(map[string]any)
			}
			if _, ok := metadata["doc_id"]; !ok {
				metadata["doc_id"] = info.Name()
			}
			metadata["source"] = filePath
			doc := ai.DocumentFromText(contentStr, metadata)
			documents = append(documents, doc)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return documents, nil
}
