package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/duynguyendang/manglekit/adapters/knowledge"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/exitcode"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/spf13/cobra"
)

var (
	policyPath string
	dataPath   string
	targets    string
	outputPath string
	format     string
)

// Supported --format values.
const (
	formatNQuads = "nquads"
	formatJSON   = "json"
)

// derivedFact is one materialized fact, serialized as an N-Quad line or a
// JSON object depending on --format.
type derivedFact struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
}

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Perform batch inference (Logic ETL)",
	Long:  `Perform batch inference by loading raw data, applying Datalog rules, and exporting derived facts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch format {
		case formatNQuads, formatJSON:
		default:
			return exitcode.UsageErrorf("invalid --format %q: must be %q or %q", format, formatNQuads, formatJSON)
		}

		// 1. Init Engine
		eng, err := engine.New()
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		// 2. Load Policy
		policyBytes, err := os.ReadFile(policyPath)
		if err != nil {
			return fmt.Errorf("failed to read policy file: %w", err)
		}
		if err := eng.LoadPolicy(cmd.Context(), string(policyBytes)); err != nil {
			return fmt.Errorf("failed to load policy: %w", err)
		}

		// 3. Load Data
		var facts []string
		ext := strings.ToLower(filepath.Ext(dataPath))
		if ext == ".json" {
			dataBytes, err := os.ReadFile(dataPath)
			if err != nil {
				return fmt.Errorf("failed to read data file: %w", err)
			}
			var input any
			if err := json.Unmarshal(dataBytes, &input); err != nil {
				return fmt.Errorf("failed to unmarshal JSON: %w", err)
			}
			// Assume root ID is "root" or similar.
			// The original requirement says: "Assume root ID is "root" for now, or generate UUID".
			// We'll use "root" for consistency with the skeleton.
			facts, err = engine.Flatten("root", input)
			if err != nil {
				return fmt.Errorf("failed to flatten JSON data: %w", err)
			}
		} else if ext == ".nq" || ext == ".nt" || ext == ".ttl" {
			// Use adapters/knowledge.ParseGraphFile
			triples, err := knowledge.ParseGraphFile(dataPath)
			if err != nil {
				return fmt.Errorf("failed to parse graph file: %w", err)
			}
			// We need to inject Decls for predicates if Mangle file-mode is strict,
			// but here we are just loading facts.
			// However, if the policy refers to them, they should be fine as EDBs.
			// Note: Eval command injects declarations.
			// The requirement for 'run' says "Inference (Materialization)".
			// If we just load facts, Mangle should accept them.
			facts = knowledge.TriplesToFacts(triples)
		} else {
			return fmt.Errorf("unsupported data file extension: %s", ext)
		}

		if err := eng.LoadFacts(cmd.Context(), facts); err != nil {
			return fmt.Errorf("failed to load facts: %w", err)
		}

		// 4. Inference: materialize derived facts for each target.
		targetList := strings.Split(targets, ",")
		ctx := cmd.Context()

		var derived []derivedFact
		for _, target := range targetList {
			target = strings.TrimSpace(target)
			// Heuristic: Try Arity 2 first: target(S, O)
			query := fmt.Sprintf("%s(S, O)", target)
			results, err := eng.Query(ctx, nil, query)

			// If Arity 2 yields results, collect them
			if err == nil && len(results) > 0 {
				for _, row := range results {
					derived = append(derived, derivedFact{
						Subject:   row["S"],
						Predicate: target,
						Object:    row["O"],
					})
				}
				continue
			}

			// Try Arity 1: target(S)
			query = fmt.Sprintf("%s(S)", target)
			results, err = eng.Query(ctx, nil, query)
			if err == nil && len(results) > 0 {
				for _, row := range results {
					// Arity 1: boolean flag style (object is "true")
					derived = append(derived, derivedFact{
						Subject:   row["S"],
						Predicate: target,
						Object:    "true",
					})
				}
				continue
			}
		}

		// 5. Emit output in the requested format.
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()

		switch format {
		case formatJSON:
			if derived == nil {
				derived = []derivedFact{}
			}
			out, err := json.MarshalIndent(derived, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON output: %w", err)
			}
			if _, err := f.Write(append(out, '\n')); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
		default: // formatNQuads
			for _, d := range derived {
				// N-Quad line: <Subject> <Predicate> "Object" .
				if _, err := fmt.Fprintf(f, "<%s> <%s> %q .\n", d.Subject, d.Predicate, d.Object); err != nil {
					return fmt.Errorf("failed to write output: %w", err)
				}
			}
		}

		fmt.Printf("Inference complete. Results written to %s\n", outputPath)
		return nil
	},
}

func init() {
	RunCmd.Flags().StringVarP(&policyPath, "policy", "p", "", "Path to .dl file")
	RunCmd.Flags().StringVarP(&dataPath, "data", "d", "", "Path to input file (.json or .nq/nt)")
	RunCmd.Flags().StringVarP(&targets, "target", "t", "", "Comma-separated list of predicates to infer")
	RunCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	RunCmd.Flags().StringVar(&format, "format", formatNQuads, "Output format (nquads, json)")

	// Required flags: cobra enforces these before RunE executes, producing
	// consistent usage errors instead of ad-hoc manual checks.
	cobra.CheckErr(RunCmd.MarkFlagRequired("policy"))
	cobra.CheckErr(RunCmd.MarkFlagRequired("data"))
	cobra.CheckErr(RunCmd.MarkFlagRequired("target"))
	cobra.CheckErr(RunCmd.MarkFlagRequired("output"))
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RunCmd)
}
