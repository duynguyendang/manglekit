package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/duynguyendang/manglekit/adapters/knowledge"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/exitcode"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/spf13/cobra"
)

var (
	policyPath    string
	dataPath      string
	knowledgePath string
	queryString   string
	outputMode    string
	quiet         bool
	explain       bool
)

const (
	outputJSON  = "json"
	outputTable = "table"
)

var EvalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate a Datalog query against policy and data",
	Long: `Evaluate a Datalog query against policy and data.

Output modes (--output):
  json   Print the query results as a JSON array on stdout. All progress
         messages go to stderr, so the output can be piped to jq.
  table  Print the results as an aligned table (default).

Exit codes: 0 success, 1 policy-deny, 2 usage error, 3 runtime error.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if outputMode != outputJSON && outputMode != outputTable {
			return exitcode.UsageErrorf("invalid --output value %q: must be %q or %q", outputMode, outputJSON, outputTable)
		}

		// Validation
		if dataPath == "" && knowledgePath == "" {
			return exitcode.UsageErrorf("must provide either --data (JSON) or --facts/--knowledge (Graph)")
		}

		// Progress/log output goes to stderr so stdout stays clean for
		// machine-readable output (mkit eval --output json ... | jq .).
		progress := func(format string, a ...any) {
			if quiet {
				return
			}
			fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", a...)
		}

		// 1. Read Files
		policyBytes, err := os.ReadFile(policyPath)
		if err != nil {
			return fmt.Errorf("failed to read policy file: %w", err)
		}

		var dataBytes []byte
		if dataPath != "" {
			dataBytes, err = os.ReadFile(dataPath)
			if err != nil {
				return fmt.Errorf("failed to read data file: %w", err)
			}
		}

		// 2. Initialize Engine
		// New() now auto-loads std.dl, so no manual declaration fixes needed!
		e, err := engine.New()
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		// 3. Process Knowledge (Facts) First to Extract Schema
		if knowledgePath != "" {
			// Using the new Graph Loader that supports .nq and .ttl
			triples, err := knowledge.ParseGraphFile(knowledgePath)
			if err != nil {
				return fmt.Errorf("failed to parse knowledge base: %w", err)
			}

			// Extract Schema Declarations
			preds := knowledge.GetPredicates(triples)
			if len(preds) > 0 {
				var decls []string
				for _, p := range preds {
					// Default to binary declaration: Decl p(Subject, Object).
					decls = append(decls, fmt.Sprintf("Decl %s(S, O).", p))
				}
				schemaBlock := strings.Join(decls, "\n")

				// Load Declarations BEFORE Policy
				if err := e.LoadPolicy(cmd.Context(), schemaBlock); err != nil {
					return fmt.Errorf("failed to inject schema declarations: %w", err)
				}
				progress("Injected %d schema declarations.", len(preds))
			}

			// Load Facts
			facts := knowledge.TriplesToFacts(triples)
			if err := e.LoadFacts(cmd.Context(), facts); err != nil {
				return fmt.Errorf("failed to load knowledge facts: %w", err)
			}
			progress("Loaded %d knowledge facts.", len(facts))
		}

		// 4. Load Policy (User Policy)
		// Now that Decls are present (if any), loading the policy should succeed.
		if err := e.LoadPolicy(cmd.Context(), string(policyBytes)); err != nil {
			// This might still error if the user uses non-standard predicates without declaring them,
			// but json_str, quad etc are now covered.
			return fmt.Errorf("failed to load policy: %w", err)
		}

		// 5. Inject Data (JSON)
		if dataPath != "" {
			var data any
			if err := json.Unmarshal(dataBytes, &data); err != nil {
				return fmt.Errorf("failed to parse data JSON: %w", err)
			}

			// Flatten the data into facts
			// "input" is used as the root ID for the data
			dataFacts, err := engine.Flatten("input", data)
			if err != nil {
				return fmt.Errorf("failed to flatten data: %w", err)
			}

			if err := e.LoadFacts(cmd.Context(), dataFacts); err != nil {
				return fmt.Errorf("failed to load data facts: %w", err)
			}
			progress("Loaded %d data facts.", len(dataFacts))
		}

		// 6. Execute Query
		ctx := context.Background()

		// 6a. --explain: render the derivation tree (proof) for the
		// matching facts instead of (table) or as (json) plain results.
		if explain {
			explanation, err := e.Explain(ctx, nil, queryString)
			if err != nil {
				return fmt.Errorf("explanation failed: %w", err)
			}
			stdout := cmd.OutOrStdout()
			if outputMode == outputJSON {
				outputBytes, err := json.MarshalIndent(explanation, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to format output: %w", err)
				}
				fmt.Fprintln(stdout, string(outputBytes))
			} else {
				fmt.Fprint(stdout, explanation.String())
			}
			return nil
		}

		// Using Query instead of ExecuteQuery to get results/bindings
		results, err := e.Query(ctx, nil, queryString)
		if err != nil {
			return fmt.Errorf("query execution failed: %w", err)
		}

		// 7. Output
		stdout := cmd.OutOrStdout()
		switch outputMode {
		case outputJSON:
			// Always emit a valid JSON array (even when empty) so
			// `mkit eval --output json ... | jq .` never chokes.
			outputBytes, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}
			fmt.Fprintln(stdout, string(outputBytes))
		default:
			if len(results) == 0 {
				progress("No results found.")
				return nil
			}
			printTable(stdout, results)
		}

		return nil
	},
}

// printTable renders query results ([]map[string]string bindings) as an
// aligned table with sorted column names.
func printTable(w io.Writer, results []map[string]string) {
	// Collect the union of variable names as columns, in a stable order.
	seen := make(map[string]bool)
	var cols []string
	for _, r := range results {
		for k := range r {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	sort.Strings(cols)

	widths := make([]int, len(cols))
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		row := make([]string, len(cols))
		for i, c := range cols {
			v := r[c]
			if len(v) > widths[i] {
				widths[i] = len(v)
			}
			row[i] = v
		}
		rows = append(rows, row)
	}
	for i, c := range cols {
		if len(c) > widths[i] {
			widths[i] = len(c)
		}
	}

	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = pad(c, widths[i])
	}
	fmt.Fprintln(w, strings.Join(header, " | "))
	sep := make([]string, len(cols))
	for i := range sep {
		sep[i] = strings.Repeat("-", widths[i])
	}
	fmt.Fprintln(w, strings.Join(sep, "-+-"))
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, v := range row {
			cells[i] = pad(v, widths[i])
		}
		fmt.Fprintln(w, strings.Join(cells, " | "))
	}
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func init() {
	EvalCmd.Flags().StringVarP(&policyPath, "policy", "p", "", "Path to the .dl file")
	EvalCmd.MarkFlagRequired("policy")

	EvalCmd.Flags().StringVarP(&dataPath, "data", "d", "", "Path to the .json input file. Optional if --knowledge is provided.")

	EvalCmd.Flags().StringVarP(&knowledgePath, "knowledge", "k", "", "Path to the .nq, .nt, or .ttl knowledge base")
	EvalCmd.Flags().StringVar(&knowledgePath, "facts", "", "Alias for --knowledge")

	EvalCmd.Flags().StringVarP(&queryString, "query", "q", "", "The Datalog query to execute")
	EvalCmd.MarkFlagRequired("query")

	EvalCmd.Flags().StringVar(&outputMode, "output", outputTable, `Output format: "table" (default) or "json"`)
	EvalCmd.Flags().BoolVarP(&quiet, "quiet", "", false, "Suppress progress messages")
	EvalCmd.Flags().BoolVarP(&explain, "explain", "e", false, "Render the derivation tree (proof) for the query instead of plain results")
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(EvalCmd)
}
