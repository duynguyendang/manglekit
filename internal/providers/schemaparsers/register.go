package schemaparsers

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf"
)

// Register registers all schema parser providers and the schema parser kind handler.
func Register(r *manglekit.Registry) {
	jsonschema.Register(r)
	rdf.Register(r)
}
