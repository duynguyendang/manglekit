package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit-wip/adapters/knowledge"
	"github.com/duynguyendang/manglekit-wip/internal/engine"
	"github.com/spf13/cobra"
)

var (
	policyPath    string
	dataPath      string
	knowledgePath string
	queryString   string
)

var EvalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate a Datalog query against policy and data",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validation
		if dataPath == "" && knowledgePath == "" {
			return fmt.Errorf("must provide either --data (JSON) or --facts/--knowledge (Graph)")
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
				fmt.Printf("Injected %d schema declarations.\n", len(preds))
			}

			// Load Facts
			facts := knowledge.TriplesToFacts(triples)
			if err := e.LoadFacts(facts); err != nil {
				return fmt.Errorf("failed to load knowledge facts: %w", err)
			}
			fmt.Printf("Loaded %d knowledge facts.\n", len(facts))
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

			if err := e.LoadFacts(dataFacts); err != nil {
				return fmt.Errorf("failed to load data facts: %w", err)
			}
		}

		// 6. Execute Query
		ctx := context.Background()
		// Using Query instead of ExecuteQuery to get results/bindings
		results, err := e.Query(ctx, nil, queryString)
		if err != nil {
			return fmt.Errorf("query execution failed: %w", err)
		}

		// 7. Output
		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		// Print as JSON array for nice formatting
		outputBytes, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
		fmt.Println(string(outputBytes))

		return nil
	},
}

func init() {
	EvalCmd.Flags().StringVarP(&policyPath, "policy", "p", "", "Path to the .dl file")
	EvalCmd.MarkFlagRequired("policy")

	EvalCmd.Flags().StringVarP(&dataPath, "data", "d", "", "Path to the .json input file. Optional if --knowledge is provided.")

	EvalCmd.Flags().StringVarP(&knowledgePath, "knowledge", "k", "", "Path to the .nq, .nt, or .ttl knowledge base")
	EvalCmd.Flags().StringVar(&knowledgePath, "facts", "", "Alias for --knowledge")

	EvalCmd.Flags().StringVarP(&queryString, "query", "q", "", "The Datalog query to execute")
	EvalCmd.MarkFlagRequired("query")
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(EvalCmd)
}
