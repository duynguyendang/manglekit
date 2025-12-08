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
// It handles simple splitting by space and supports both 3-part (default graph) and 4-part (named graph) lines.
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

		// Sanitize: Trim trailing " ." (space + dot) from the line.
		// Handling simply trailing point with optional space
		line = strings.TrimSuffix(line, ".")
		line = strings.TrimSpace(line)

		// Split: Split the string by spaces.
		parts := strings.Fields(line) // Fields splits by one or more whitespace characters

		var s, p, o, g string

		if len(parts) == 3 {
			// <s> <p> <o>
			s, p, o = parts[0], parts[1], parts[2]
			g = DefaultGraph
		} else if len(parts) == 4 {
			// <s> <p> <o> <g>
			s, p, o, g = parts[0], parts[1], parts[2], parts[3]
		} else {
			// Else: Skip line or log warning (do not panic).
			continue
		}

		// Clean: removes angle brackets < and > from URIs.
		s = l.clean(s)
		p = l.clean(p)
		o = l.clean(o)
		g = l.clean(g) // Graph can also be a URI

		// Generate Datalog fact
		// We use %q to properly quote the strings for Datalog (Fact logic)
		// Usually Datalog expects: quad("s", "p", "o", "g").
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
