//go:build !rdf

package knowledge

import "fmt"

// RDFLoader stub for when rdf build tag is not present.
type RDFLoader struct{}

// NewRDFLoader creates a new instance of RDFLoader stub.
func NewRDFLoader() *RDFLoader {
	return &RDFLoader{}
}

// Parse returns an error indicating RDF support is disabled.
func (l *RDFLoader) Parse(path string) ([]string, error) {
	return nil, fmt.Errorf("RDF support is not enabled; build with '-tags rdf' to enable")
}
