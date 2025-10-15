package providers

import (
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf"
)

func (s *Set) WithJSONSchemaParser() *Set {
	s.registrations = append(s.registrations, jsonschema.Register)
	return s
}

func (s *Set) WithRDFSchemaParser() *Set {
	s.registrations = append(s.registrations, rdf.Register)
	return s
}
