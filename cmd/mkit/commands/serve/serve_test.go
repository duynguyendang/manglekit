package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/adapters/mcp"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// DummyAction is a mock action for testing
type DummyAction struct{}

func (d *DummyAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	input.SetMeta(core.KeyDecision, core.DecisionProceed)
	// Change state to stop routing loop
	input.SetMeta("state", "done")
	return input, nil
}

func (d *DummyAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "dummy_action",
		Type: "test",
	}
}

func TestServe_Success(t *testing.T) {
	ctx := context.Background()
	// Create client with dummy action and steering enabled so
	// EvaluateSteering can route requests via Datalog.
	client, err := sdk.NewClient(ctx, sdk.WithSteeringEnabled())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Register dummy action, MUST be supervised for governance
	client.RegisterAction("dummy_action", client.Supervise(&DummyAction{}))

	// Policy: Route ONLY if state is 'init'
	// Note: route and meta are already declared in StdLib
	policy := `
	route("dummy_action") :- meta("state", "init").
	`

	err = client.Engine().LoadPolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	handler := createHandler(client)
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create request with initial state
	payload := map[string]string{"foo": "bar"}
	envelope := core.NewEnvelope(payload)
	envelope.SetMeta("state", "init")

	body, _ := json.Marshal(envelope)

	resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestServe_V1Endpoints(t *testing.T) {
	ctx := context.Background()
	client, err := sdk.NewClient(ctx, sdk.WithSteeringEnabled())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.RegisterAction("dummy_action", client.Supervise(&DummyAction{}))

	policy := `
	route("dummy_action") :- meta("state", "init").
	halt("Req", "blocked_in_test", "T1") :- meta("block", "true").
	`
	if err := client.Engine().LoadPolicy(ctx, policy); err != nil {
		t.Fatalf("load policy: %v", err)
	}

	state := &ServerState{
		Client:     client,
		PolicyPath: "test.mg",
	}
	state.SetPolicyHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

	handler := createHandlerWithState(state)
	server := httptest.NewServer(handler)
	defer server.Close()

	// 1. GET /v1/health
	t.Run("health", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/v1/health")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", resp.StatusCode)
		}
		var h HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
			t.Fatal(err)
		}
		if h.Status != "ok" || !h.PolicyLoaded {
			t.Fatalf("unexpected health resp: %+v", h)
		}
	})

	// 2. GET /v1/policy
	t.Run("policy", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/v1/policy")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", resp.StatusCode)
		}
		var p PolicyResponse
		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			t.Fatal(err)
		}
		if p.PolicyHash != state.GetPolicyHash() {
			t.Fatalf("hash mismatch: want %s got %s", state.GetPolicyHash(), p.PolicyHash)
		}
	})

	// 3. GET /v1/openapi.json
	t.Run("openapi", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/v1/openapi.json")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", resp.StatusCode)
		}
	})

	// 4. POST /v1/assess - allowed
	t.Run("assess_allow", func(t *testing.T) {
		env := core.NewEnvelope(map[string]string{"foo": "bar"})
		env.SetMeta("state", "init")
		body, _ := json.Marshal(env)

		resp, err := http.Post(server.URL+"/v1/assess", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", resp.StatusCode)
		}
		var a AssessResponse
		if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
			t.Fatal(err)
		}
		if a.Decision != "ALLOW" {
			t.Fatalf("want ALLOW, got %s", a.Decision)
		}
	})

	// 5. POST /v1/assess - deny
	t.Run("assess_deny", func(t *testing.T) {
		env := core.NewEnvelope(map[string]string{"foo": "bar"})
		env.SetMeta("state", "init")
		env.SetMeta("block", "true")
		body, _ := json.Marshal(env)

		resp, err := http.Post(server.URL+"/v1/assess", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("want 403, got %d", resp.StatusCode)
		}
		var d DenyResponse
		if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
			t.Fatal(err)
		}
		if d.Decision != core.DecisionHalt {
			t.Fatalf("want HALT, got %s", d.Decision)
		}
		if d.PolicyHash == "" {
			t.Fatal("expected non-empty policy hash in deny response")
		}
	})

	// 6. POST /v1/explain
	t.Run("explain", func(t *testing.T) {
		req := ExplainRequest{
			Query: `route("dummy_action")`,
			Facts: []string{`meta("state", "init").`},
		}
		body, _ := json.Marshal(req)
		resp, err := http.Post(server.URL+"/v1/explain", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", resp.StatusCode)
		}
		var expl ExplainResponse
		if err := json.NewDecoder(resp.Body).Decode(&expl); err != nil {
			t.Fatal(err)
		}
		if !expl.Outcome {
			t.Fatalf("expected explain outcome true, got false")
		}
		if expl.ProofTree == "" {
			t.Fatal("expected non-empty proof tree")
		}
	})
}

