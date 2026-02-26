package adapters

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit-wip/internal/core/domain"
	_ "github.com/duynguyendang/manglekit-wip/internal/core/ports" // for Interface declaration if needed
)

// ConsolePerception implements ports.PerceptionPort for a CLI interface.
type ConsolePerception struct {
	reader *bufio.Reader
}

func NewConsolePerception() *ConsolePerception {
	return &ConsolePerception{
		reader: bufio.NewReader(os.Stdin),
	}
}

// Normalize takes a raw string intent (Command-Line arg) and simulates turning it into Atomics.
func (c *ConsolePerception) Normalize(ctx context.Context, signal domain.Signal) (domain.Payload, error) {
	// A real perception adapter might run NER (Named Entity Recognition) over the text here.
	// For the example, we just push a dummy intent atom.

	atoms := []domain.Atom{
		{
			Subject:      "User",
			Predicate:    "requests",
			Object:       string(signal.Intent),
			Weight:       1.0,
			OriginIntent: signal.Intent,
		},
	}

	return func(yield func(domain.Atom) bool) {
		for _, a := range atoms {
			if !yield(a) {
				return
			}
		}
	}, nil
}

// ConsolePresentation implements ports.PresentationPort.
type ConsolePresentation struct{}

func NewConsolePresentation() *ConsolePresentation {
	return &ConsolePresentation{}
}

func (c *ConsolePresentation) Render(ctx context.Context, output domain.DecisionOutput) error {
	fmt.Printf("\n[AGENT OUTPUT]\nACTION: %s\nPARAMS: %+v\n\n", output.Action, output.Params)
	return nil
}
