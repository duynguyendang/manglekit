package gen

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/policy/rulegenerator"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/spf13/cobra"
)

var (
	provider string
	model    string
	schema   string
	prompt   string
	out      string
)

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Generate a Datalog rule from a natural language prompt and a JSON schema.",
	Long:  `Generates a Mangle Datalog rule based on a natural language policy description and a sample JSON schema. This command leverages an LLM to translate the policy into formal Datalog syntax.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var llmProvider core.LLMClient

		// Use the SDK's programmatic builder to correctly initialize the LLM with all its dependencies.
		builder, err := sdk.NewBuilder(ctx)
		if err != nil {
			return fmt.Errorf("failed to create builder: %w", err)
		}

		llmName := fmt.Sprintf("mkit-%s-llm", provider)

		switch provider {
		case "google":
			if err := builder.WithOptions(llmName, &llm.GoogleOptions{Model: model}); err != nil {
				return fmt.Errorf("failed to configure google llm: %w", err)
			}
		// NOTE: To add openai, you would add a case here and the corresponding Options struct.
		// case "openai":
		// 	if err := builder.WithOptions(llmName, &llm.OpenAIOptions{Model: model}); err != nil {
		// 		return fmt.Errorf("failed to configure openai llm: %w", err)
		// 	}
		default:
			return fmt.Errorf("unsupported LLM provider: %s", provider)
		}

		resolved, err := builder.Build(ctx)
		if err != nil {
			return fmt.Errorf("failed to build components: %w", err)
		}

		var ok bool
		llmProvider, ok = resolved.LLMs[llmName]
		if !ok {
			return fmt.Errorf("could not find built llm named '%s'", llmName)
		}

		schemaData, err := os.ReadFile(schema)
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}

		var schemaSample map[string]any
		if err := json.Unmarshal(schemaData, &schemaSample); err != nil {
			return fmt.Errorf("failed to unmarshal schema JSON: %w", err)
		}

		generator := rulegenerator.New(llmProvider)
		datalogRule, err := generator.Generate(ctx, prompt, schemaSample)
		if err != nil {
			return fmt.Errorf("failed to generate rule: %w", err)
		}

		if out != "" {
			err = os.WriteFile(out, []byte(datalogRule), 0644)
			if err != nil {
				return fmt.Errorf("failed to write rule to file: %w", err)
			}
			fmt.Printf("Datalog rule successfully written to %s\n", out)
		} else {
			fmt.Println(datalogRule)
		}

		return nil
	},
}

func init() {
	ruleCmd.Flags().StringVar(&provider, "provider", "google", "LLM Provider (e.g., google, openai)")
	ruleCmd.Flags().StringVar(&model, "model", "", "Model name to use for generation")
	ruleCmd.Flags().StringVar(&schema, "schema", "", "Path to a JSON schema sample file")
	ruleCmd.Flags().StringVar(&prompt, "prompt", "", "The natural language policy description")
	ruleCmd.Flags().StringVar(&out, "out", "", "Output file path (default: stdout)")

	ruleCmd.MarkFlagRequired("model")
	ruleCmd.MarkFlagRequired("schema")
	ruleCmd.MarkFlagRequired("prompt")

	GenCmd.AddCommand(ruleCmd)
}
