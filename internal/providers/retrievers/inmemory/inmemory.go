package inmemory

import (
	"errors"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/retrieve"
)

// InMemoryRetriever is a simple, thread-safe in-memory document store that is
// useful for testing or small-scale applications. It implements both the
// retrieve.Retriever and retrieve.Updatable interfaces.
type InMemoryRetriever struct {
	mu   sync.RWMutex
	docs map[string]core.Doc // Use a map for efficient upserts by ID.
}

// New creates a new InMemoryRetriever. It is the constructor function registered
// with the MangleKit registry for the "inmemory" retriever.
//
// opts can contain an initial set of documents to populate the retriever with.
// It returns an initialized retriever or an error if any document is missing an ID.
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

	return &InMemoryRetriever{
		docs: docMap,
	}, nil
}

// Retrieve returns documents from the in-memory store. For simplicity, this
// implementation does not perform any searching or filtering based on the query;
// it returns all stored documents up to the requested TopK limit.
// It satisfies the retrieve.Retriever interface.
func (r *InMemoryRetriever) Retrieve(req retrieve.Request) (retrieve.Result, error) {
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
// the same ID. This operation is thread-safe.
// It satisfies the retrieve.Updatable interface.
func (r *InMemoryRetriever) Upsert(docs []core.Doc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, doc := range docs {
		if doc.ID == "" {
			return errors.New("cannot upsert document without an ID")
		}
		r.docs[doc.ID] = doc
	}
	fmt.Printf("Upserted %d documents. Total documents: %d\n", len(docs), len(r.docs))
	return nil
}

// Replace clears all existing documents in the store and replaces them with a
// new set. This operation is thread-safe.
// It satisfies the retrieve.Updatable interface.
func (r *InMemoryRetriever) Replace(docs []core.Doc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.docs = make(map[string]core.Doc)
	for _, doc := range docs {
		if doc.ID == "" {
			return errors.New("cannot replace with a document that has no ID")
		}
		r.docs[doc.ID] = doc
	}
	fmt.Printf("Replaced all documents. New total: %d\n", len(r.docs))
	return nil
}

func init() {
	manglekit.RegisterRetriever("in-memory", New)
}
