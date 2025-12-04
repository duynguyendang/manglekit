package engine

import (
	"fmt"
	"reflect"
)

// Flatten converts a dynamic JSON-like structure (maps, slices) into graph-based Mangle Datalog facts.
// It uses recursive traversal to generate nodes and links.
//
// Predicates:
//   - json_str(NodeID, Key, Value)
//   - json_num(NodeID, Key, Value)
//   - json_bool(NodeID, Key, Value)
//   - json_link(ParentID, Key, ChildNodeID)
//
// Parameters:
//   - rootID: The ID of the root node (e.g., "Req").
//   - input: The data to flatten (expected map[string]any or []any).
func Flatten(rootID string, input any) ([]string, error) {
	var facts []string
	if input == nil {
		return facts, nil
	}

	counter := 0
	if err := flattenRecursive(rootID, input, &facts, &counter); err != nil {
		return nil, err
	}
	return facts, nil
}

func flattenRecursive(nodeID string, data any, facts *[]string, counter *int) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		data = v.Interface()
	}

	switch val := data.(type) {
	case map[string]any:
		for k, child := range val {
			safeKey := escapeString(k)
			if isComplex(child) {
				*counter++
				childID := fmt.Sprintf("node_%d", *counter)
				// Link: json_link("NodeID", "Key", "ChildID")
				fact := fmt.Sprintf("json_link(\"%s\", \"%s\", \"%s\")", escapeString(nodeID), safeKey, escapeString(childID))
				*facts = append(*facts, fact)
				if err := flattenRecursive(childID, child, facts, counter); err != nil {
					return err
				}
			} else {
				addPrimitiveFact(nodeID, safeKey, child, facts)
			}
		}
	case []any:
		for i, child := range val {
			key := fmt.Sprintf("%d", i)
			safeKey := key
			if isComplex(child) {
				*counter++
				childID := fmt.Sprintf("node_%d", *counter)
				fact := fmt.Sprintf("json_link(\"%s\", \"%s\", \"%s\")", escapeString(nodeID), safeKey, escapeString(childID))
				*facts = append(*facts, fact)
				if err := flattenRecursive(childID, child, facts, counter); err != nil {
					return err
				}
			} else {
				addPrimitiveFact(nodeID, safeKey, child, facts)
			}
		}
	default:
		// If root is primitive, just ignore or log?
		// Logic handles children of root.
	}
	return nil
}

func isComplex(v any) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	k := rv.Kind()
	return k == reflect.Map || k == reflect.Slice || k == reflect.Array
}

func addPrimitiveFact(nodeID, key string, val any, facts *[]string) {
	nodeID = escapeString(nodeID)
	// key is already escaped

	switch v := val.(type) {
	case string:
		fact := fmt.Sprintf("json_str(\"%s\", \"%s\", \"%s\")", nodeID, key, escapeString(v))
		*facts = append(*facts, fact)
	case float64:
		fact := fmt.Sprintf("json_num(\"%s\", \"%s\", %f)", nodeID, key, v)
		*facts = append(*facts, fact)
	case int, int32, int64:
		fact := fmt.Sprintf("json_num(\"%s\", \"%s\", %d)", nodeID, key, v)
		*facts = append(*facts, fact)
	case bool:
		sVal := "false"
		if v {
			sVal = "true"
		}
		fact := fmt.Sprintf("json_bool(\"%s\", \"%s\", \"%s\")", nodeID, key, sVal)
		*facts = append(*facts, fact)
	default:
		sVal := fmt.Sprintf("%v", v)
		fact := fmt.Sprintf("json_str(\"%s\", \"%s\", \"%s\")", nodeID, key, escapeString(sVal))
		*facts = append(*facts, fact)
	}
}
