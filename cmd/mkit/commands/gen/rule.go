package gen

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	adapterai "github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/sdk"
	genkit "github.com/firebase/genkit/go/ai"
	"github.com/spf13/cobra"
)

var (
	provider string
	model    string
	schema   string
	prompt   string
	out      string
	ruleHead string
)

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Generate a Datalog rule from a natural language prompt and a JSON schema.",
	Long:  `Generates a Mangle Datalog rule based on a natural language policy description and a sample JSON schema. This command leverages an LLM to translate the policy into formal Datalog syntax. Dogfooding core.Action for universal work execution.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize the LLM model based on the provider
		var genKitModel genkit.Model
		switch provider {
		case "google":
			// Use genkit's built-in Google AI integration
			// This requires GOOGLE_GENAI_API_KEY environment variable
			// For production use, initialize genkit properly with a registry
			// For now, we'll create a direct reference using the model name
			genKitModel = genkit.NewModel(model, nil, nil)
			if genKitModel == nil {
				return fmt.Errorf("failed to initialize google model '%s'. ensure genkit google plugin is initialized", model)
			}

		default:
			return fmt.Errorf("unsupported LLM provider: %s. supported: google", provider)
		}

		// Create a core.Action from the Genkit model using adapters/ai (dogfooding)
		// This wraps the Genkit model with GenkitGenerator, then LLMAction, providing core.Action interface
		llmAction := adapterai.NewGenkitAction(fmt.Sprintf("%s-%s", provider, model), genKitModel)

		// Read schema from file
		schemaData, err := os.ReadFile(schema)
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}

		var schemaSample map[string]any
		if err := json.Unmarshal(schemaData, &schemaSample); err != nil {
			return fmt.Errorf("failed to unmarshal schema JSON: %w", err)
		}

		// Create generator with the core.Action
		opts := sdk.GeneratorOptions{
			RuleHead: ruleHead,
		}

		generator, err := sdk.NewPolicyGenerator(llmAction, opts)
		if err != nil {
			return fmt.Errorf("failed to create generator: %w", err)
		}

		// Generate the Datalog rule using the action
		datalogRule, err := generator.GenerateRule(ctx, schemaSample, prompt)
		if err != nil {
			return fmt.Errorf("failed to generate rule: %w", err)
		}

		// Output the result
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
	ruleCmd.Flags().StringVar(&provider, "provider", "google", "LLM Provider (e.g., google)")
	ruleCmd.Flags().StringVar(&model, "model", "", "Model name to use for generation (e.g., gemini-2.0-flash)")
	ruleCmd.Flags().StringVar(&schema, "schema", "", "Path to a JSON schema sample file")
	ruleCmd.Flags().StringVar(&prompt, "prompt", "", "The natural language policy description")
	ruleCmd.Flags().StringVar(&ruleHead, "rule-head", "deny(Req)", "Target rule predicate (e.g., deny(Req), allow(Req), route(Req, Target))")
	ruleCmd.Flags().StringVar(&out, "out", "", "Output file path (default: stdout)")

	ruleCmd.MarkFlagRequired("model")
	ruleCmd.MarkFlagRequired("schema")
	ruleCmd.MarkFlagRequired("prompt")

	GenCmd.AddCommand(ruleCmd)
}
