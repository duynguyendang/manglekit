package providers

import (
	"github.com/duynguyendang/manglekit/internal/providers/llm"
)

func (s *Set) WithGoogleLLM() *Set {
	s.registrations = append(s.registrations, llm.RegisterGoogle)
	return s
}

func (s *Set) WithOpenAI() *Set {
	s.registrations = append(s.registrations, llm.RegisterOpenAI)
	return s
}
