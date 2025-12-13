package inductor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SchemaHint holds the inferred schema information.
type SchemaHint struct {
	Declarations []string // For Graph: "Decl is_vip(S, O)."
	JsonKeys     []string // For JSON: "amount (number)", "desc (string)"
	FileType     string   // "graph" or "json"
}

// InferFromFile scans the file at path and returns a SchemaHint.
func InferFromFile(path string) (*SchemaHint, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return parseJSON(path)
	case ".nq", ".nt":
		return parseGraph(path)
	case ".ttl":
		return parseTurtle(path)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func parseJSON(path string) (*SchemaHint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}

	hint := &SchemaHint{FileType: "json"}
	var targetMap map[string]any

	switch v := raw.(type) {
	case map[string]any:
		targetMap = v
	case []any:
		if len(v) > 0 {
			if m, ok := v[0].(map[string]any); ok {
				targetMap = m
			}
		}
	}

	if targetMap == nil {
		// Could not find a suitable object to infer schema from
		return hint, nil // Empty hint but valid file type detected
	}

	walkJSON("", targetMap, &hint.JsonKeys)
	return hint, nil
}

func walkJSON(prefix string, data map[string]any, keys *[]string) {
	for k, v := range data {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]any:
			walkJSON(fullKey, val, keys)
		case []any:
			if len(val) > 0 {
				first := val[0]
				if m, ok := first.(map[string]any); ok {
					walkJSON(fullKey+"[]", m, keys)
				} else {
					switch first.(type) {
					case string:
						*keys = append(*keys, fmt.Sprintf("%s (array of string)", fullKey))
					case float64:
						*keys = append(*keys, fmt.Sprintf("%s (array of number)", fullKey))
					case bool:
						*keys = append(*keys, fmt.Sprintf("%s (array of boolean)", fullKey))
					}
				}
			}
		case float64:
			*keys = append(*keys, fmt.Sprintf("%s (number)", fullKey))
		case string:
			*keys = append(*keys, fmt.Sprintf("%s (string)", fullKey))
		case bool:
			*keys = append(*keys, fmt.Sprintf("%s (boolean)", fullKey))
		}
	}
}
