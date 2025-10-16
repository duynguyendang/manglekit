package providers

import "github.com/duynguyendang/manglekit"

// Set is a builder that collects provider registration functions.
type Set struct {
	registrations []func(r *manglekit.Registry)
}

// NewSet returns a new, empty Set.
func NewSet() *Set {
	return &Set{}
}

// With adds a registration function to the set.
func (s *Set) With(registration func(r *manglekit.Registry)) *Set {
	s.registrations = append(s.registrations, registration)
	return s
}

// ApplyTo applies the collected registration functions to the given registry.
func (s *Set) ApplyTo(r *manglekit.Registry) {
	for _, reg := range s.registrations {
		reg(r)
	}
}
