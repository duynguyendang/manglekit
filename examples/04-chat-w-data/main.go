package main

import (
	"context"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	log.Println("Attempting to build rules engine with minimal policy...")

	rulesOpts := core.MangleOptions{
		Path: []string{"examples/04-chat-w-data/policy/access_control.dlog"},
	}

	constructor, err := manglekit.Get(manglekit.Registry.Rules, "mangle")
	if err != nil {
		log.Fatalf("failed to look up rules provider: %v", err)
	}

	newFn, ok := constructor.(func(context.Context, core.MangleOptions) (core.RuleSet, error))
	if !ok {
		log.Fatalf("unexpected constructor signature for rules provider: %T", constructor)
	}

	if _, err := newFn(context.Background(), rulesOpts); err != nil {
		log.Fatalf("failed to build rules engine: %v", err)
	}

	log.Println("Rules engine built successfully. The Datalog syntax appears to be valid.")
}