func TestServe_PolicyViolation(t *testing.T) {
	ctx := context.Background()

	client, err := sdk.NewClient(ctx, sdk.WithSteeringEnabled())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.RegisterAction("dummy_action", client.Supervise(&DummyAction{}))

	fullPolicy := `
	Decl sensitive_data(Flag).

	route("dummy_action") :- meta("state", "init").
	halt("Req", "Blocked by Test Policy", "T1") :- sensitive_data("true").
	`

	err = client.Engine().LoadPolicy(ctx, fullPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	handler := createHandler(client)
	server := httptest.NewServer(handler)
	defer server.Close()

	rawJSON := `{"data": {"foo": "bar"}, "metadata": {"state": "init"}, "facts": ["sensitive_data(\"true\")."]}`

	resp, err := http.Post(server.URL, "application/json", bytes.NewBufferString(rawJSON))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	reasons, ok := result["reasons"].([]any)
	if !ok {
		t.Fatalf("Response missing 'reasons' field: %v", result)
	}

	found := false
	for _, r := range reasons {
		if rStr, ok := r.(string); ok && strings.Contains(rStr, "Blocked by Test Policy") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected reason 'Blocked by Test Policy', got %v", reasons)
	}
}

func TestLoadMCPActions_MissingFile(t *testing.T) {
	if _, err := loadMCPActions(context.Background(), filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadMCPActions_BadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(path, []byte(`{"not": "an array"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadMCPActions(context.Background(), path); err == nil {
		t.Fatal("expected parse error for non-array config")
	}
}

// A server whose client fails, with FailOnStartup=false: the loader returns
// unhealthy stub actions (one per expected tool) instead of failing.
type failFactory struct{}

func (failFactory) CreateClient(context.Context, config.MCPServerConfig) (mcp.Client, error) {
	return failClient{}, nil
}

type failClient struct{}

func (failClient) GetActiveTools(context.Context, *genkit.Genkit) ([]ai.Tool, error) {
	return nil, errors.New("connection refused")
}

func TestLoadMCPActions_UnhealthyStubs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	cfg := `[{"name":"todo","transport":"stdio","command":"echo","tools":["create","list"]}]`
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	actions, err := loadMCPActionsWithFactory(context.Background(), path, failFactory{})
	if err != nil {
		t.Fatalf("expected degraded start with unhealthy stubs, got %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("want 2 unhealthy actions, got %d", len(actions))
	}
}

func TestLoadMCPActions_FailOnStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	cfg := `[{"name":"todo","transport":"stdio","command":"echo","fail_on_startup":true}]`
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadMCPActionsWithFactory(context.Background(), path, failFactory{}); err == nil {
		t.Fatal("expected error when FailOnStartup is set")
	}
}

// JSON keys in the config file map via the config struct's json tags.
func TestLoadMCPActions_ParsesLowercaseKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	cfg := `[{"name":"s1","transport":"unknown-transport","command":"x","tools":["t"]}]`
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	// An unknown transport yields an unhealthy stub (no real client) rather
	// than a hard failure: proves the entry parsed and reached the loader.
	actions, err := loadMCPActionsWithFactory(context.Background(), path, failFactory{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("want 1 action, got %d", len(actions))
	}
}
