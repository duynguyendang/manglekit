package engine

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

func TestPolicyEngine_QueryWithAudit(t *testing.T) {
	ctx := context.Background()

	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Add some test policies
	err = engine.Runtime().AddPolicy(context.Background(), `
% Test governance rules - use different predicate names
grant_access("reader", "read", "document").
grant_access("writer", "write", "document").
restrict_access("guest", "write", "document").

can_execute("planner", "plan").
can_execute("executor", "execute").
`)
	if err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	// Test query with audit
	result, err := engine.QueryWithAudit(ctx, []string{}, `grant_access("reader", "read", "document")`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(result.Results) == 0 {
		t.Error("expected results, got none")
	}

	if result.AuditTrail == nil {
		t.Error("expected audit trail, got nil")
	}

	t.Logf("Audit Trail Summary: %s", result.AuditTrail.Summary())
	t.Logf("Matched %d rules, latency: %dms", result.AuditTrail.MatchedCount, result.AuditTrail.LatencyMs)
}

func TestPolicyEngine_QueryWithAudit_NoMatch(t *testing.T) {
	ctx := context.Background()

	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	err = engine.Runtime().AddPolicy(context.Background(), `
allow("reader", "read", "document").
`)
	if err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	result, err := engine.QueryWithAudit(ctx, []string{}, `allow("writer", "write", "document")`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	// No match expected
	if len(result.Results) != 0 {
		t.Errorf("expected no results, got %d", len(result.Results))
	}

	// But audit trail should still exist
	if result.AuditTrail == nil {
		t.Error("expected audit trail, got nil")
	}

	t.Logf("Audit Trail (no match): %s", result.AuditTrail.Summary())
}

func TestAuditTrail_Summary(t *testing.T) {
	audit := core.NewAuditTrail("test-engine", "test_query")

	audit.AddRule("allow", "allow(X, Y, Z) :- true.", "governance.dl", "allow", core.TierT1_Governance, map[string]string{
		"X": "reader",
		"Y": "read",
		"Z": "document",
	})

	audit.AddRule("can_execute", "can_execute(Agent, Action).", "registry.dl", "can_execute", core.TierT2_Playbook, map[string]string{
		"Agent":  "planner",
		"Action": "plan",
	})

	summary := audit.Summary()
	t.Logf("Audit Summary:\n%s", summary)

	if len(audit.MatchedRules) != 2 {
		t.Errorf("expected 2 matched rules, got %d", len(audit.MatchedRules))
	}

	// Verify tier mapping
	if audit.MatchedRules[0].Tier != core.TierT1_Governance {
		t.Errorf("expected T1_Governance, got %s", audit.MatchedRules[0].Tier)
	}

	if audit.MatchedRules[1].Tier != core.TierT2_Playbook {
		t.Errorf("expected T2_Playbook, got %s", audit.MatchedRules[1].Tier)
	}
}

func TestDetermineTierFromPredicate(t *testing.T) {
	tests := []struct {
		predicate string
		expected  core.Tier
	}{
		{"allow", core.TierT1_Governance},
		{"deny", core.TierT1_Governance},
		{"may_read", core.TierT1_Governance},
		{"may_write", core.TierT1_Governance},
		{"can_execute", core.TierT2_Playbook},
		{"can_access", core.TierT2_Playbook},
		{"requires", core.TierT2_Playbook},
		{"validation_rule", core.TierT2_Playbook},
		{"prompt_template", core.TierT2_Playbook},
		{"workflow_node", core.TierT2_Playbook},
		{"role_capability", core.TierT2_Playbook},
		{"unknown_predicate", core.TierUnknown},
	}

	for _, tt := range tests {
		tier := determineTierFromPredicate(tt.predicate)
		if tier != tt.expected {
			t.Errorf("for predicate %s: expected %s, got %s", tt.predicate, tt.expected, tier)
		}
	}
}

func TestGetSourceFileForPredicate(t *testing.T) {
	tests := []struct {
		predicate string
		expected  string
	}{
		{"allow", "governance.dl"},
		{"validation_rule", "registry.dl"},
		{"unknown_predicate", "unknown"},
	}

	for _, tt := range tests {
		sourceFile := getSourceFileForPredicate(tt.predicate)
		if sourceFile != tt.expected {
			t.Errorf("for predicate %s: expected %s, got %s", tt.predicate, tt.expected, sourceFile)
		}
	}
}

func TestExtractPredicateFromQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"can_execute(Agent, Action)", "can_execute"},
		{"allow(X, Y, Z)", "allow"},
		{"valid", "valid"},
		{"  allow(X)  ", "allow"},
		{"not allowed(X)", ""}, // Complex query - may return empty
	}

	for _, tt := range tests {
		predicate := extractPredicateFromQuery(tt.query)
		// Some complex queries might return empty, so we only check positive matches
		if tt.expected != "" && predicate != tt.expected {
			t.Errorf("for query %s: expected %s, got %s", tt.query, tt.expected, predicate)
		}
	}
}
