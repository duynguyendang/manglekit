package parse

import (
	"fmt"
	"regexp"
	"strings"
)

// Simple regex to capture: pred("arg1", "arg2")
var atomRegex = regexp.MustCompile(`^(\w+)\((.*)\)$`)

// ParseAtomContent splits a fact string into predicate and arguments.
// Example: `json_str("A", "B", "C")` -> "json_str", ["A", "B", "C"]
func ParseAtomContent(fact string) (string, []string, error) {
	fact = strings.TrimSpace(fact)
	matches := atomRegex.FindStringSubmatch(fact)
	if len(matches) != 3 {
		return "", nil, fmt.Errorf("invalid atom format")
	}

	pred := matches[1]
	argsStr := matches[2]

	// Split args by comma, respecting quotes (Simplified version)
	// A real tokenizer is better, but this works for basic json_* facts
	// IMPROVEMENT: Handle commas inside quotes
	var args []string
	var currentArg strings.Builder
	inQuote := false

	for _, r := range argsStr {
		switch r {
		case '"':
			inQuote = !inQuote
			currentArg.WriteRune(r)
		case ',':
			if inQuote {
				currentArg.WriteRune(r)
			} else {
				args = append(args, strings.TrimSpace(currentArg.String()))
				currentArg.Reset()
			}
		default:
			currentArg.WriteRune(r)
		}
	}
	if currentArg.Len() > 0 {
		args = append(args, strings.TrimSpace(currentArg.String()))
	}

	// Clean quotes from args
	for i, a := range args {
		if strings.HasPrefix(a, "\"") && strings.HasSuffix(a, "\"") {
			if len(a) >= 2 {
				args[i] = a[1 : len(a)-1]
			} else {
				args[i] = ""
			}
		}
	}

	return pred, args, nil
}
