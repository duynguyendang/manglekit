package engine

import (
	"fmt"
	"reflect"
	"strings"
)

// ToFacts converts a Go data structure into a slice of string-represented Datalog facts.
// It generates facts in the format: predicate(entityID, value).
// For nested structures, field names are flattened with underscores.
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

// LabelsToFacts converts a slice of security labels into Mangle facts.
// It handles escaping to prevent Datalog injection and ensure valid parsing.
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

// escapeString escapes special characters for Mangle string constants.
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// toFactsRecursive is the recursive helper that traverses the data structure.
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
