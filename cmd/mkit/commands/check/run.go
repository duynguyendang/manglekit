// Package check implements the `mkit check` subcommand: load and lint a
// policy without running a query.
package check

import (
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/spf13/cobra"
)

var CheckCmd = &cobra.Command{
	Use:   "check <policy.dl>",
	Short: "Load and lint a policy without running a query",
	Long: `Load and lint a policy without running a query.

Checks that the file parses and compiles against the manglekit engine
(including predicate declarations; undeclared or misdeclared predicates
are reported). Exits non-zero on any problem.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read policy file: %w", err)
		}

		e, err := engine.New()
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		// LoadPolicy parses + compiles the program against the standard
		// library (auto-loaded by engine.New), so undeclared predicates,
		// syntax errors, and arity mismatches all surface here.
		if err := e.LoadPolicy(cmd.Context(), string(content)); err != nil {
			return fmt.Errorf("policy %s is invalid: %w", path, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "OK: %s parsed and compiled successfully\n", path)
		return nil
	},
}

func init() {
	CheckCmd.Flags().SortFlags = true
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(CheckCmd)
}
