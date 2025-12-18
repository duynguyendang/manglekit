package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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
	question := "Tôi có được phép làm việc từ xa (WFH) vào thứ Sáu không?"
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

	// Note: Currently, the SDK registers Google Actions by their model name (googleai/...), ignoring the config alias.
	resp, err := client.ExecuteByName(ctx, "googleai/gemini-2.5-flash", question)
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
		"chính sách làm việc từ xa WFH remote work",
		"FACT: Nhân viên được phép WFH tối đa 2 ngày/tuần. Tuy nhiên, THỨ SÁU là ngày họp toàn công ty, bắt buộc phải lên văn phòng, không được WFH.",
	)
	if err != nil {
		log.Printf("Failed to seed WFH policy: %v", err)
	}

	// Knowledge 2: Expense Policy
	mem.Memorize(ctx,
		"chính sách hoàn tiền grab taxi expense",
		"FACT: Công ty chỉ hoàn tiền Grab sau 22h đêm hoặc đi gặp khách hàng.",
	)

	// Sleep briefly to ensure async write finishes (though inmem is fast)
	time.Sleep(100 * time.Millisecond)
	fmt.Println("✅ Knowledge loaded!")
}
