package serve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/duynguyendang/manglekit/adapters/mcp"
	"github.com/duynguyendang/manglekit/config"
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
	Short: "Start the Manglekit HTTP Sidecar Server",
	Long: `Exposes the Manglekit SDK via an HTTP API, enforcing governance policies on every request.

Endpoints:
  POST /v1/assess        Evaluate action, facts, and payload against the policy gate
  POST /v1/explain       Render policy derivation tree for an explain query
  GET  /v1/health        Health check and policy status
  GET  /v1/policy        Loaded policy fingerprint (SHA-256)
  GET  /v1/openapi.json  OpenAPI 3.1 wire specification

Exit codes follow the mkit contract: runtime failures return exit 3
(serve never exits 1 — policy denies are answered as HTTP 403).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(cmd.Context())
	},
}

func AddCommands(root *cobra.Command) {
	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on")
	serveCmd.Flags().StringVarP(&policyPath, "policy", "f", "", "Path to the Datalog policy file")
	serveCmd.Flags().StringVarP(&mcpConfig, "mcp", "m", "", "Path to MCP configuration file (JSON array of MCP servers; discovered tools are registered as policy-supervised actions)")
	root.AddCommand(serveCmd)
}

func computeFileHash(path string) string {
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// ServerState tracks the active client, policy path, and cryptographic policy fingerprint.
type ServerState struct {
	Client     *sdk.Client
	PolicyPath string
	PolicyHash atomic.Value // string
}

func (s *ServerState) GetPolicyHash() string {
	if val := s.PolicyHash.Load(); val != nil {
		return val.(string)
	}
	return ""
}

func (s *ServerState) SetPolicyHash(h string) {
	s.PolicyHash.Store(h)
}

func loadMCPActions(ctx context.Context, path string) ([]core.Action, error) {
	return loadMCPActionsWithFactory(ctx, path, &mcp.DefaultFactory{})
}

func loadMCPActionsWithFactory(ctx context.Context, path string, factory mcp.ClientFactory) ([]core.Action, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config %q: %w", path, err)
	}
	var servers []config.MCPServerConfig
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config %q (expecting a JSON array of MCP servers): %w", path, err)
	}
	var actions []core.Action
	for _, srv := range servers {
		loaded, err := mcp.NewLoader(srv).WithFactory(factory).Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("MCP server %q: %w", srv.Name, err)
		}
		actions = append(actions, loaded...)
	}
	return actions, nil
}

func runServer(ctx context.Context) error {
	opts := []sdk.ClientOption{}
	if policyPath != "" {
		opts = append(opts, sdk.WithPolicyPath(policyPath))
	}

	client, err := sdk.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	state := &ServerState{
		Client:     client,
		PolicyPath: policyPath,
	}
	state.SetPolicyHash(computeFileHash(policyPath))

	if mcpConfig != "" {
		mcpActions, err := loadMCPActions(ctx, mcpConfig)
		if err != nil {
			return err
		}
		for _, act := range mcpActions {
			client.RegisterSupervised(act.Metadata().Name, act)
		}
		fmt.Printf("Registered %d MCP tool action(s) from %s\n", len(mcpActions), mcpConfig)
	}

	// Hot policy reload: SIGHUP re-reads the policy file and atomically swaps it in.
	if policyPath != "" {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP)
		go func() {
			for range sigCh {
				if err := client.ReloadPolicy(ctx, policyPath); err != nil {
					fmt.Fprintf(os.Stderr, "Policy reload failed (keeping old policy): %v\n", err)
				} else {
					newHash := computeFileHash(policyPath)
					state.SetPolicyHash(newHash)
					fmt.Printf("Policy reloaded from %s (hash: %s)\n", policyPath, newHash)
				}
			}
		}()
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: createHandlerWithState(state),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Manglekit HTTP Sidecar listening on :%s (policy_hash: %s)\n", port, state.GetPolicyHash())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// Wire request/response types

type AssessRequest struct {
	Action         string         `json:"action"`
	Facts          []string       `json:"facts,omitempty"`
	SecurityLabels []string       `json:"security_labels,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Payload        any            `json:"payload,omitempty"`
}

