package agents

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/agents/tools"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/multiagent"
)

func TestSessionStoreAdapterRoundTrip(t *testing.T) {
	ctx := context.Background()
	adapter := newSessionStoreAdapter(tools.NewTransientStore())
	sessionID := "sess-1"
	workflowID := "wf"

	inst := core.NewWorkflowInstance(workflowID, sessionID)
	inst.Variables["marker"] = "persisted"

	if err := adapter.Create(ctx, inst); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !adapter.Exists(ctx, inst.SessionKey()) {
		t.Fatal("expected instance to exist after Create")
	}

	// Get must return the persisted instance, not a freshly constructed one.
	got, err := adapter.Get(ctx, inst.SessionKey())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil || got.Variables["marker"] != "persisted" {
		t.Fatalf("Get returned a non-persisted instance: %+v", got)
	}

	// Regression guard: List by plain sessionID (write partition) still works.
	list, err := adapter.List(ctx, sessionID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 instance in session, got %d", len(list))
	}

	// Update must be reflected on read.
	inst.Variables["marker"] = "updated"
	if err := adapter.Update(ctx, inst); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	got, err = adapter.Get(ctx, inst.SessionKey())
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if got.Variables["marker"] != "updated" {
		t.Fatalf("Update not reflected: got %v", got.Variables["marker"])
	}

	// Delete must remove the instance so Exists reports false.
	if err := adapter.Delete(ctx, inst.SessionKey()); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if adapter.Exists(ctx, inst.SessionKey()) {
		t.Fatal("expected instance to be gone after Delete")
	}

	if err := adapter.ClearSession(ctx, sessionID); err != nil {
		t.Fatalf("ClearSession failed: %v", err)
	}
}

func TestArchitectAgentLifecycle(t *testing.T) {
	ctx := context.Background()
	sys, err := multiagent.NewAgentSystem(ctx)
	if err != nil {
		t.Fatalf("failed to create agent system: %v", err)
	}
	if err := sys.LoadAgentDefinitions(ctx); err != nil {
		t.Fatalf("failed to load agent definitions: %v", err)
	}

	agent, err := NewArchitectAgent(ctx, sys, "architect-workflow")
	if err != nil {
		t.Fatalf("failed to create architect agent: %v", err)
	}

	if summary := agent.GetAuditTrail(); summary == "" {
		t.Error("expected a non-empty audit trail summary")
	}

	if err := agent.ClearSession(ctx); err != nil {
		t.Fatalf("ClearSession failed: %v", err)
	}
}
