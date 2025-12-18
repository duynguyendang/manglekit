package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
)

// Generate generates a JSON Schema definition from a Go Struct.
// It uses google/jsonschema-go/jsonschema for generation.
func Generate(v any) (string, error) {
	t := reflect.TypeOf(v)
	schema, err := jsonschema.ForType(t, &jsonschema.ForOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to generate schema: %w", err)
	}

	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	return string(schemaBytes), nil
}
