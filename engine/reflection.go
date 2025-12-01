package engine

import (
	"fmt"
	"reflect"
	"strings"
)

// ToFacts converts a Go data structure into a slice of string-represented Mangle Datalog facts.
// It recursively traverses structs, maps, slices, and arrays.
//
// Format:
//   predicate(entityID, value)
//
// Nested fields are flattened using underscore delimiters (e.g., "address_city").
//
// Parameters:
//   - id: The unique identifier for the entity (e.g., "Req", "Output").
//   - input: The Go value to convert.
//
// Returns:
//   - A slice of strings, where each string is a Datalog fact (e.g., 'name("Req", "Alice")').
//   - An error if conversion fails.
func ToFacts(id string, input any) ([]string, error) {
	if input == nil {
		return nil, nil
	}
	var facts []string
	v := reflect.ValueOf(input)

	// Use empty path initially.
	if err := toFactsRecursive(id, "", v, &facts); err != nil {
		return nil, err
	}
	return facts, nil
}

// LabelsToFacts converts a slice of security label strings into Mangle Datalog facts.
// This is used for propagating taint information (e.g., "secret", "pii") into the policy engine.
//
// Format:
//   has_label("entityID", "label_value")
//
// Parameters:
//   - entityID: The unique identifier for the entity.
//   - labels: A slice of security label strings.
//
// Returns:
//   - A slice of Datalog fact strings.
//   - An error if conversion fails (unlikely, mostly wrapper around string formatting).
func LabelsToFacts(entityID string, labels []string) ([]string, error) {
	var facts []string
	for _, label := range labels {
		// Escape double quotes and backslashes in the label and entityID
		safeID := escapeString(entityID)
		safeLabel := escapeString(label)
		fact := fmt.Sprintf("has_label(\"%s\", \"%s\")", safeID, safeLabel)
		facts = append(facts, fact)
	}
	return facts, nil
}

// escapeString escapes special characters (backslashes and double quotes)
// to ensure the resulting string is a valid Mangle string constant.
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// toFactsRecursive traverses the reflection value tree and appends facts to the slice.
func toFactsRecursive(id, path string, v reflect.Value, facts *[]string) error {
	// Dereference pointers, skipping if nil.
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			structField := t.Field(i)

			// Skip unexported fields.
			if !structField.IsExported() {
				continue
			}

			// Determine the predicate name from the tag or field name.
			tag := structField.Tag.Get("mangle")
			fieldName := tag
			if fieldName == "" {
				fieldName = strings.ToLower(structField.Name)
			}

			newPath := fieldName
			if path != "" {
				newPath = path + "_" + fieldName
			}

			if err := toFactsRecursive(id, newPath, field, facts); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			// Assuming keys can be reasonably converted to strings for the predicate.
			// Flattening map keys: path_key(id, value)
			keyStr := fmt.Sprintf("%v", key.Interface())
			// Basic sanitization for keyStr to be part of predicate
			keyStr = strings.ReplaceAll(keyStr, " ", "_")
			keyStr = strings.ReplaceAll(keyStr, "-", "_")

			newPath := keyStr
			if path != "" {
				newPath = path + "_" + keyStr
			}

			if err := toFactsRecursive(id, newPath, val, facts); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			// For slice elements, generate facts using the same predicate.
			if err := toFactsRecursive(id, path, v.Index(i), facts); err != nil {
				return err
			}
		}
	default:
		// Base case for literal values.
		strVal := fmt.Sprintf("%v", v.Interface())

		// If path is empty (top-level primitive), use "value" default
		predicate := path
		if predicate == "" {
			predicate = "value"
		}

		safeID := escapeString(id)
		safeVal := escapeString(strVal)

		fact := fmt.Sprintf("%s(\"%s\", \"%s\")", predicate, safeID, safeVal)
		*facts = append(*facts, fact)
	}
	return nil
}
