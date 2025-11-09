package pipeline_test

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
)

type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{
		Docs: []core.Doc{
			{Text: "mock document"},
		},
	}, nil
}

type mockRetrieverOptions struct{}

func (m *mockRetrieverOptions) ProviderName() string {
	return "mock-retriever"
}

func (m *mockRetrieverOptions) ProviderKind() core.Kind {
	return core.KindRetriever
}