type AssessResponse struct {
	Decision   string         `json:"decision"` // "ALLOW" or "HALT"
	DecisionID string         `json:"decision_id"`
	PolicyHash string         `json:"policy_hash,omitempty"`
	Action     string         `json:"action,omitempty"`
	Payload    any            `json:"payload,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type DenyResponse struct {
	Decision   string   `json:"decision"` // "HALT"
	DecisionID string   `json:"decision_id"`
	PolicyHash string   `json:"policy_hash,omitempty"`
	RuleID     string   `json:"rule_id,omitempty"`
	Tier       string   `json:"tier,omitempty"`
	Reasons    []string `json:"reasons"`
	ProofTree  string   `json:"proof_tree,omitempty"`
}

type ExplainRequest struct {
	Query string   `json:"query"`
	Facts []string `json:"facts"`
}

type ExplainResponse struct {
	Outcome    bool   `json:"outcome"`
	PolicyHash string `json:"policy_hash,omitempty"`
	ProofTree  string `json:"proof_tree"`
}

type HealthResponse struct {
	Status       string `json:"status"`
	PolicyLoaded bool   `json:"policy_loaded"`
	PolicyHash   string `json:"policy_hash,omitempty"`
}

type PolicyResponse struct {
	PolicyPath string `json:"policy_path"`
	PolicyHash string `json:"policy_hash"`
}

// createHandler maintains backward-compatibility for tests.
func createHandler(client *sdk.Client) http.HandlerFunc {
	state := &ServerState{
		Client: client,
	}
	state.SetPolicyHash("")
	h := createHandlerWithState(state)
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}

func createHandlerWithState(state *ServerState) http.Handler {
	mux := http.NewServeMux()

	// GET /v1/health
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{
			Status:       "ok",
			PolicyLoaded: state.GetPolicyHash() != "" || state.PolicyPath != "",
			PolicyHash:   state.GetPolicyHash(),
		})
	})

	// GET /v1/policy
	mux.HandleFunc("/v1/policy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PolicyResponse{
			PolicyPath: state.PolicyPath,
			PolicyHash: state.GetPolicyHash(),
		})
	})

	// GET /v1/openapi.json
	mux.HandleFunc("/v1/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(openAPISpecJSON))
	})

	// POST /v1/explain
	mux.HandleFunc("/v1/explain", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req ExplainRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Query == "" {
			req.Query = `halt("Req", Reason, Tier)`
		}
		expl, err := state.Client.Explain(r.Context(), req.Query, req.Facts)
		if err != nil {
			http.Error(w, fmt.Sprintf("Explain failed: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ExplainResponse{
			Outcome:    expl.Outcome,
			PolicyHash: state.GetPolicyHash(),
			ProofTree:  expl.String(),
		})
	})

	// POST /v1/assess and fallback root POST /
	assessHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var envelope core.Envelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		if envelope.Metadata == nil {
			envelope.Metadata = make(map[string]any)
		}
		if envelope.ID == uuid.Nil {
			envelope.ID = uuid.New()
		}

		result, err := state.Client.Execute(r.Context(), envelope)
		policyHash := state.GetPolicyHash()

		if err != nil {
			if core.IsPolicyViolationError(err) || core.IsAlignmentError(err) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)

				var alignErr *core.AlignmentError
				var pve *core.PolicyViolationError

				ruleID := ""
				tier := "T1"
				reasons := []string{}

				if errors.As(err, &pve) {
					ruleID = pve.MatchedRule
					tier = pve.Tier
					if pve.MatchedRule != "" {
						reasons = append(reasons, pve.MatchedRule)
					}
					if pve.Violation != "" && pve.Violation != pve.MatchedRule {
						reasons = append(reasons, pve.Violation)
					}
				} else if errors.As(err, &alignErr) {
					ruleID = alignErr.RuleID
					reasons = append(reasons, alignErr.Message)
				} else {
					reasons = append(reasons, err.Error())
				}

				// Try to enrich with derivation proof tree
				proofTree := ""
				explFacts := append([]string{}, envelope.Facts...)
				if expl, explErr := state.Client.Explain(r.Context(), `halt("Req", Reason, Tier)`, explFacts); explErr == nil {
					proofTree = expl.String()
				}

				resp := DenyResponse{
					Decision:   core.DecisionHalt,
					DecisionID: envelope.ID.String(),
					PolicyHash: policyHash,
					RuleID:     ruleID,
					Tier:       tier,
					Reasons:    reasons,
					ProofTree:  proofTree,
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			http.Error(w, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
			return
		}

		// Check decision in metadata
		if d, ok := result.Metadata[core.KeyDecision]; ok && d == core.DecisionHalt {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			resp := DenyResponse{
				Decision:   core.DecisionHalt,
				DecisionID: envelope.ID.String(),
				PolicyHash: policyHash,
				Reasons:    []string{"Execution halted by policy"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// For backward compatibility on root path, return envelope if requested
		if r.URL.Path == "/" {
			json.NewEncoder(w).Encode(result)
			return
		}

		resp := AssessResponse{
			Decision:   "ALLOW",
			DecisionID: envelope.ID.String(),
			PolicyHash: policyHash,
			Payload:    result.Payload,
			Metadata:   result.Metadata,
		}
		json.NewEncoder(w).Encode(resp)
	}

	mux.HandleFunc("/v1/assess", assessHandler)
	mux.HandleFunc("/", assessHandler)

	return mux
}

const openAPISpecJSON = `{
  "openapi": "3.1.0",
  "info": {
    "title": "MangleKit Zero-Trust Policy Gate Sidecar",
    "version": "1.0.0",
    "description": "Deterministic Datalog policy enforcement sidecar for AI agent actions."
  },
  "paths": {
    "/v1/health": {
      "get": {
        "summary": "Health check and policy status",
        "responses": {
          "200": { "description": "Server healthy" }
        }
      }
    },
    "/v1/policy": {
      "get": {
        "summary": "Current loaded policy metadata and cryptographic fingerprint",
        "responses": {
          "200": { "description": "Policy metadata" }
        }
      }
    },
    "/v1/assess": {
      "post": {
        "summary": "Evaluate action request against policy gate",
        "responses": {
          "200": { "description": "Action allowed" },
          "403": { "description": "Action blocked by policy with derivation proof" }
        }
      }
    },
    "/v1/explain": {
      "post": {
        "summary": "Query derivation tree for a set of facts",
        "responses": {
          "200": { "description": "Proof tree explanation" }
        }
      }
    }
  }
}`
