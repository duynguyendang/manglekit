// Package inmemory provides a basic, thread-safe in-memory retriever that is
// useful for testing, demos, or small-scale applications where the entire
// document set can fit comfortably in memory.
package inmemory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/retrieve"
)

// InMemoryRetriever is a simple, thread-safe in-memory document store. It uses
// a map for efficient O(1) lookups, upserts, and deletes by document ID.
// It implements both the `retrieve.Retriever` and `retrieve.Updatable` interfaces,
// making it a fully functional component for dynamic applications.
type InMemoryRetriever struct {
	mu   sync.RWMutex
	docs map[string]core.Doc // Use a map for efficient operations by document ID.
	log  core.Logger
}

func Register(r *manglekit.Registry) {
	r.RegisterRetriever("in-memory", func(ctx context.Context, options any, deps manglekit.FactoryDeps) (retrieve.Retriever, error) {
		var opts retrieve.InMemoryOptions
		if options != nil {
			if typedOpts, ok := options.(*retrieve.InMemoryOptions); ok {
				opts = *typedOpts
			} else {
				return nil, fmt.Errorf("invalid options type, expected *retrieve.InMemoryOptions, got %T", options)
			}
		}
		return New(opts)
	})
	r.RegisterOptions("in-memory", (*retrieve.InMemoryOptions)(nil))
}

// New is the constructor for the InMemoryRetriever. It is registered with the
// MangleKit registry for the "in-memory" provider name.
//
// opts can contain an initial set of documents to populate the retriever with.
// It returns an initialized `retrieve.Retriever` or an error if any provided
// document is missing a required ID.
func New(opts retrieve.InMemoryOptions) (retrieve.Retriever, error) {
	docMap := make(map[string]core.Doc)
	// opts.Documents can be nil if the retriever is initialized empty.
	if opts.Documents != nil {
		for _, doc := range opts.Documents {
			if doc.ID == "" {
				return nil, errors.New("all documents for InMemoryRetriever must have a non-empty ID")
			}
			docMap[doc.ID] = doc
		}
	}

	log := opts.Logger
	if log == nil {
		log = obslogger.NewStdLogger().With("component", "retriever", "provider", "in-memory")
	}

	return &InMemoryRetriever{
		docs: docMap,
		log:  log,
	}, nil
}

// Retrieve returns documents from the in-memory store. For simplicity, this
// basic implementation does not perform any searching or filtering based on the
// query text; it returns all stored documents, respecting the requested `TopK`
// limit. This method is thread-safe.
//
// It satisfies the `retrieve.Retriever` interface.
func (r *InMemoryRetriever) Retrieve(ctx context.Context, req retrieve.Request) (retrieve.Result, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resultDocs := make([]core.Doc, 0, len(r.docs))
	for _, doc := range r.docs {
		resultDocs = append(resultDocs, doc)
	}

	// Respect the TopK parameter if provided.
	if req.TopK > 0 && len(resultDocs) > req.TopK {
		resultDocs = resultDocs[:req.TopK]
	}

	return retrieve.Result{Docs: resultDocs}, nil
}

// Upsert adds new documents to the store or updates existing ones if they share
// the same ID. This operation is thread-safe, using a write lock to prevent
// concurrent access during modification.
//
// It satisfies the `retrieve.Updatable` interface.
func (r *InMemoryRetriever) Upsert(ctx context.Context, docs []core.Doc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, doc := range docs {
		if doc.ID == "" {
			return errors.New("cannot upsert document without an ID")
		}
		r.docs[doc.ID] = doc
	}
	r.log.Infof("in-memory retriever upsert", "count", len(docs), "total", len(r.docs))
	return nil
}

// Replace clears all existing documents in the store and replaces them with a
// new set. This is a destructive operation that is useful for completely
// refreshing the knowledge base. This operation is thread-safe.
//
// It satisfies the `retrieve.Updatable` interface.
func (r *InMemoryRetriever) Replace(ctx context.Context, docs []core.Doc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.docs = make(map[string]core.Doc)
	for _, doc := range docs {
		if doc.ID == "" {
			return errors.New("cannot replace with a document that has no ID")
		}
		r.docs[doc.ID] = doc
	}
	r.log.Infof("in-memory retriever replace", "total", len(r.docs))
	return nil
}
