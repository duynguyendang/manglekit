package reflection

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/mangle/ast"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ToFacts converts any Go struct/map/interface into a slice of Mangle Atoms.
// id: The entity identifier (e.g., "request_123").
// input: The struct/map/value to traverse.
func ToFacts(id string, input any) ([]ast.Atom, error) {
	if input == nil {
		return []ast.Atom{}, nil
	}
	val := reflect.ValueOf(input)
	return walk(id, val, "")
}

func walk(id string, val reflect.Value, prefix string) ([]ast.Atom, error) {
	var facts []ast.Atom

	// Auto-Dereference (Pointers)
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil, nil // Stop immediately, omit the fact.
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			tag := field.Tag.Get("mangle")

			if tag == "-" {
				continue // Skip field
			}

			fieldName := tag
			if fieldName == "" {
				fieldName = toSnakeCase(field.Name)
			}

			newPrefix := fieldName
			if prefix != "" {
				newPrefix = prefix + "." + fieldName
			}

			fieldFacts, err := walk(id, val.Field(i), newPrefix)
			if err != nil {
				return nil, err
			}
			facts = append(facts, fieldFacts...)
		}

	case reflect.Map:
		for _, key := range val.MapKeys() {
			// Convert non-string keys to string representation.
			keyStr := fmt.Sprintf("%v", key.Interface())
			newPrefix := keyStr
			if prefix != "" {
				newPrefix = prefix + "." + newPrefix
			}

			valueFacts, err := walk(id, val.MapIndex(key), newPrefix)
			if err != nil {
				return nil, err
			}
			facts = append(facts, valueFacts...)
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			// Do not add array indices to the path.
			// The prefix remains the same for all elements.
			elemFacts, err := walk(id, val.Index(i), prefix)
			if err != nil {
				return nil, err
			}
			facts = append(facts, elemFacts...)
		}

	// Primitive Values (The Leaf Nodes)
	case reflect.String:
		atom := ast.NewAtom(prefix, ast.String(id), ast.String(val.String()))
		facts = append(facts, atom)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		atom := ast.NewAtom(prefix, ast.String(id), ast.Number(val.Int()))
		facts = append(facts, atom)
	case reflect.Float32, reflect.Float64:
		// Mangle does not have a float constant. We can convert it to string.
		s := strconv.FormatFloat(val.Float(), 'f', -1, 64)
		atom := ast.NewAtom(prefix, ast.String(id), ast.String(s))
		facts = append(facts, atom)
	case reflect.Bool:
		valStr := "false"
		if val.Bool() {
			valStr = "true"
		}
		atom := ast.NewAtom(prefix, ast.String(id), ast.String(valStr))
		facts = append(facts, atom)
	}

	return facts, nil
}
