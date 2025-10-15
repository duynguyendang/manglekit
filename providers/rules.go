package providers

import (
	"github.com/duynguyendang/manglekit/internal/providers/mangle"
)

func (s *Set) WithMangleRules() *Set {
	s.registrations = append(s.registrations, mangle.Register)
	return s
}
