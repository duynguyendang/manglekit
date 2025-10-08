package main

import (
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	// This is a minimal program to test Datalog parsing.
	// It does not run a full pipeline, only builds the rules engine.
	log.Println("Attempting to build pipeline with minimal policy...")

	rulesOpts := core.MangleOptions{
		Path: []string{"examples/04-chat-w-data/policy/access_control.dlog"},
	}

	builder := manglekit.NewBuilder().WithRules(&rulesOpts)

	_, err := builder.Build()
	if err != nil {
		log.Fatalf("failed to build pipeline: %v", err)
	}

	log.Println("Pipeline built successfully. The Datalog syntax appears to be valid.")
}