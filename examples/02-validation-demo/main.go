package main

import (
"context"
"fmt"
"log"
"os"

"github.com/duynguyendang/manglekit/core"
"github.com/duynguyendang/manglekit/internal/providers/llm"
"github.com/duynguyendang/manglekit/internal/providers/retrievers/bm25"
"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
"github.com/duynguyendang/manglekit/pipeline/sandwich"
"github.com/duynguyendang/manglekit/sdk"
"github.com/joho/godotenv"
)

func main() {
err := godotenv.Load()
if err != nil {
log.Println("Note: .env file not found, will fall back to environment variables")
}

// DEMO: Show how the builder now validates provider dependencies

ctx := context.Background()

// Create a new programmatic builder
builder, err := sdk.NewBuilder(ctx)
if err != nil {
log.Fatalf("Failed to create builder: %v", err)
}

// Configure components
bm25Opts := &bm25.BM25Options{
Path: "examples/01-programmatic-setup/docs",
}

// Google LLM - requires GOOGLE_API_KEY
googleOpts := &llm.GoogleOptions{
Model:          "gemini-2.5-flash",
PromptTemplate: "Explain in details ",
}

mangleOpts := &core.MangleOptions{
Path:              []string{"examples/rules/acme-rules.dlog"},
DefaultConverters: true,
}

stateOpts := &inmemory.Options{}
sandwichOpts := &sandwich.Options{
LLM:           "google",
Retriever:     "bm25",
RuleSet:       "mangle",
StateProvider: "inmemory",
}

// Configure with the builder
// NEW FEATURE: The builder now validates provider dependencies as you add them
fmt.Println("=== Configuring Builder with Provider Dependency Validation ===\n")

builder.WithOptions("bm25", bm25Opts)
fmt.Println("✓ Configured: bm25 retriever (no special requirements)")

builder.WithOptions("mangle", mangleOpts)
fmt.Println("✓ Configured: mangle rules (no special requirements)")

builder.WithOptions("inmemory", stateOpts)
fmt.Println("✓ Configured: inmemory state provider (no special requirements)")

fmt.Println("\nConfiguring: google LLM (requires GOOGLE_API_KEY)...")
builder.WithOptions("google", googleOpts)

// Check if GOOGLE_API_KEY is set
if os.Getenv("GOOGLE_API_KEY") == "" {
fmt.Println("⚠️  GOOGLE_API_KEY not set - builder has recorded this as an error")
fmt.Println("    The Build() method will fail with a clear message")
} else {
fmt.Println("✓ GOOGLE_API_KEY detected - validation passed")
}

builder.WithOptions("sandwich", sandwichOpts)
fmt.Println("✓ Configured: sandwich orchestrator\n")

// Now try to build - this is where validation errors surface
fmt.Println("=== Attempting to Build ===\n")

orch, _, err := builder.Build(ctx, "sandwich", "")
if err != nil {
fmt.Printf("❌ Build failed with error:\n    %v\n\n", err)
fmt.Println("This is expected if GOOGLE_API_KEY is not set!")
fmt.Println("The builder caught the missing dependency early in the validation process.\n")

// Demonstrate the fix
fmt.Println("=== To Fix This ===")
fmt.Println("1. Set your Google API key:")
fmt.Println("   export GOOGLE_API_KEY=\"AIzaSy...\"")
fmt.Println("2. Or reconfigure to use a different LLM provider (e.g., OpenAI)")
fmt.Println("3. Or if you don't need LLM, remove the google provider configuration\n")
return
}

fmt.Println("✓ Build successful!\n")

// If we get here, everything is configured correctly
defer func() {
if err := orch.Close(ctx); err != nil {
log.Printf("Warning: Error closing orchestrator: %v", err)
}
}()

query := core.Query{
Text: "What is manglekit?",
}
log.Printf("Executing query: %s\n", query.Text)

answer, err := orch.Execute(ctx, "session-123", query)
if err != nil {
log.Fatalf("Failed to execute query: %v", err)
}

fmt.Printf("\nAnswer: %s\n", answer.Text)
if len(answer.Citations) > 0 {
fmt.Println("\nCitations:")
for _, citation := range answer.Citations {
fmt.Printf("  - %s (Source: %s)\n", citation.ID, citation.Source)
}
}
}
