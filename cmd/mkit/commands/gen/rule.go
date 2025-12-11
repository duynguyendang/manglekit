package gen

import (
	"context"
	"fmt"
	"os"

	adapterai "github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/spf13/cobra"
)

var (
	provider string
	model    string
	prompt   string
)

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Generate a Datalog rule from a natural language prompt.",
	Long:  `Generates a Mangle Datalog rule based on a natural language policy description. This command leverages an LLM to translate the policy into formal Datalog syntax using a Neuro-Symbolic Feedback Loop (Teacher-Student Protocol).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if provider != "google" {
			return fmt.Errorf("unsupported LLM provider: %s. supported: google", provider)
		}

		apiKey := os.Getenv("GOOGLE_GENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("GOOGLE_API_KEY")
		}
		if apiKey == "" {
			return fmt.Errorf("GOOGLE_GENAI_API_KEY or GOOGLE_API_KEY environment variable is required")
		}

		// 1. Initialize Genkit
		// Using the pattern from examples/semantic_feedback/main.go
		gai := &googlegenai.GoogleAI{APIKey: apiKey}
		gk := genkit.Init(ctx, genkit.WithPlugins(gai))

		// Use googlegenai.GoogleAIModel to get the model instance
		rawModel := googlegenai.GoogleAIModel(gk, model)
		if rawModel == nil {
			return fmt.Errorf("model %s not found in googlegenai plugin", model)
		}

		adapter := adapterai.NewGenkitAdapter(rawModel, gk)

		// 2. Prepare Context (Known Facts)
		knownFacts := []string{}

		// 3. Execute
		result, err := GenerateWithFeedback(ctx, adapter, prompt, knownFacts)
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}

		// 4. Output
		fmt.Println("Explanation:", result.Explanation)
		fmt.Println("\n--- Datalog Code ---")
		fmt.Println(result.DatalogContent)

		return nil
	},
}

func init() {
	ruleCmd.Flags().StringVar(&provider, "provider", "google", "LLM Provider (e.g., google)")
	ruleCmd.Flags().StringVar(&model, "model", "", "Model name to use for generation (e.g., gemini-2.0-flash)")
	ruleCmd.Flags().StringVar(&prompt, "prompt", "", "The natural language policy description")

	ruleCmd.MarkFlagRequired("model")
	ruleCmd.MarkFlagRequired("prompt")

	GenCmd.AddCommand(ruleCmd)
}
