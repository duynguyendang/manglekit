package knowledge

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// DefaultGraph is the default graph name used when parsing N-Triples or when the graph component is missing.
const DefaultGraph = "default"

// BaseRules acts as the Datalog schema compatibility layer, mapping quads to triples.
const BaseRules = `
% Auto-map quads to triples (ignoring graph context)
triple(S, P, O) :- quad(S, P, O, _).
`

// NQuadsLoader is a Zero-Dependency parser that handles both N-Triples (.nt) and N-Quads (.nq) formats.
// It normalizes all entries into Datalog "quad/4" facts, using a default graph for triples.
type NQuadsLoader struct{}

// NewNQuadsLoader creates a new instance of NQuadsLoader.
func NewNQuadsLoader() *NQuadsLoader {
	return &NQuadsLoader{}
}

// GetBaseRules returns the base Datalog rules associated with this loader.
func (l *NQuadsLoader) GetBaseRules() string {
	return BaseRules
}

// Parse reads from the given reader and converts N-Triples/N-Quads into Datalog facts.
// It robustly handles quoted literals containing spaces.
func (l *NQuadsLoader) Parse(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var facts []string

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Filter: Skip empty lines and lines starting with #
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple Tokenizer
		var tokens []string
		var currentToken strings.Builder
		inQuote := false
		escaped := false

		// We iterate manually to handle quotes
		for _, r := range line {
			if inQuote {
				if r == '"' && !escaped {
					inQuote = false
					currentToken.WriteRune(r)
				} else if r == '\\' && !escaped {
					escaped = true
					currentToken.WriteRune(r)
				} else {
					escaped = false
					currentToken.WriteRune(r)
				}
				continue
			}

			// Not in quote
			if r == '"' {
				inQuote = true
				currentToken.WriteRune(r)
				continue
			}

			if r == ' ' || r == '\t' {
				if currentToken.Len() > 0 {
					tokens = append(tokens, currentToken.String())
					currentToken.Reset()
				}
				continue
			}

			// Check for end of statement dot (if isolated or at end)
			// But determining if '.' is a token or part of URI/Literal is tricky if not careful.
			// Standard N-Quads: dot is separate token usually separated by space,
			// or at the end of the line.
			// Let's treat it as a char, and handle it in post-processing or tokenizing.
			// Assuming '.' followed by space or EOF is a separator only if not in token.
			// Simplification: Standard format has space before dot.
			currentToken.WriteRune(r)
		}
		if currentToken.Len() > 0 {
			tokens = append(tokens, currentToken.String())
		}

		// Validate and Extract
		// Tokens should be: S, P, O, [G], "."
		// The last token should be "."
		if len(tokens) > 0 && tokens[len(tokens)-1] == "." {
			tokens = tokens[:len(tokens)-1]
		}

		var s, p, o, g string

		if len(tokens) == 3 {
			// <s> <p> <o>
			s, p, o = tokens[0], tokens[1], tokens[2]
			g = DefaultGraph
		} else if len(tokens) == 4 {
			// <s> <p> <o> <g>
			s, p, o, g = tokens[0], tokens[1], tokens[2], tokens[3]
		} else {
			// Malformed or complex case (e.g. lang tags parsed as separate tokens if spaces?)
			// For this objective, we assume standard format.
			continue
		}

		// Clean: removes angle brackets < and > from URIs.
		s = l.clean(s)
		p = l.clean(p)
		o = l.clean(o)
		g = l.clean(g)

		// Generate Datalog fact
		fact := fmt.Sprintf("quad(%q, %q, %q, %q)", s, p, o, g)
		facts = append(facts, fact)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return facts, nil
}

// clean removes angle brackets < and > from URIs.
func (l *NQuadsLoader) clean(s string) string {
	if strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") {
		return s[1 : len(s)-1]
	}
	return s
}
