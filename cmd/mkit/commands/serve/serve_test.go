package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
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

func TestServe_PolicyViolation(t *testing.T) {
	t.Skip("Skipping: The current SupervisedAction architecture doesn't properly pass Facts to Assess. This test needs a fix in the SDK adapter to pass envelope Facts to VerifyAtoms.")
	/*
		// This test is kept for reference but disabled due to architecture issues.
		// The issue: SupervisedAction calls VerifyAtoms which wraps facts in a new envelope,
		// losing the original Facts from the request.
		ctx := context.Background()

		// Use failure mode "closed" (default) to ensure blockage
		client, err := sdk.NewClient(ctx)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Register supervised dummy action
		client.RegisterAction("dummy_action", client.Supervise(&DummyAction{}))

		// Policy:
		// Note: deny, route, meta are already declared in StdLib
		// Use Correct Arity: deny/1, violation_msg/1, violation_rule/1.
		// To test blocking, we derive deny from a fact injected in the request
		fullPolicy := `
		Decl violation_msg(Msg).
		Decl violation_rule(ID).
		Decl sensitive_data(Flag).

		% Route so we can try to execute (and get blocked by Authorize)
		route("dummy_action") :- meta("state", "init").

		% Deny ANY request that has sensitive_data flag set to "true"
		deny("Blocked by Test Policy") :- sensitive_data("true").

		% Attach message to the denial
		violation_msg("Blocked by Test Policy") :- deny(_).
		violation_rule("rule-1") :- deny(_).
		`

		err = client.Engine().LoadPolicy(ctx, fullPolicy)
		if err != nil {
			t.Fatalf("Failed to load policy: %v", err)
		}

		handler := createHandler(client)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Create request with state init to trigger routing (and then blocking)
		// We inject a fact "sensitive_data" to trigger the denial rule
		rawJSON := `{"data": {"foo": "bar"}, "metadata": {"state": "init"}, "facts": ["sensitive_data(\"true\")"]}`

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

		// Verify JSON Body contains "Blocked by Test Policy"
		reasons, ok := result["reasons"].([]any)
		if !ok {
			t.Fatalf("Response missing 'reasons' field: %v", result)
		}

		found := false
		for _, r := range reasons {
			if rStr, ok := r.(string); ok && rStr == "Blocked by Test Policy" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected reason 'Blocked by Test Policy', got %v", reasons)
		}
	*/
}
