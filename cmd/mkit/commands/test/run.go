// Package testcmd implements the `mkit test` subcommand: declarative fixture testing
// for Datalog policies without writing Go code.
package testcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/cmd/mkit/commands/exitcode"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	policyFlag string
	verbose    bool
	outputFmt  string
)

var TestCmd = &cobra.Command{
	Use:   "test [flags] <test-file>",
	Short: "Run declarative unit test fixtures against a Datalog policy",
	Long: `Run declarative unit test fixtures against a Datalog policy.

Test fixtures can be defined in YAML or JSON, asserting that specific facts
and actions produce expected allow/halt decisions, matched rules, and tiers.

Exit codes follow the mkit contract:
  0  all tests passed
  1  one or more policy tests failed
  2  usage/flag error
  3  runtime/file error`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		testFilePath := args[0]
		return RunTests(cmd.Context(), testFilePath, policyFlag, verbose, outputFmt, cmd)
	},
}

func init() {
	TestCmd.Flags().StringVarP(&policyFlag, "policy", "p", "", "Path to the .mg policy file (overrides policy path in fixture)")
	TestCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed explanation tree for every test")
	TestCmd.Flags().StringVarP(&outputFmt, "output", "o", "text", "Output format: text or json")
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(TestCmd)
}

// FixtureSuite represents a collection of declarative policy test cases.
type FixtureSuite struct {
	Policy string     `yaml:"policy" json:"policy"`
	Tests  []TestCase `yaml:"tests" json:"tests"`
}

// TestCase represents an individual policy assertion.
type TestCase struct {
	Name           string   `yaml:"name" json:"name"`
	Action         string   `yaml:"action,omitempty" json:"action,omitempty"`
	Facts          []string `yaml:"facts" json:"facts"`
	Query          string   `yaml:"query,omitempty" json:"query,omitempty"`
	Expect         string   `yaml:"expect" json:"expect"` // "halt", "deny", "allow", "pass"
	ExpectedReason string   `yaml:"expected_reason,omitempty" json:"expected_reason,omitempty"`
	ExpectedTier   string   `yaml:"expected_tier,omitempty" json:"expected_tier,omitempty"`
}

type TestResult struct {
	Name           string `json:"name"`
	Passed         bool   `json:"passed"`
	Error          string `json:"error,omitempty"`
	ProofTree      string `json:"proof_tree,omitempty"`
	DurationMicros int64  `json:"duration_micros"`
}

type SuiteReport struct {
	PolicyPath string       `json:"policy_path"`
	Total      int          `json:"total"`
	Passed     int          `json:"passed"`
	Failed     int          `json:"failed"`
	DurationMs int64        `json:"duration_ms"`
	Results    []TestResult `json:"results"`
}

type TestFailureError struct {
	FailedCount int
	TotalCount  int
}

func (e *TestFailureError) Error() string {
	return fmt.Sprintf("%d of %d tests failed", e.FailedCount, e.TotalCount)
}

func (e *TestFailureError) Is(target error) bool {
	return target == core.ErrPolicyViolation
}

