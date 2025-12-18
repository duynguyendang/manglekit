package schema

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// ValidateJSON validates a JSON string against a given schema string.
// It uses google/jsonschema-go/jsonschema for validation.
func ValidateJSON(schemaStr string, jsonStr string) error {
	var s jsonschema.Schema
	if err := json.Unmarshal([]byte(schemaStr), &s); err != nil {
		return fmt.Errorf("failed to parse schema json: %w", err)
	}

	resolved, err := s.Resolve(nil)
	if err != nil {
		return fmt.Errorf("failed to resolve schema: %w", err)
	}

	var v any
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	if err := resolved.Validate(v); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// ValidateStruct validates a struct against a given schema string.
func ValidateStruct(schemaStr string, input any) error {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input struct: %w", err)
	}

	return ValidateJSON(schemaStr, string(jsonBytes))
}
