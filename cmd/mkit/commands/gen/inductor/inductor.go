package inductor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	case ".nq", ".nt", ".ttl":
		return parseGraph(path)
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

	for k, v := range targetMap {
		switch v.(type) {
		case float64:
			hint.JsonKeys = append(hint.JsonKeys, fmt.Sprintf("%s (number)", k))
		case string:
			hint.JsonKeys = append(hint.JsonKeys, fmt.Sprintf("%s (string)", k))
		case bool:
			hint.JsonKeys = append(hint.JsonKeys, fmt.Sprintf("%s (boolean)", k))
		}
	}

	return hint, nil
}

func parseGraph(path string) (*SchemaHint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hint := &SchemaHint{FileType: "graph"}
	predicates := make(map[string]bool)
	scanner := bufio.NewScanner(f)

	// Regex to capture the predicate (second term) in N-Triples/N-Quads
	// Lines are like: <s> <p> <o> .
	// or: _:b1 <p> "lit" .
	// We look for the first occurrence of <...> that is not at the start,
	// or more robustly: subject predicate object.

	// A simple heuristic for N-Triples/Quads:
	// Subject is either <URI> or _:BNode
	// Predicate is <URI>
	// We skip the first term.

	// Regex:
	// ^\s*(?:<[^>]+>|_:[^\s]+)\s+(<[^>]+>)
	re := regexp.MustCompile(`^\s*(?:<[^>]+>|_:[^\s]+)\s+(<[^>]+>)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			pred := matches[1]
			// Check if we already have it
			if !predicates[pred] {
				predicates[pred] = true
				// Convert <http://example.org/pred> to Decl statement
				// We assume standard Mangle decl: Decl predicate(S, O).
				// We use the full URI as the predicate name if it's enclosed in <>.
				// Mangle supports <URI>(Arg1, Arg2).
				decl := fmt.Sprintf("Decl %s(S, O).", pred)
				hint.Declarations = append(hint.Declarations, decl)
			}
		} else {
			// Fallback for Turtle if it's simple (e.g. :pred object ;)
			// This is much harder to parse with regex.
			// If the user provided .ttl, we might miss things without a real parser.
			// But for now we stick to the NQ/NT style regex as per "Reuse regex logic" hint.
			// If the line starts with a predicate (continuation), it's harder.
			// We'll stick to the explicit triples for now.
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hint, nil
}
