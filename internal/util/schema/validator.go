package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ValidateJSON validates a JSON string against a given schema string.
func ValidateJSON(schemaStr string, jsonStr string) error {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", strings.NewReader(schemaStr)); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	sch, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var v interface{}
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	if err := sch.Validate(v); err != nil {
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
