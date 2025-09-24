// Package main provides the HTTP server entry point for the Manglekit agent.
// It implements a simple REST API for processing queries through the Sandwich pattern.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ndduy.dev/manglekit/internal/agent"
	"ndduy.dev/manglekit/internal/llm"
	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/types"
)

// QueryRequest represents the incoming HTTP request structure.
type QueryRequest struct {
	User    string                 `json:"user"`
	Query   string                 `json:"query"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// QueryResponse represents the HTTP response structure.
type QueryResponse struct {
	Answer       string              `json:"answer"`
	Citations    []string            `json:"citations,omitempty"`
	Explanations []types.Explanation `json:"explanations,omitempty"`
	Error        string              `json:"error,omitempty"`
}

func main() {
	// Initialize components
	llmClient := llm.NewLLMFromEnv()
	mangleProcessor := mangle.NewMangle("internal/mangle/rules.mng", "internal/mangle/facts.json")
	retriever := agent.NewRetriever()
	orchestrator := agent.NewOrchestrator(mangleProcessor, retriever, llmClient)

	// Setup HTTP handlers
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/answer", answerHandler(orchestrator))

	// Setup server with timeouts
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// healthHandler provides a simple health check endpoint.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// answerHandler creates an HTTP handler for processing queries.
func answerHandler(orchestrator types.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Failed to decode request: %v", err)
			response := QueryResponse{Error: "Invalid JSON request"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Validate required fields
		if req.Query == "" {
			response := QueryResponse{Error: "Query field is required"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Prepare input for orchestrator
		input := &types.QueryInput{
			Query:       req.Query,
			UserContext: req.Context,
		}

		// Add user to context if provided
		if req.User != "" {
			if input.UserContext == nil {
				input.UserContext = make(map[string]interface{})
			}
			input.UserContext["user"] = req.User
		}

		// Process the query
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		result, err := orchestrator.RunFlow(ctx, input)
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			response := QueryResponse{Error: fmt.Sprintf("Processing failed: %v", err)}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Return successful response
		response := QueryResponse{
			Answer:       result.Answer,
			Citations:    result.Citations,
			Explanations: result.Explanations,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
