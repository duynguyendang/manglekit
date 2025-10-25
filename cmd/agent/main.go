package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/all"
	"gopkg.in/yaml.v3"
)

func main() {
	// a. Parse command-line flags.
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	log.Printf("Starting Manglekit agent with config: %s", *configFile)

	// b. Read the configuration file.
	configData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// c. Create a new registry.
	registry := manglekit.NewRegistry()

	// d. Parse the config to get the orchestrator name.
	var cfg config.Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Error unmarshalling config: %v", err)
	}

	// e. Create a new builder and load from config.
	builder := manglekit.NewBuilder(registry).WithHandlers(all.ComponentHandlers()...)
	orchestrator, _, err := builder.FromConfig(context.Background(), configData)
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}

	// f. Set up the HTTP handler.
	mux := http.NewServeMux()
	mux.HandleFunc("/query", queryHandler(orchestrator))

	// g. Set up the HTTP server struct.
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// h. Start the server in a separate goroutine.
	go func() {
		log.Println("Server starting on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
		}
	}()

	// i. Set up and block on the graceful shutdown signal handler.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutdown signal received, starting graceful shutdown...")

	// Create a context with a timeout for the shutdown process.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shut down the HTTP server first.
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server gracefully stopped")
	}

	// Only after the server is down, close the orchestrator's resources.
	if err := orchestrator.Close(shutdownCtx); err != nil {
		log.Printf("Orchestrator resource cleanup error: %v", err)
	} else {
		log.Println("Orchestrator resources gracefully closed")
	}

	log.Println("Shutdown complete.")
}

func queryHandler(orchestrator core.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var query core.Query
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Use the request's context to support cancellation.
		sessionID := "" // Or extract from headers/body if needed
		answer, err := orchestrator.Execute(r.Context(), sessionID, query)
		if err != nil {
			log.Printf("Error executing orchestrator: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(answer); err != nil {
			log.Printf("Error encoding response: %v", err)
			// Don't write another http.Error as the header is likely already written.
		}
	}
}
