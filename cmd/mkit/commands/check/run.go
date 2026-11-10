// Package check implements the `mkit check` and `mkit lint` subcommands:
// load, parse, compile, and lint a Datalog policy without running a query.
package check

import (
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/spf13/cobra"
)

var lintFlag bool

type LintError struct {
	Count int
}

func (e *LintError) Error() string {
	return fmt.Sprintf("policy contains %d lint error(s)", e.Count)
}

func (e *LintError) Is(target error) bool {
	return target == core.ErrPolicyViolation
}

func runCheck(cmd *cobra.Command, args []string) error {
	path := args[0]

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	e, err := engine.New()
	if err != nil {
		return fmt.Errorf("failed to initialize engine: %w", err)
	}

	if err := e.LoadPolicy(cmd.Context(), string(content)); err != nil {
		return fmt.Errorf("policy %s is invalid: %w", path, err)
	}

	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	if lintFlag {
		issues, lintErr := LintPolicy(string(content))
		if lintErr != nil {
			return fmt.Errorf("linter error: %w", lintErr)
		}

		errCount := 0
		for _, issue := range issues {
			if issue.Severity == SeverityError {
				errCount++
				fmt.Fprintf(errOut, "ERROR: %s\n", issue.Message)
			} else {
				fmt.Fprintf(out, "WARN: %s\n", issue.Message)
			}
			if issue.Rule != "" {
				fmt.Fprintf(out, "       rule: %s\n", issue.Rule)
			}
		}

		if errCount > 0 {
			return &LintError{Count: errCount}
		}
	}

	fmt.Fprintf(out, "OK: %s parsed and compiled successfully\n", path)
	return nil
}

var CheckCmd = &cobra.Command{
	Use:   "check <policy.dl>",
	Short: "Load, compile, and lint a policy without running a query",
	Long: `Load, compile, and lint a policy without running a query.

Checks that the file parses and compiles against the manglekit engine
(including predicate declarations and mode checks). With --lint (default),
it checks for ungrounded negated variables and dead rule premises.`,
	Args: cobra.ExactArgs(1),
	RunE: runCheck,
}

var LintCmd = &cobra.Command{
	Use:   "lint <policy.dl>",
	Short: "Lint a Datalog policy for ungrounded variables and dead rules",
	Long:  "Alias for `mkit check --lint`.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lintFlag = true
		return runCheck(cmd, args)
	},
}

func init() {
	CheckCmd.Flags().BoolVar(&lintFlag, "lint", true, "Perform static lint checks for ungrounded negated variables and dead rules")
	LintCmd.Flags().BoolVar(&lintFlag, "lint", true, "Perform static lint checks for ungrounded negated variables and dead rules")
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(CheckCmd)
	rootCmd.AddCommand(LintCmd)
}
