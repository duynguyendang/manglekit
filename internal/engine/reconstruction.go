package engine

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/duynguyendang/manglekit-wip/internal/engine/parse"
)

// FactIndex organizes facts for O(1) lookup during reconstruction.
// Map: SubjectID -> Predicate/Key -> []Value
type FactIndex map[string]map[string][]string

// FromFacts reconstructs a struct of type T from a list of Datalog fact strings.
func FromFacts[T any](rootID string, facts []string) (*T, error) {
	// 1. Build Index
	index, err := buildFactIndex(facts)
	if err != nil {
		return nil, fmt.Errorf("failed to index facts: %w", err)
	}

	// 2. Instantiate Target
	var result T
	val := reflect.ValueOf(&result).Elem()

	// 3. Hydrate recursively
	if err := hydrate(val, rootID, index); err != nil {
		return nil, err
	}

	return &result, nil
}

func buildFactIndex(facts []string) (FactIndex, error) {
	index := make(FactIndex)

	for _, f := range facts {
		// Simple parser for standard predicates: pred("arg1", "arg2", ...)
		pred, args, err := parse.ParseAtomContent(f)
		if err != nil {
			continue // Skip malformed facts
		}

		if len(args) < 2 { continue }

		var subject, key, value string

		switch pred {
		case "json_str", "json_num", "json_bool", "json_link":
			if len(args) < 3 { continue }
			subject = args[0]
			key = args[1]
			value = args[2]
		case "triple":
			if len(args) < 3 { continue }
			subject = args[0]
			key = args[1] // Predicate acts as key
			value = args[2]
		default:
			// Handle custom predicates mapped via mangle tags
			// heuristic: pred(Subject, Value)
			if len(args) == 2 {
				subject = args[0]
				key = pred
				value = args[1]
			} else {
				continue
			}
		}

		if _, ok := index[subject]; !ok {
			index[subject] = make(map[string][]string)
		}
		index[subject][key] = append(index[subject][key], value)
	}
	return index, nil
}

func hydrate(val reflect.Value, currentID string, index FactIndex) error {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct, got %v", val.Kind())
	}

	typ := val.Type()
	nodeData := index[currentID] // Data for this node

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !field.IsExported() { continue }

		// Get mapping key
		tag := field.Tag.Get("mangle")
		if tag == "-" { continue }
		if tag == "" { tag = strings.ToLower(field.Name) }

		// Look up values
		values, found := nodeData[tag]
		if !found { continue }

		// Handle Slice
		if fieldVal.Kind() == reflect.Slice {
			// For each value found, append to slice
			elemType := fieldVal.Type().Elem()
			slice := reflect.MakeSlice(fieldVal.Type(), 0, len(values))

			for _, v := range values {
				newElem := reflect.New(elemType).Elem()
				// Check if element is struct (requires recursion)
				if isStructOrPtr(elemType) {
					// v is the ChildID in json_link
					if err := hydrate(newElem, v, index); err != nil {
						return err
					}
				} else {
					if err := setPrimitive(newElem, v); err != nil {
						return fmt.Errorf("field %s: %w", field.Name, err)
					}
				}
				slice = reflect.Append(slice, newElem)
			}
			fieldVal.Set(slice)
			continue
		}

		// Handle Single Value (take the first one if multiple exist)
		rawVal := values[0]

		if isStructOrPtr(fieldVal.Type()) {
			// Recurse: rawVal is the ChildID
			if err := hydrate(fieldVal, rawVal, index); err != nil {
				return err
			}
		} else {
			if err := setPrimitive(fieldVal, rawVal); err != nil {
				return fmt.Errorf("field %s: %w", field.Name, err)
			}
		}
	}
	return nil
}

func setPrimitive(v reflect.Value, s string) error {
	s = strings.Trim(s, "\"") // Remove quotes if present

	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil { return err }
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil { return err }
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil { return err }
		v.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil { return err }
		v.SetBool(b)
	}
	return nil
}

func isStructOrPtr(t reflect.Type) bool {
	if t.Kind() == reflect.Struct { return true }
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct { return true }
	return false
}
