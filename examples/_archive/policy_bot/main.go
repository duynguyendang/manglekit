package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/duynguyendang/manglekit/providers/google"
	"github.com/duynguyendang/manglekit/providers/memory/inmem"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Setup Environment
	ctx := context.Background()
	_ = godotenv.Overload()
	if os.Getenv("GOOGLE_API_KEY") == "" {
		log.Fatal("Please set GOOGLE_API_KEY environment variable")
	}

	// 2. Register Plugins
	// Note: Google is auto-wired via SDK Config Loader
	inmem.Register()

	// 3. Initialize Client
	fmt.Println("🚀 Initializing Policy Bot...")
	// Try to locate the specific config file for this example
	configPath := "mangle.yaml"
	if _, err := os.Stat("examples/policy_bot/mangle.yaml"); err == nil {
		configPath = "examples/policy_bot/mangle.yaml"
	}

	// Use NewClientFromFile for loading from a YAML file path.
	client, err := sdk.NewClientFromFile(ctx, configPath)
	if err != nil {
		log.Fatalf("Failed to init client: %v", err)
	}

	// 4. [SEEDING] Teach the Bot some private knowledge
	// In a real app, this would happen via a separate ingestion pipeline.
	seedKnowledge(ctx, client)

	// 5. Ask a question that requires this knowledge
	question := "Am I allowed to work from home (WFH) on Friday?"
	fmt.Printf("\n❓ User: %s\n", question)

	// 6. Execute
	// The SDK will automatically:
	// - Recall: Find the WFH policy
	// - Inject: Add it to the prompt
	// - Execute: Call Gemini
	//
	// Workaround: Manually inject context until SDK Loop is fixed
	if mem := client.Memory(); mem != nil {
		contextData, err := mem.Recall(ctx, question)
		if err == nil && contextData != "" {
			fmt.Printf("\n[Debug] Used Context:\n%s\n", contextData)
			question = fmt.Sprintf("CONTEXT:\n%s\n\nQUESTION:\n%s", contextData, question)
		}
	}

	// Note: We use the alias defined in mangle.yaml
	resp, err := client.ExecuteByName(ctx, "chat_policy", question)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// 7. Output Result
	fmt.Printf("🤖 Bot: %v\n", resp.Payload)
}

func seedKnowledge(ctx context.Context, c *sdk.Client) {
	mem := c.Memory()
	if mem == nil {
		log.Println("⚠️ Memory is not enabled!")
		return
	}

	fmt.Println("📚 Seeding knowledge base...")

	// Knowledge 1: WFH Policy
	// Note: We use simple text for InMem keyword search.
	err := mem.Memorize(ctx,
		"remote work WFH policy",
		"FACT: Employees are allowed to WFH up to 2 days/week. However, FRIDAY is the all-hands meeting day, attendance at the office is mandatory, WFH is not allowed.",
	)
	if err != nil {
		log.Printf("Failed to seed WFH policy: %v", err)
	}

	// Knowledge 2: Expense Policy
	mem.Memorize(ctx,
		"grab taxi expense reimbursement policy",
		"FACT: The company only reimburses Grab expenses for rides after 10 PM or for client meetings.",
	)

	// Sleep briefly to ensure async write finishes (though inmem is fast)
	time.Sleep(100 * time.Millisecond)
	fmt.Println("✅ Knowledge loaded!")
}
