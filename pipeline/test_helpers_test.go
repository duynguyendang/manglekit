//go:build testhooks
// +build testhooks

package pipeline_test

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
)

// mockRetrieverOptions provides a dummy options struct for the mock retriever.
type mockRetrieverOptions struct{}

func (o *mockRetrieverOptions) ProviderName() string { return "mock-retriever" }
func (o *mockRetrieverOptions) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *mockRetrieverOptions) GetProviderOptions() any   { return o }

// mockRetriever is a mock implementation of core.Retriever for testing.
type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{Docs: []core.Doc{{Text: "mock document"}}}, nil
}
