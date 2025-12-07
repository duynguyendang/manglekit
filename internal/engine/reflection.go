package engine

import (
	"fmt"
	"reflect"
	"strings"
)

// ToFacts converts a Go data structure into Mangle Datalog facts.
// It is the entry point for turning runtime objects into logic predicates.
func ToFacts(id string, input any) ([]string, error) {
	if input == nil {
		return nil, nil
	}
	var facts []string
	v := reflect.ValueOf(input)

	// Track visited pointers to prevent infinite recursion (Cycles)
	visited := make(map[uintptr]bool)

	if err := toFactsRecursive(id, "", v, &facts, visited); err != nil {
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

func toFactsRecursive(id, path string, v reflect.Value, facts *[]string, visited map[uintptr]bool, args ...string) error {
	// 1. Dereference Pointers & Interfaces
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		// Cycle Detection for Pointers
		if v.Kind() == reflect.Ptr {
			ptr := v.Pointer()
			if visited[ptr] {
				return nil // Cycle detected: stop this branch silently
			}
			visited[ptr] = true
		}
		v = v.Elem()
	}

	// 2. Switch on Kind
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			structField := t.Field(i)

			// Skip unexported fields
			if !structField.IsExported() {
				continue
			}

			// Determine Field Name (Tag Priority: mangle > json > struct name)
			tag := structField.Tag.Get("mangle")
			fieldName := tag
			if fieldName == "" {
				jsonTag := structField.Tag.Get("json")
				if jsonTag != "" && jsonTag != "-" {
					parts := strings.Split(jsonTag, ",")
					fieldName = parts[0]
				}
			}
			if fieldName == "" {
				fieldName = strings.ToLower(structField.Name)
			}

			// Handle Embedded (Anonymous) Fields
			// Strategy: Flatten if anonymous AND untagged
			newPath := path
			isAnonymousUntagged := structField.Anonymous && tag == "" && structField.Tag.Get("json") == ""

			if !isAnonymousUntagged {
				if newPath != "" {
					newPath = newPath + "_" + fieldName
				} else {
					newPath = fieldName
				}
			}

			if err := toFactsRecursive(id, newPath, field, facts, visited, args...); err != nil {
				return err
			}
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			keyStr := fmt.Sprintf("%v", key.Interface())

			// Append key to args for the next level
			newArgs := make([]string, len(args)+1)
			copy(newArgs, args)
			newArgs[len(args)] = keyStr

			if err := toFactsRecursive(id, path, val, facts, visited, newArgs...); err != nil {
				return err
			}
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			// Treat Slice as Set (Ignore index). Recursively process elements.
			if err := toFactsRecursive(id, path, v.Index(i), facts, visited, args...); err != nil {
				return err
			}
		}

	default:
		// primitive handling
		generatePrimitiveFact(id, path, v, facts, args...)
	}
	return nil
}

// generatePrimitiveFact creates the final Datalog string: predicate("id", "arg", value).
func generatePrimitiveFact(id, path string, v reflect.Value, facts *[]string, args ...string) {
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
		strVal = fmt.Sprintf("%g", v.Float()) // Use %g to preserve significant digits
		isNumeric = true
	case reflect.Bool:
		strVal = fmt.Sprintf("%v", v.Bool())
	default:
		strVal = fmt.Sprintf("%v", v.Interface())
	}

	predicate := path
	if predicate == "" {
		predicate = "value"
	}

	// Helper to escape strings (Must ensure this exists in file)
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
