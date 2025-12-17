package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"

	mangleai "github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/providers/google"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/genkit"
	"github.com/joho/godotenv"
)

// ---------------------------------------------------------
// Data Structures
// ---------------------------------------------------------

// SolutionCandidates matches the JSON output from Gemini
type SolutionCandidates struct {
	Equation   string    `json:"equation"`
	Candidates []float64 `json:"candidates" description:"List of potential values for t"`
	Reasoning  string    `json:"reasoning"`
}

// VerificationResult is passed from Verifier to Printer
type VerificationResult struct {
	OriginalEquation string
	ValidSolutions   []float64
	InvalidSolutions []float64
}

// ---------------------------------------------------------
// Actions
// ---------------------------------------------------------

// 1. GeminiSolverAction: Uses "google gemini" to propose solutions.
type GeminiSolverAction struct {
	llm core.TextGenerator
}

func (a *GeminiSolverAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name:      "gemini_solver",
		InputType: "string",
		Type:      "model",
	}
}

func (a *GeminiSolverAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	problem, ok := env.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("expected string payload")
	}

	fmt.Printf("\n[Gemini Solver] Analyzing problem: %q\n", problem)

	sysPrompt := "You are a mathematical engine. Solve the given equation for variable 't'. " +
		"Find ALL real solutions, including negative ones. " +
		"Return a JSON object with the equation, a list of candidate values (floats), and brief reasoning."

	resp, err := mangleai.GenerateStruct[SolutionCandidates](
		ctx,
		a.llm,
		sysPrompt,
		problem,
	)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("gemini failed to solve: %w", err)
	}

	fmt.Printf("[Gemini Solver] Proposed Candidates: %v\n", resp.Candidates)

	// Output packet
	out := core.NewEnvelope(resp)
	out.Facts = []string{"status(\"proposed\")"} // Logic Engine trigger
	return out, nil
}

// 2. VerifierAction: Uses Go logic ("logic engine") to validate candidates mathematically.
type VerifierAction struct{}

func (v *VerifierAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name:      "verifier",
		InputType: "struct", // SolutionCandidates
		Type:      "computation",
	}
}

func (v *VerifierAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	// 1. Unpack Payload
	// Note: In a real system we'd check types safely.
	// Since we are in the same process, we can cast.
	// However, json unmarshalling might have made it a map if generic,
	// but here we passed the struct pointer in previous step?
	// Manglekit Envelope.Payload is `any`. In-memory it stays as struct.

	candidates, ok := env.Payload.(*SolutionCandidates)
	if !ok {
		// Try by value
		val, ok2 := env.Payload.(SolutionCandidates)
		if ok2 {
			candidates = &val
		} else {
			return core.Envelope{}, fmt.Errorf("verifier received invalid payload type: %T", env.Payload)
		}
	}

	fmt.Printf("[Verifier] Checking %d candidates for %s...\n", len(candidates.Candidates), candidates.Equation)

	result := VerificationResult{
		OriginalEquation: candidates.Equation,
		ValidSolutions:   []float64{},
		InvalidSolutions: []float64{},
	}

	// 2. Verification Logic: 2^t = t^2.
	// We allow a small epsilon for floating point math.
	epsilon := 1e-4

	for _, t := range candidates.Candidates {
		lhs := math.Pow(2, t)
		rhs := math.Pow(t, 2)
		diff := math.Abs(lhs - rhs)

		isSolution := diff < epsilon

		status := "❌"
		if isSolution {
			status = "✅"
			result.ValidSolutions = append(result.ValidSolutions, t)
		} else {
			result.InvalidSolutions = append(result.InvalidSolutions, t)
		}

		fmt.Printf("   -> t=%.4f | 2^t=%.4f, t^2=%.4f | Diff: %.6f %s\n", t, lhs, rhs, diff, status)
	}

	// Output
	out := core.NewEnvelope(result)
	out.Facts = []string{"status(\"verified\")"} // Logic Engine trigger
	return out, nil
}

// 3. PrinterAction: Final step to display results.
type PrinterAction struct{}

func (p *PrinterAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name:      "printer",
		InputType: "struct",
		Type:      "io",
	}
}

func (p *PrinterAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	res, ok := env.Payload.(VerificationResult) // Likely passed by value
	if !ok {
		// Handle ptr
		if ptr, ok2 := env.Payload.(*VerificationResult); ok2 {
			res = *ptr
		} else {
			return core.Envelope{}, fmt.Errorf("printer received invalid payload: %T", env.Payload)
		}
	}

	finalMsg := fmt.Sprintf("\n>>> [Logic Engine Result] \nEquation: %s\nVerified Solutions: %v\nRejected: %v\n",
		res.OriginalEquation, res.ValidSolutions, res.InvalidSolutions)

	fmt.Println(finalMsg)

	return core.NewEnvelope(finalMsg), nil
}

// ---------------------------------------------------------
// Main
// ---------------------------------------------------------

func main() {
	ctx := context.Background()
	_ = godotenv.Overload()

	// 0. Initialize Debug Logger
	// We use the DEBUG log level to show the "RunLoop" internals clearly.
	debugLogger := logger.New("DEBUG")

	// 1. Initialize Client with Protocol (Logic Engine)
	protocolPath := "examples/math_solver/protocol.dl"

	client, err := sdk.NewClient(ctx,
		sdk.WithBlueprintPath(protocolPath),
		sdk.WithFailMode("closed"),
		sdk.WithLogger(debugLogger), // Inject Logger
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 2. Initialize Google Gemini (The Brain)
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Println("GOOGLE_API_KEY is not set. Please set it to run this example.")
		os.Exit(1)
	}

	g := mangleai.GetGenkit(ctx)
	modelID := "gemini-2.5-flash"
	modelName, err := google.Init(ctx, g, apiKey, modelID)
	if err != nil {
		log.Fatalf("Failed to init google provider: %v", err)
	}
	model := genkit.LookupModel(g, modelName)
	if model == nil {
		log.Fatalf("Model %q not found", modelName)
	}
	gemini := mangleai.NewGenkitAdapter(model, g)

	// 3. Register Actions
	client.RegisterAction("gemini_solver", client.Supervise(&GeminiSolverAction{llm: gemini}))
	client.RegisterAction("verifier", client.Supervise(&VerifierAction{}))
	client.RegisterAction("printer", client.Supervise(&PrinterAction{}))

	// 4. Execute the Flow
	// We ask a query where approximations might occur.
	inputQuery := "Find t when 2^t=t^2"

	fmt.Printf("--- Starting Math Solver Agent (Debug Mode) ---\n")
	fmt.Printf("Query: %s\n", inputQuery)

	// Execute
	res, err := client.ExecuteByName(ctx, "gemini_solver", inputQuery)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// Dump final payload to confirm structure
	fmt.Printf("\nFinal System Payload: %+v\n", res.Payload)
}
