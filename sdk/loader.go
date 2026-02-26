package sdk

import (
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit-wip/adapters/knowledge"
)

// LoadNTriplesFile opens a .nt file, parses it, and loads facts into the engine.
func (c *Client) LoadNTriplesFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open .nt file: %w", err)
	}
	defer f.Close()

	facts, err := knowledge.ParseNTriples(f)
	if err != nil {
		return fmt.Errorf("failed to parse NTriples: %w", err)
	}

	return c.engine.LoadFacts(facts)
}
