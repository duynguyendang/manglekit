package knowledge

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ParseNTriples reads an N-Triples stream and returns Datalog facts.
// Format: triple("Subject", "Predicate", "Object")
func ParseNTriples(r io.Reader) ([]string, error) {
	var facts []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split line by space (limit 3 to perform basic triple separation)
		// Note: This is a simplified parser. It assumes standard N-Triples where
		// Subject and Predicate do not contain spaces. Object may contain spaces.
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}

		sub := clean(parts[0])
		pred := clean(parts[1])

		// The object part might include the trailing dot " ."
		objRaw := parts[2]
		obj := cleanObject(objRaw)

		// Create Manglekit fact: triple("Sub", "Pred", "Obj")
		facts = append(facts, fmt.Sprintf("triple(\"%s\", \"%s\", \"%s\")", escape(sub), escape(pred), escape(obj)))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return facts, nil
}

// clean removes wrapping brackets < > from IRIs.
func clean(s string) string {
	if strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") {
		return s[1 : len(s)-1]
	}
	return s
}

// cleanObject removes wrapping brackets < >, trailing dot, and handles literals.
func cleanObject(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".")
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") {
		return s[1 : len(s)-1]
	}

	// Handle literals: "value"^^<type> or "value"@lang
	if strings.HasPrefix(s, "\"") {
		// Remove wrapping quotes if present
		if strings.HasSuffix(s, "\"") {
			return s[1 : len(s)-1]
		}
		// If it has suffixes like ^^<type> or @lang
		// We'll simplisticly try to strip them if requested, but "Clean < and > brackets" was the instruction.
		// "Object: Remove surrounding < > (if URI) or " (if literal) and the trailing .."
		// For literals: logic to find the last quote and take content?
		// User instruction: Object: " (if literal)
		// E.g. "foo" -> foo
		// "foo"@en -> foo (maybe?)

		// Let's implement the "remove surrounding quotes" logic strictly for the simple case
		// If complex literal, we might need regex or smarter parsing.
		// For now, let's just strip leading quote.

		// Find last quote
		lastIdx := strings.LastIndex(s, "\"")
		if lastIdx > 0 {
			return s[1:lastIdx]
		}
	}

	return s
}

// escape escapes special characters for Datalog strings.
func escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

