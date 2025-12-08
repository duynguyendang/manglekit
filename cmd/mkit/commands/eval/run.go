package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/adapters/knowledge"
	"github.com/duynguyendang/manglekit/internal/engine"
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
		// 1. Read Files
		policyBytes, err := os.ReadFile(policyPath)
		if err != nil {
			return fmt.Errorf("failed to read policy file: %w", err)
		}

		dataBytes, err := os.ReadFile(dataPath)
		if err != nil {
			return fmt.Errorf("failed to read data file: %w", err)
		}

		// 2. Initialize Engine
		// Using New() as we need a basic engine.
		e := engine.New()

		// 3. Load Policy
		if err := e.LoadPolicy(string(policyBytes)); err != nil {
			return fmt.Errorf("failed to load policy: %w", err)
		}

		// 4. Load Knowledge (Optional)
		if knowledgePath != "" {
			kFile, err := os.Open(knowledgePath)
			if err != nil {
				return fmt.Errorf("failed to open knowledge file: %w", err)
			}
			defer kFile.Close()

			loader := knowledge.NewNQuadsLoader()
			facts, err := loader.Parse(kFile)
			if err != nil {
				return fmt.Errorf("failed to parse knowledge base: %w", err)
			}

			if err := e.LoadFacts(facts); err != nil {
				return fmt.Errorf("failed to load knowledge facts: %w", err)
			}
		}

		// 5. Inject Data
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

	EvalCmd.Flags().StringVarP(&dataPath, "data", "d", "", "Path to the .json input file")
	EvalCmd.MarkFlagRequired("data")

	EvalCmd.Flags().StringVarP(&knowledgePath, "knowledge", "k", "", "Path to the .nq or .nt knowledge base")

	EvalCmd.Flags().StringVarP(&queryString, "query", "q", "", "The Datalog query to execute")
	EvalCmd.MarkFlagRequired("query")
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(EvalCmd)
}
