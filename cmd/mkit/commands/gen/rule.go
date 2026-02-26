package gen

import (
	"context"
	"fmt"
	"os"

	adapterai "github.com/duynguyendang/manglekit-wip/adapters/ai"
	"github.com/duynguyendang/manglekit-wip/cmd/mkit/commands/gen/inductor"
	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
)

var (
	provider string
	model    string
	prompt   string
	desc     string
	vocab    []string
	sample   string
	iclFlag  string
)

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Generate a Datalog rule from a natural language prompt.",
	Long:  `Generates a Mangle Datalog rule based on a natural language policy description. This command leverages an LLM to translate the policy into formal Datalog syntax using a Neuro-Symbolic Feedback Loop (Teacher-Student Protocol).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var rawModel genkitai.Model
		var gk *genkit.Genkit

		if provider == "google" {
			apiKey := os.Getenv("GOOGLE_GENAI_API_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("GOOGLE_API_KEY")
			}
			if apiKey == "" {
				return fmt.Errorf("GOOGLE_GENAI_API_KEY or GOOGLE_API_KEY environment variable is required")
			}
			gai := &googlegenai.GoogleAI{APIKey: apiKey}
			gk = genkit.Init(ctx, genkit.WithPlugins(gai))
			rawModel = googlegenai.GoogleAIModel(gk, model)
		} else if provider == "openai" {
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				return fmt.Errorf("OPENAI_API_KEY environment variable is required")
			}
			baseURL := os.Getenv("OPENAI_BASE_URL")

			oai := &openai.OpenAI{APIKey: apiKey}
			if baseURL != "" {
				oai.Opts = []option.RequestOption{option.WithBaseURL(baseURL)}
			}

			gk = genkit.Init(ctx, genkit.WithPlugins(oai))
			rawModel = oai.Model(gk, model)

			// If model is not found (e.g. custom OpenRouter/LocalAI model), define it dynamically
			if rawModel == nil {
				rawModel = oai.DefineModel(model, genkitai.ModelOptions{
					Label: model,
					Supports: &genkitai.ModelSupports{
						Multiturn:  true,
						SystemRole: true,
						Tools:      false,
						Media:      false,
					},
				})
			}
		} else {
			return fmt.Errorf("unsupported LLM provider: %s. supported: google, openai", provider)
		}

		if rawModel == nil {
			return fmt.Errorf("model %s not found in provider %s", model, provider)
		}

		adapter := adapterai.NewGenkitAdapter(rawModel, gk)

		// Determine prompt
		finalPrompt := prompt
		if desc != "" {
			finalPrompt = desc
		}
		if finalPrompt == "" {
			return fmt.Errorf("either --prompt or --desc must be provided")
		}

		var schema *inductor.SchemaHint
		if sample != "" {
			var err error
			schema, err = inductor.InferFromFile(sample)
			if err != nil {
				return fmt.Errorf("failed to infer schema from sample file: %w", err)
			}
			fmt.Printf("Detected schema from %s (%s)\n", sample, schema.FileType)
		}

		// 3. Prepare ICL
		iclContent, err := GetICLContent(iclFlag)
		if err != nil {
			return fmt.Errorf("failed to load ICL content: %w", err)
		}

		// 3. Execute
		result, err := GenerateWithFeedback(ctx, adapter, finalPrompt, vocab, schema, iclContent)
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
	ruleCmd.Flags().StringVar(&provider, "provider", "google", "LLM Provider (e.g., google, openai)")
	ruleCmd.Flags().StringVar(&model, "model", "", "Model name to use for generation (e.g., gemini-2.0-flash, gpt-4o)")
	ruleCmd.Flags().StringVar(&prompt, "prompt", "", "The natural language policy description")
	ruleCmd.Flags().StringVar(&desc, "desc", "", "The natural language policy description (alias for --prompt)")
	ruleCmd.Flags().StringSliceVar(&vocab, "vocab", []string{}, "Domain vocabulary (e.g., predicates like is_vip(User))")
	ruleCmd.Flags().StringSliceVar(&vocab, "facts", []string{}, "Alias for --vocab")
	ruleCmd.Flags().StringVar(&sample, "sample", "", "Path to a sample file (.json, .nq, .nt, .ttl) to infer schema")
	ruleCmd.Flags().StringVar(&iclFlag, "icl", "", "Path to a custom Datalog example file (optional). Defaults to built-in golden rules.")

	ruleCmd.MarkFlagRequired("model")

	GenCmd.AddCommand(ruleCmd)
}
