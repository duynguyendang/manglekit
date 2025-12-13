package inductor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func parseGraph(path string) (*SchemaHint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hint := &SchemaHint{FileType: "graph"}
	predicates := make(map[string]bool)
	scanner := bufio.NewScanner(f)

	// Regex to capture Subject, Predicate, Object in N-Triples/N-Quads
	// Groups:
	// 1: Subject (URI or BNode)
	// 2: Predicate (URI)
	// 3: Object (URI, BNode, or Literal) - we capture everything until the final dot
	re := regexp.MustCompile(`^\s*(<[^>]+>|_:[^\s]+)\s+(<[^>]+>)\s+(.+)\s*\.\s*$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) > 3 {
			rawSub := matches[1]
			rawPred := matches[2]
			rawObj := strings.TrimSpace(matches[3])

			sanitizedPred := SanitizeTerm(rawPred)

			// Skip invalid or reserved predicates
			if sanitizedPred == "" || sanitizedPred == "_" {
				continue
			}

			if !predicates[sanitizedPred] {
				predicates[sanitizedPred] = true

				sanitizedSub := SanitizeTerm(rawSub)
				sanitizedObj := SanitizeTerm(rawObj)

				// Format: Decl pred(S, O).       % Sample: pred("sub", "obj")
				// We use %q to safely quote the sample values.
				decl := fmt.Sprintf("Decl %s(S, O).       %% Sample: %s(%q, %q)",
					sanitizedPred, sanitizedPred, sanitizedSub, sanitizedObj)
				hint.Declarations = append(hint.Declarations, decl)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hint, nil
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

	// Simple Tokenization
	content = strings.ReplaceAll(content, ";", " ; ")
	content = strings.ReplaceAll(content, ".", " . ")
	content = strings.ReplaceAll(content, ",", " , ")

	tokens := strings.Fields(content)

	hint := &SchemaHint{FileType: "graph"}
	predicates := make(map[string]bool)

	const (
		StateSubject   = 0
		StatePredicate = 1
		StateObject    = 2
	)
	state := StateSubject

	var currentSub, currentPred string

	for _, token := range tokens {
		if token == "" {
			continue
		}

		if token == "." {
			state = StateSubject
			continue
		}
		if token == ";" {
			state = StatePredicate
			continue
		}
		if token == "," {
			state = StateObject
			continue
		}

		switch state {
		case StateSubject:
			currentSub = token
			state = StatePredicate

		case StatePredicate:
			// Check for string start, though rare for predicate
			if strings.HasPrefix(token, "\"") {
				state = StateObject
				continue
			}
			currentPred = token
			state = StateObject

		case StateObject:
			// Capture the triple sample
			sanitizedPred := SanitizeTerm(currentPred)

			if sanitizedPred != "" && sanitizedPred != "_" {
				if !predicates[sanitizedPred] {
					predicates[sanitizedPred] = true

					sanitizedSub := SanitizeTerm(currentSub)
					sanitizedObj := SanitizeTerm(token)

					decl := fmt.Sprintf("Decl %s(S, O).       %% Sample: %s(%q, %q)",
						sanitizedPred, sanitizedPred, sanitizedSub, sanitizedObj)
					hint.Declarations = append(hint.Declarations, decl)
				}
			}
			// Don't advance state here; wait for punctuation to transition or stay in Object (list)
			// Actually, naive tokenizer treats next token as...
			// In Turtle, "S P O" is end of triple if followed by punctuation.
			// Or "S P O1, O2".
			// If we are in StateObject, we consumed one object token.
			// The next token should be punctuation.
			// If it's not punctuation, it implies we are parsing a multi-token object (not supported well)
			// or we are lost.
			// But since we just want *one* sample, capturing the first token of object is often "enough" for a hint.
		}
	}

	return hint, nil
}

// SanitizeTerm cleans up N-Quads/Turtle terms (Subjects, Predicates, Objects).
// Input: <http://example.org/total_cost>, "value", _:b1
// Output: total_cost, value, b1
func SanitizeTerm(raw string) string {
	// Handle Turtle shorthand "a" -> "type"
	if raw == "a" {
		return "type"
	}

	// 1. Remove < > and quotes "
	// We allow trimming quotes so "5000" becomes 5000, which we then re-quote in output.
	s := strings.Trim(raw, "<>\"")

	// 2. Get last part of URL/URN or QName
	// We look for the last occurrence of /, #, or :
	// Note: This logic might be too aggressive for objects that are just strings like "Hello World"
	// but "Hello World" isn't a URI. LastIndexAny might pick something if present.
	// E.g. "User: Alice". LastIndexAny(':') -> " Alice".
	// For hints, this is acceptable.
	if idx := strings.LastIndexAny(s, "/#:"); idx != -1 {
		s = s[idx+1:]
	}

	// 3. Replace invalid chars (e.g. '-') with '_'
	// Datalog identifiers must match [a-zA-Z_][a-zA-Z0-9_]*
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")

	// Clean up any remaining characters that might break %q or Datalog?
	// %q handles escaping.

	return s
}
