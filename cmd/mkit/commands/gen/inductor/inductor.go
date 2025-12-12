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
	case ".nq", ".nt":
		return parseGraph(path)
	case ".ttl":
		return parseTurtle(path)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func parseTurtle(path string) (*SchemaHint, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(b)

	// Remove comments
	lines := strings.Split(content, "\n")
	var cleanLines []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "#") || strings.HasPrefix(l, "@prefix") || strings.HasPrefix(l, "@base") {
			continue
		}
		cleanLines = append(cleanLines, l)
	}
	content = strings.Join(cleanLines, " ")

	// Tokenize
	// We do a naive split by whitespace, but we must handle punctuation attached to words.
	// For simplicity in this inductor, we assume standard spacing or we replace some punctuation.
	// We'll trust strings.Fields which splits by space.
	// To be safer, let's pad punctuation with spaces.
	content = strings.ReplaceAll(content, ";", " ; ")
	content = strings.ReplaceAll(content, ".", " . ")
	content = strings.ReplaceAll(content, ",", " , ")

	tokens := strings.Fields(content)

	hint := &SchemaHint{FileType: "graph"}
	predicates := make(map[string]bool)

	// State Machine
	const (
		StateSubject   = 0
		StatePredicate = 1
		StateObject    = 2
	)
	state := StateSubject

	for _, token := range tokens {
		if token == "" {
			continue
		}

		// Handle Punctuation transitions
		if token == "." {
			state = StateSubject
			continue
		}
		if token == ";" {
			state = StatePredicate
			continue
		}
		if token == "," {
			state = StateObject // Next is another object for same predicate
			continue
		}

		switch state {
		case StateSubject:
			// Ignore Subject
			state = StatePredicate

		case StatePredicate:
			// CAPTURE Predicate
			// Ignore if it looks like a string literal start (though usually predicates aren't literals)
			if strings.HasPrefix(token, "\"") {
				// Unexpected, but let's just move on
				state = StateObject
				continue
			}

			// Sanitize
			sanitized := SanitizePredicate(token)
			if sanitized != "" && sanitized != "_" {
				if !predicates[sanitized] {
					predicates[sanitized] = true
					decl := fmt.Sprintf("Decl %s(S, O).", sanitized)
					hint.Declarations = append(hint.Declarations, decl)
				}
			}
			state = StateObject

		case StateObject:
			// Ignore Object
			// Stay in StateObject until punctuation changes state
			// But since we are token by token, we just wait for next punctuation or if it's a list, the punctuation handler above catches ","
			// Wait, if we have "S P O .", O is token. Next is ".".
			// If we have "S P O ;", O is token. Next is ";".
			// So we just consume the object here.
			// But what if the object is "literal string with spaces"?
			// Our naive tokenizer split by space destroys string literals.
			// This is a limitation of this simple inductor.
			// However, for *schema induction*, we rarely care about the object values unless we are aggressive.
			// We only care about predicates found in State 1.
			// So even if "Hello World" becomes "Hello", "World", the state machine might get confused.
			// "S P "Hello" "World" ." -> S(0) P(CAPTURE) Hello(2) World(2? No, loop).
			// If we assume O is one token, then "World" would be seen as "." or start of next triple?
			// Actually, "Hello" puts us in Object state. "World" would be processed as... wait, we need to stay in proper flow.
			// If we are in StateObject, and we encounter "World" (not punctuation), what is it?
			// It implies the previous object didn't finish, OR we are lost.
			// Let's try to just stay in StateObject?
			// But if we encounter the next S?
			// "S P O . S2 P2 O2"
			// O -> . (StateSubject) -> S2 (StatePredicate) ... works.
			// The issue is only multi-token objects.
			// We can just ignore *everything* until we hit punctuation.
		}
	}

	return hint, nil
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
			rawPred := matches[1]
			// Check if we already have it
			if !predicates[rawPred] {
				predicates[rawPred] = true

				// Sanitize the predicate
				sanitized := SanitizePredicate(rawPred)

				// Convert to Decl statement
				// We assume standard Mangle decl: Decl predicate(S, O).
				decl := fmt.Sprintf("Decl %s(S, O).", sanitized)
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

// SanitizePredicate cleans up N-Quads predicates.
// Input: <http://example.org/total_cost>
// Output: total_cost
func SanitizePredicate(raw string) string {
	// Handle Turtle shorthand "a" -> "type"
	if raw == "a" {
		return "type"
	}

	// 1. Remove < >
	s := strings.Trim(raw, "<>")

	// 2. Get last part of URL/URN or QName
	// We look for the last occurrence of /, #, or :
	if idx := strings.LastIndexAny(s, "/#:"); idx != -1 {
		s = s[idx+1:]
	}

	// 3. Replace invalid chars (e.g. '-') with '_'
	// Datalog identifiers must match [a-zA-Z_][a-zA-Z0-9_]*
	// We replace common separators first
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")

	return s
}
