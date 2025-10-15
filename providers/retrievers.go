package providers

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
	inmemory "github.com/duynguyendang/manglekit/internal/providers/retrievers/inmemory"
	"github.com/duynguyendang/manglekit/retrieve"
)

func (s *Set) WithInMemoryRetriever() *Set {
	s.registrations = append(s.registrations, inmemory.Register)
	return s
}

func (s *Set) WithBM25Retriever() *Set {
	s.registrations = append(s.registrations, bm25.Register, func(r *manglekit.Registry) {
		r.RegisterOptions("bm25", (*retrieve.BM25Options)(nil))
	})
	return s
}

func (s *Set) WithDenseRetriever() *Set {
	s.registrations = append(s.registrations, dense.Register)
	return s
}

func (s *Set) WithHybridRetriever() *Set {
	s.registrations = append(s.registrations, hybrid.Register)
	return s
}
