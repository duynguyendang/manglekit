package providers

import (
	"github.com/duynguyendang/manglekit/internal/embedders/google"
	"github.com/duynguyendang/manglekit/internal/embedders/openai"
)

func (s *Set) WithGoogleEmbedder() *Set {
	s.registrations = append(s.registrations, google.Register)
	return s
}

func (s *Set) WithOpenAIEmbedder() *Set {
	s.registrations = append(s.registrations, openai.Register)
	return s
}
