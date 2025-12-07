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
//
//	predicate(entityID, value)
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
//
//	has_label("entityID", "label_value")
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
	if len(labels) > 0 {
		facts = make([]string, 0, len(labels))
	}

	safeID := escapeString(entityID)

	for _, label := range labels {
		var sb strings.Builder
		sb.WriteString("has_label(\"")
		sb.WriteString(safeID)
		sb.WriteString("\", \"")
		sb.WriteString(escapeString(label))
		sb.WriteString("\")")
		facts = append(facts, sb.String())
	}
	return facts, nil
}

// escapeString escapes special characters (backslashes and double quotes)
// to ensure the resulting string is a valid Mangle string constant.
func escapeString(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '\\' || b == '"' {
			sb.WriteByte('\\')
		}
		sb.WriteByte(b)
	}
	return sb.String()
}

// toFactsRecursive traverses the reflection value tree and appends facts to the slice.
func toFactsRecursive(id, path string, v reflect.Value, facts *[]string, args ...string) error {
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
			// Priority: mangle tag > json tag > field name
			tag := structField.Tag.Get("mangle")
			fieldName := tag
			if fieldName == "" {
				jsonTag := structField.Tag.Get("json")
				if jsonTag != "" && jsonTag != "-" {
					// json tag often looks like "name,omitempty"
					parts := strings.Split(jsonTag, ",")
					fieldName = parts[0]
				}
			}
			if fieldName == "" {
				fieldName = strings.ToLower(structField.Name)
			}

			newPath := fieldName
			if path != "" {
				newPath = path + "_" + fieldName
			}

			if err := toFactsRecursive(id, newPath, field, facts, args...); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			keyStr := fmt.Sprintf("%v", key.Interface())

			// For maps, we treat the key as an additional argument to the predicate.
			newArgs := append([]string{}, args...)
			newArgs = append(newArgs, keyStr)

			if err := toFactsRecursive(id, path, val, facts, newArgs...); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := toFactsRecursive(id, path, v.Index(i), facts, args...); err != nil {
				return err
			}
		}
	default:
		// Base case for literal values.
		var strVal string
		var isNumeric bool

		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			strVal = fmt.Sprintf("%d", v.Int())
			isNumeric = true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			strVal = fmt.Sprintf("%d", v.Uint())
			isNumeric = true
		case reflect.Float32, reflect.Float64:
			strVal = fmt.Sprintf("%f", v.Float())
			isNumeric = true
		default:
			strVal = fmt.Sprintf("%v", v.Interface())
		}

		// If path is empty (top-level primitive), use "value" default
		predicate := path
		if predicate == "" {
			predicate = "value"
		}

		safeID := escapeString(id)

		var sb strings.Builder
		sb.WriteString(predicate)
		sb.WriteByte('(')
		sb.WriteByte('"')
		sb.WriteString(safeID)
		sb.WriteByte('"')

		for _, arg := range args {
			sb.WriteString(", \"")
			sb.WriteString(escapeString(arg))
			sb.WriteByte('"')
		}

		sb.WriteString(", ")
		if isNumeric {
			sb.WriteString(strVal)
		} else {
			sb.WriteByte('"')
			sb.WriteString(escapeString(strVal))
			sb.WriteByte('"')
		}
		sb.WriteByte(')')

		*facts = append(*facts, sb.String())
	}
	return nil
}
