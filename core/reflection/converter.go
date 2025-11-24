package reflection

import (
	"fmt"
	"reflect"

	"github.com/google/mangle/ast"
)

// ToFacts converts a Go struct into a slice of Mangle Atoms.
// It uses the "mangle" struct tag to determine which fields to convert and what predicate name to use.
// Fields without the "mangle" tag are ignored.
//
// The generated atoms follow the pattern: predicate_name(entityID, value).
//
// Supported field types:
//   - string: converted to ast.StringConstant
//   - int, int8, int16, int32, int64: converted to ast.NumberConstant
//   - bool: converted to ast.StringConstant ("true" or "false")
//   - Pointers to the above types: dereferenced (nil pointers are ignored)
//
// entityID: The identifier for the subject (e.g., "request_123", "user_alice").
// entity: The struct instance to reflect over.
func ToFacts(entityID string, entity any) ([]ast.Atom, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, fmt.Errorf("entity is a nil pointer")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("entity must be a struct or a pointer to a struct, got %v", val.Kind())
	}

	typ := val.Type()
	var facts []ast.Atom

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("mangle")
		if tag == "" {
			continue
		}

		fieldVal := val.Field(i)
		// Handle pointers in fields
		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				continue
			}
			fieldVal = fieldVal.Elem()
		}

		var constVal ast.Constant

		switch fieldVal.Kind() {
		case reflect.String:
			constVal = ast.String(fieldVal.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			constVal = ast.Number(fieldVal.Int())
		case reflect.Bool:
			if fieldVal.Bool() {
				constVal = ast.String("true")
			} else {
				constVal = ast.String("false")
			}
		default:
			// For now, we skip unsupported types rather than erroring,
			// or we could return an error. The spec implies "Opt-in behavior" via tags,
			// but doesn't strictly say what to do with tagged but unsupported types.
			// Given it's a reflection engine, skipping or erroring is fine.
			// Let's return an error to be safe and explicit about what's supported.
			return nil, fmt.Errorf("unsupported type %v for field %s", fieldVal.Kind(), field.Name)
		}

		atom := ast.NewAtom(tag, ast.String(entityID), constVal)
		facts = append(facts, atom)
	}

	return facts, nil
}
