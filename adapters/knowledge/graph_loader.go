package knowledge

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Triple represents a raw graph triple.
type Triple struct {
	Subject   string
	Predicate string
	Object    string
}

// ParseGraphFile loads triples from .nq, .nt, or .ttl files.
func ParseGraphFile(path string) ([]Triple, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".nq", ".nt":
		return parseNQuads(path)
	case ".ttl":
		return parseTurtle(path)
	default:
		return nil, fmt.Errorf("unsupported graph file type: %s", ext)
	}
}

// parseNQuads parses N-Quads/N-Triples using regex.
func parseNQuads(path string) ([]Triple, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var triples []Triple
	scanner := bufio.NewScanner(f)

	// Regex: Capture Subject (URI/BNode), Predicate (URI), Object (URI/BNode/Literal)
	// Simplified: S P O .
	// ^\s*(<[^>]+>|_:[^\s]+)\s+(<[^>]+>)\s+(.*)\s+\.
	// Note: N-Quads has optional Graph G. We ignore G for now.
	// Actually, matching O is tricky because of literals with spaces.
	// But N-Quads usually escape inner quotes.
	// Let's use a simpler heuristic of splitting by space but respecting quotes?
	// Or stick to the regex we used for Schema Inductor but expand to capture S and O.

	// <subject> <predicate> <object> .
	re := regexp.MustCompile(`^\s*(<[^>]+>|_:[^\s]+)\s+(<[^>]+>)\s+(.*)\s+\.`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) > 3 {
			// If O ends with a graph label like <http://g> ., we need to trim it.
			// But regex greediness on .* might eat it.
			// N-Quads: S P O G .
			// If we split by space, we get [S, P, O, ., (G?)]?
			// Let's rely on basic splitting for this MVP since regex for full N-Quads is complex.
			// Re-parse the line without regex.
			parts := splitLineRespectingQuotes(line)
			if len(parts) >= 3 {
				triples = append(triples, Triple{
					Subject:   parts[0],
					Predicate: SanitizePredicate(parts[1]),
					Object:    parts[2],
				})
			}
		}
	}
	return triples, scanner.Err()
}

// parseTurtle parses Turtle files using a simple state machine.
func parseTurtle(path string) ([]Triple, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(b)

	// Pre-process: remove comments
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
	content = strings.ReplaceAll(content, ";", " ; ")
	content = strings.ReplaceAll(content, ".", " . ")
	content = strings.ReplaceAll(content, ",", " , ")

	tokens := strings.Fields(content)

	var triples []Triple

	const (
		StateSubject   = 0
		StatePredicate = 1
		StateObject    = 2
	)
	state := StateSubject

	var currentSubject, currentPredicate string

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
			currentSubject = token
			state = StatePredicate
		case StatePredicate:
			currentPredicate = SanitizePredicate(token)
			state = StateObject
		case StateObject:
			// Capture triple
			triples = append(triples, Triple{
				Subject:   currentSubject,
				Predicate: currentPredicate,
				Object:    token,
			})
			// Wait for punctuation to transition
		}
	}

	return triples, nil
}

// TriplesToFacts converts graph triples to Datalog atoms.
func TriplesToFacts(triples []Triple) []string {
	var facts []string
	for _, t := range triples {
		// Predicate: Raw identifier
		pred := t.Predicate
		// Subject: Always a quoted string identifier
		subj := fmt.Sprintf("%q", strings.Trim(t.Subject, "<>"))
		// Object: Smart cast (Number or String)
		obj := smartCast(t.Object)

		facts = append(facts, fmt.Sprintf("%s(%s, %s).", pred, subj, obj))
	}
	return facts
}

func smartCast(val string) string {
	// 1. Aggressive Cleaning
	// Remove RDF type suffixes like "^^<http://www.w3.org/2001/XMLSchema#integer>"
	if idx := strings.Index(val, "^^"); idx != -1 {
		val = val[:idx]
	}
	// Remove whitespace
	val = strings.TrimSpace(val)

	// 2. Handle URIs: Strip < > if present
	if strings.HasPrefix(val, "<") && strings.HasSuffix(val, ">") {
		val = val[1 : len(val)-1]
	}

	// 3. Remove Quotes
	cleanVal := strings.Trim(val, "\"")

	// 4. Try Parsing as Float
	if _, err := strconv.ParseFloat(cleanVal, 64); err == nil {
		// Heuristic: Check for Leading Zeros to preserve IDs/ZipCodes
		// Rule: If it starts with '0', it must be exactly "0" or "0.xyz".
		// If it is like "0123", treat as String.
		if strings.HasPrefix(cleanVal, "0") && cleanVal != "0" && !strings.HasPrefix(cleanVal, "0.") {
			return fmt.Sprintf("%q", cleanVal) // Keep as String
		}

		// It is a valid number (e.g., "15000", "3.14", "0", "0.5")
		return cleanVal // Return Raw Number
	}

	// 5. Fallback: Not a number, return quoted string
	return fmt.Sprintf("%q", cleanVal)
}

// splitLineRespectingQuotes helps parse N-Triples lines.
func splitLineRespectingQuotes(s string) []string {
	// naive impl for MVP
	return strings.Fields(s)
}

// SanitizePredicate cleans up predicates (Shared logic).
func SanitizePredicate(raw string) string {
	if raw == "a" {
		return "type"
	}
	s := strings.Trim(raw, "<>")
	if idx := strings.LastIndexAny(s, "/#:"); idx != -1 {
		s = s[idx+1:]
	}
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}
