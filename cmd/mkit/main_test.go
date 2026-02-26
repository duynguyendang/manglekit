package main

import (
	"testing"

	"github.com/duynguyendang/manglekit-wip/cmd/mkit/commands/eval"
	"github.com/duynguyendang/manglekit-wip/cmd/mkit/commands/gen"
	"github.com/duynguyendang/manglekit-wip/cmd/mkit/commands/inspect"
	"github.com/duynguyendang/manglekit-wip/cmd/mkit/commands/kg"
	"github.com/duynguyendang/manglekit-wip/cmd/mkit/commands/serve"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRootCmd(t *testing.T) {
	// Verify basic properties of the root command
	assert.Equal(t, "mkit", rootCmd.Use)
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
}

func TestCLIWiring(t *testing.T) {
	// Create a dummy root command to test wiring behavior
	// This simulates main()
	dummyRoot := &cobra.Command{Use: "mkit-test"}

	// Manually wire commands as main() does
	gen.AddCommands(dummyRoot)
	inspect.AddCommands(dummyRoot)
	kg.AddCommands(dummyRoot)
	eval.AddCommands(dummyRoot)
	serve.AddCommands(dummyRoot)

	// Verify all expected subcommands are present
	commands := dummyRoot.Commands()
	assert.NotEmpty(t, commands, "Subcommands should be registered")

	expected := map[string]bool{
		"generate": false,
		"inspect":  false,
		"kg":       false,
		"eval":     false,
		"serve":    false,
	}

	for _, cmd := range commands {
		name := cmd.Name()
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}

	for name, found := range expected {
		assert.True(t, found, "Command '%s' should be registered", name)
	}
}
