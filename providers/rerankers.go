package providers

import (
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
)

func (s *Set) WithCosineReranker() *Set {
	s.registrations = append(s.registrations, cosine.Register)
	return s
}
