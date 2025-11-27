package retrieve

import (
	"context"

	"github.com/duynguyendang/manglekit/v1/core"
)

// Updatable defines the interface for retrievers that support runtime
// modification of their document index. This is useful for systems where the
// knowledge base can change without restarting the application. A component that
// implements this interface must also embed the base Retriever interface.
type Updatable interface {
	core.Retriever
	// Upsert adds new documents or updates existing ones in the index. If a
	// document with the same ID already exists in the index, it should be
	// replaced with the new version.
	//
	// docs is a slice of documents to add or update in the retriever's index.
	// It returns an error if the upsert operation fails.
	Upsert(ctx context.Context, docs []core.Doc) error
	// Replace clears the entire existing index and replaces it with a new set
	// of documents. This is a destructive operation useful for completely
	// refreshing the retriever's knowledge base.
	//
	// docs is the new, complete slice of documents to populate the index with.
	// It returns an error if the replacement operation fails.
	Replace(ctx context.Context, docs []core.Doc) error
}
