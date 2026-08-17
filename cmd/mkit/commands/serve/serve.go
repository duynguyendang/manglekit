package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	port       string
	policyPath string
	mcpConfig  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Manglekit HTTP Server",
	Long:  `Exposes the Manglekit SDK via an HTTP API, enforcing governance policies on every request.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer(cmd.Context())
	},
}

func AddCommands(root *cobra.Command) {
	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on")
	serveCmd.Flags().StringVarP(&policyPath, "policy", "f", "", "Path to the Datalog policy file")
	serveCmd.Flags().StringVarP(&mcpConfig, "mcp", "m", "", "Path to MCP configuration file")
	root.AddCommand(serveCmd)
}

func runServer(ctx context.Context) {
	opts := []sdk.ClientOption{}
	if policyPath != "" {
		opts = append(opts, sdk.WithBlueprintPath(policyPath))
	}

	client, err := sdk.NewClient(ctx, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize client: %v\n", err)
		os.Exit(1)
	}

	// Hot policy reload: SIGHUP re-reads the policy file and atomically
	// swaps it in. A failed reload (parse/evaluation error) keeps the old
	// policy serving; in-flight requests are unaffected.
	if policyPath != "" {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP)
		go func() {
			for range sigCh {
				if err := client.ReloadPolicy(ctx, policyPath); err != nil {
					fmt.Fprintf(os.Stderr, "Policy reload failed (keeping old policy): %v\n", err)
				} else {
					fmt.Printf("Policy reloaded from %s\n", policyPath)
				}
			}
		}()
	}

	srv := &http.Server{Addr: ":" + port, Handler: createHandler(client)}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Manglekit HTTP Server listening on :%s\n", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
		os.Exit(1)
	}
}

// Handler handles the HTTP request.
// It is separated for testability.
func createHandler(client *sdk.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var envelope core.Envelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		// FIX: Initialize metadata if nil
		if envelope.Metadata == nil {
			envelope.Metadata = make(map[string]any)
		}

		// FIX: Generate UUID if missing
		if envelope.ID == uuid.Nil {
			envelope.ID = uuid.New()
		}

		// Execution
		result, err := client.Execute(r.Context(), envelope)

		// Response Mapping
		if err != nil {
			// Check for Policy Violation (AlignmentError)
			if core.IsAlignmentError(err) {
				// Case B: Policy Violation
				w.WriteHeader(http.StatusForbidden)

				// Construct JSON body with deny reasons
				var alignErr *core.AlignmentError
				errors.As(err, &alignErr)

				// Create a structured response
				resp := map[string]any{
					"error":    "Policy Violation",
					"reasons":  []string{alignErr.Message},
					"rule_id":  alignErr.RuleID,
					"decision": core.DecisionHalt,
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			// Case A: Internal Error
			http.Error(w, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
			return
		}

		// Case B: Policy Violation (Check Metadata)
		// Even if err is nil, check decision metadata
		if d, ok := result.Metadata[core.KeyDecision]; ok && d == core.DecisionHalt {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(result) // Return full result as body
			return
		}

		// Case C: Success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}