// RunTests executes the suite against the resolved policy.
func RunTests(ctx context.Context, testFilePath, overridePolicy string, verbose bool, output string, cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	testData, err := os.ReadFile(testFilePath)
	if err != nil {
		return fmt.Errorf("failed to read test file %q: %w", testFilePath, err)
	}

	var suite FixtureSuite
	if strings.HasSuffix(testFilePath, ".json") {
		if err := json.Unmarshal(testData, &suite); err != nil {
			return fmt.Errorf("failed to parse JSON test suite: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(testData, &suite); err != nil {
			return fmt.Errorf("failed to parse YAML test suite: %w", err)
		}
	}

	resolvedPolicyPath := overridePolicy
	if resolvedPolicyPath == "" {
		resolvedPolicyPath = suite.Policy
	}
	if resolvedPolicyPath == "" {
		return &exitcode.UsageError{Err: fmt.Errorf("policy path must be specified via --policy flag or in test fixture")}
	}

	// Resolve relative policy path relative to test file directory
	if !filepath.IsAbs(resolvedPolicyPath) {
		resolvedPolicyPath = filepath.Join(filepath.Dir(testFilePath), resolvedPolicyPath)
	}

	policyBytes, err := os.ReadFile(resolvedPolicyPath)
	if err != nil {
		return fmt.Errorf("failed to read policy file %q: %w", resolvedPolicyPath, err)
	}

	eng, err := engine.New()
	if err != nil {
		return fmt.Errorf("engine init failed: %w", err)
	}
	if err := eng.LoadPolicy(ctx, string(policyBytes)); err != nil {
		return fmt.Errorf("policy compilation failed for %q: %w", resolvedPolicyPath, err)
	}

	startTime := time.Now()
	report := SuiteReport{
		PolicyPath: resolvedPolicyPath,
		Total:      len(suite.Tests),
		Results:    make([]TestResult, 0, len(suite.Tests)),
	}

	for _, tc := range suite.Tests {
		tcStart := time.Now()
		facts := append([]string{}, tc.Facts...)

		// Inject action_operation if action is specified and not present
		if tc.Action != "" {
			actionFact := fmt.Sprintf(`action_operation("Req", "%s").`, tc.Action)
			hasAction := false
			for _, f := range facts {
				if strings.Contains(f, "action_operation") {
					hasAction = true
					break
				}
			}
			if !hasAction {
				facts = append(facts, actionFact)
			}
		}

		query := tc.Query
		if query == "" {
			query = `halt("Req", Reason, Tier)`
		}

		expl, err := eng.Explain(ctx, facts, query)
		duration := time.Since(tcStart)

		res := TestResult{
			Name:           tc.Name,
			DurationMicros: duration.Microseconds(),
		}

		if err != nil {
			res.Passed = false
			res.Error = fmt.Sprintf("evaluation error: %v", err)
		} else {
			res.ProofTree = expl.String()
			expectHalt := strings.EqualFold(tc.Expect, "halt") || strings.EqualFold(tc.Expect, "deny")

			if expectHalt {
				if !expl.Outcome {
					res.Passed = false
					res.Error = "expected halt rule to fire, but query produced no derivations (action allowed)"
				} else {
					res.Passed = true
					// Verify expected reason substring if specified
					if tc.ExpectedReason != "" && !strings.Contains(expl.String(), tc.ExpectedReason) {
						res.Passed = false
						res.Error = fmt.Sprintf("expected reason %q not found in derivations:\n%s", tc.ExpectedReason, expl.String())
					}
					// Verify expected tier if specified
					if tc.ExpectedTier != "" && !strings.Contains(expl.String(), tc.ExpectedTier) {
						res.Passed = false
						res.Error = fmt.Sprintf("expected tier %q not found in derivations:\n%s", tc.ExpectedTier, expl.String())
					}
				}
			} else { // expect allow
				if expl.Outcome {
					res.Passed = false
					res.Error = fmt.Sprintf("expected allow, but halt rule derived:\n%s", expl.String())
				} else {
					res.Passed = true
				}
			}
		}

		if res.Passed {
			report.Passed++
			if output == "text" {
				fmt.Fprintf(out, "=== RUN   %s\n", tc.Name)
				fmt.Fprintf(out, "--- PASS: %s (%dµs)\n", tc.Name, res.DurationMicros)
				if verbose && res.ProofTree != "" {
					fmt.Fprintf(out, "    proof tree:\n%s\n", indent(res.ProofTree, "      "))
				}
			}
		} else {
			report.Failed++
			if output == "text" {
				fmt.Fprintf(errOut, "=== RUN   %s\n", tc.Name)
				fmt.Fprintf(errOut, "--- FAIL: %s (%dµs)\n", tc.Name, res.DurationMicros)
				fmt.Fprintf(errOut, "    %s\n", res.Error)
				if res.ProofTree != "" {
					fmt.Fprintf(errOut, "    proof tree:\n%s\n", indent(res.ProofTree, "      "))
				}
			}
		}
		report.Results = append(report.Results, res)
	}

	report.DurationMs = time.Since(startTime).Milliseconds()

	if output == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
	} else {
		if report.Failed == 0 {
			fmt.Fprintf(out, "\nPASS: %d/%d tests passed in %dms (policy: %s)\n", report.Passed, report.Total, report.DurationMs, filepath.Base(resolvedPolicyPath))
		} else {
			fmt.Fprintf(errOut, "\nFAIL: %d/%d tests failed in %dms (policy: %s)\n", report.Failed, report.Total, report.DurationMs, filepath.Base(resolvedPolicyPath))
		}
	}

	if report.Failed > 0 {
		return &TestFailureError{FailedCount: report.Failed, TotalCount: report.Total}
	}

	return nil
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}
