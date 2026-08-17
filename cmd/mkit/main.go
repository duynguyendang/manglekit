package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/cmd/mkit/commands/check"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/eval"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/exitcode"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/gen"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/inspect"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/kg"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/run"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/serve"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/skill"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/version"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mkit",
	Short: "mkit: The CLI for Neuro-Symbolic AI Governance",
	Long: `mkit is a powerful CLI tool designed for the Manglekit framework.
It facilitates neuro-symbolic AI governance by providing utilities to generate rules, inspect data schemas, and run local pipelines.

Exit codes:
  0  success
  1  policy-deny (a governance policy blocked the action)
  2  usage/flag error
  3  runtime error`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default action when no subcommand is provided
		cmd.Help()
	},
}

func Execute() {
	// We print errors ourselves (with an exit-code classification), so stop
	// cobra from double-printing them; usage output is suppressed because
	// runtime errors should not dump the full help text.
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	err := rootCmd.Execute()
	if err == nil {
		os.Exit(exitcode.Success)
	}

	code := exitcode.CodeFor(err)
	// Cobra reports unknown commands/subcommands as plain errors; classify
	// them as usage errors.
	if code == exitcode.Runtime && strings.HasPrefix(err.Error(), "unknown command") {
		code = exitcode.Usage
	}

	fmt.Fprintln(os.Stderr, "mkit:", err)
	if code == exitcode.Usage {
		fmt.Fprintln(os.Stderr, "Run 'mkit --help' for usage.")
	}
	os.Exit(code)
}

// flagErrorAsUsage makes cobra flag errors (unknown flag, missing required
// flag) map to the usage exit code.
func flagErrorAsUsage(cmd *cobra.Command, err error) error {
	return &exitcode.UsageError{Err: err}
}

func main() {
	_ = godotenv.Load()
	rootCmd.SetFlagErrorFunc(flagErrorAsUsage)
	gen.AddCommands(rootCmd)
	inspect.AddCommands(rootCmd)
	kg.AddCommands(rootCmd)
	eval.AddCommands(rootCmd)
	run.AddCommands(rootCmd)
	serve.AddCommands(rootCmd)
	skill.AddCommands(rootCmd)
	check.AddCommands(rootCmd)
	version.AddCommands(rootCmd)
	Execute()
}
