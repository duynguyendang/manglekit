package schema

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// Generate generates a JSON Schema definition from a Go Struct.
func Generate(v any) (string, error) {
	r := new(jsonschema.Reflector)
	schema := r.Reflect(v)

	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	return string(schemaBytes), nil
}
