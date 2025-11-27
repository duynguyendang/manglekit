package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/knakk/rdf"
)

// LoadFromPath loads RDF data from a Turtle file and converts it to Mangle facts.
func LoadFromPath(path string) ([]string, error) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open knowledge file %s: %w", path, err)
	}
	defer f.Close()

	// Initialize decoder
	decoder := rdf.NewTripleDecoder(f, rdf.Turtle)

	// Decode all triples
	triples, err := decoder.DecodeAll()
	if err != nil {
		return nil, fmt.Errorf("failed to decode RDF from %s: %w", path, err)
	}

	var facts []string
	for _, t := range triples {
		fact, err := tripleToFact(t)
		if err != nil {
			// Log warning? For now we skip invalid conversions
			continue
		}
		facts = append(facts, fact)
	}
	return facts, nil
}

// tripleToFact converts an RDF triple to a Mangle Datalog atom string.
// Format: predicate("subject", "object")
func tripleToFact(t rdf.Triple) (string, error) {
	predName, err := cleanPredicate(t.Pred)
	if err != nil {
		return "", err
	}

	subName := cleanTerm(t.Subj)
	objName := cleanTerm(t.Obj)

	// Escape strings for Datalog
	subName = escapeString(subName)
	objName = escapeString(objName)

	return fmt.Sprintf("%s(\"%s\", \"%s\")", predName, subName, objName), nil
}

// cleanPredicate extracts the local name from the URI and converts it to snake_case.
func cleanPredicate(term rdf.Term) (string, error) {
	uri, ok := term.(rdf.IRI)
	if !ok {
		return "", fmt.Errorf("predicate must be an IRI, got %T", term)
	}

	// IRI.String() might return raw string or <string>.
	// We strip < > just in case.
	raw := strings.TrimSuffix(strings.TrimPrefix(uri.String(), "<"), ">")

	name := extractLocalName(raw)
	return toSnakeCase(name), nil
}

// cleanTerm extracts the local name from a URI or returns the literal value.
func cleanTerm(term rdf.Term) string {
	switch v := term.(type) {
	case rdf.IRI:
		raw := strings.TrimSuffix(strings.TrimPrefix(v.String(), "<"), ">")
		return extractLocalName(raw)
	case rdf.Literal:
		val, err := v.Typed()
		if err == nil {
			return fmt.Sprintf("%v", val)
		}
		// Fallback to String() but strip quotes and datatype
		// Example: "value"^^<type>
		str := v.String()
		if idx := strings.LastIndex(str, "^^"); idx != -1 {
			str = str[:idx]
		}
		// Strip surrounding quotes if present
		if len(str) >= 2 && strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"") {
			str = str[1 : len(str)-1]
		}
		return str
	default:
		return term.String()
	}
}

// extractLocalName gets the last part of a URI (after / or #).
func extractLocalName(uri string) string {
	// Handle fragment
	if idx := strings.LastIndex(uri, "#"); idx != -1 {
		return uri[idx+1:]
	}
	// Handle path
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")

// toSnakeCase converts CamelCase to snake_case.
func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// escapeString escapes special characters for Datalog strings.
func escapeString(s string) string {
	// Basic escaping for quotes and backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// IsKnowledgeFile checks if the file is a supported knowledge file (e.g. .ttl)
func IsKnowledgeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".ttl"
}
