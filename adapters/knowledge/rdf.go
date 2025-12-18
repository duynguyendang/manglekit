//go:build rdf

package knowledge

import (
	"fmt"
	"os"
	"strings"

	"github.com/knakk/rdf"
)

// RDFLoader loads RDF data using an external library (knakk/rdf).
// Supported formats: Turtle (.ttl), RDF/XML (.rdf, .xml).
type RDFLoader struct{}

// NewRDFLoader creates a new instance of RDFLoader.
func NewRDFLoader() *RDFLoader {
	return &RDFLoader{}
}

// Parse loads RDF from a file path and converts it to Manglekit facts.
// Format: triple("Sub", "Pred", "Obj")
func (l *RDFLoader) Parse(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open RDF file: %w", err)
	}
	defer f.Close()

	// Detect format
	var format rdf.Format
	if strings.HasSuffix(strings.ToLower(path), ".xml") || strings.HasSuffix(strings.ToLower(path), ".rdf") {
		format = rdf.RDFXML
	} else {
		format = rdf.Turtle
	}

	decoder := rdf.NewTripleDecoder(f, format)
	triples, err := decoder.DecodeAll()
	if err != nil {
		return nil, fmt.Errorf("failed to decode RDF: %w", err)
	}

	var facts []string
	for _, t := range triples {
		sub := l.cleanTerm(t.Subj)
		pred := l.cleanTerm(t.Pred)
		obj := l.cleanTerm(t.Obj)

		facts = append(facts, fmt.Sprintf("triple(\"%s\", \"%s\", \"%s\")", l.escape(sub), l.escape(pred), l.escape(obj)))
	}

	return facts, nil
}

func (l *RDFLoader) cleanTerm(term rdf.Term) string {
	switch v := term.(type) {
	case rdf.IRI:
		return strings.TrimSuffix(strings.TrimPrefix(v.String(), "<"), ">")
	case rdf.Literal:
		// Return the raw value string, ignoring language/datatype for simplicity in the 'triple' predicate
		// detailed handling can be added if 'triple' supports more arguments
		return v.String()
	default:
		return v.String()
	}
}

func (l *RDFLoader) escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
