package scoring_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit"
	function "github.com/duynguyendang/manglekit/adapters/func"
	"github.com/duynguyendang/manglekit/core"
)

// The comparison-in-policy spike.
//
// Four places in this workspace claim Mangle's analyzer rejects cross-fact
// :lt/:gt, which is why every numeric gate so far pre-computes booleans in Go
// and lets the policy pattern-match them. The engine's own test
// (internal/engine/manglekit_deps_test.go:27-28,59-60) contradicts that claim
// for integers. This file settles it for the shape that matters here:
//
//   - the observation arrives as an EXPLICIT FACT AT GATE TIME (envelope
//     Facts), not from the loaded program,
//   - the threshold is RULE-defined (so ReloadPolicy can swap it),
//   - and the two are joined through a VARIABLE (choice -> its own score ->
//     that class's bar) before :lt compares them.
//
// If this passes, the policy can own the decision arithmetic and Go only
// reports measurements.

const perClassPolicy = `
Decl policy_anchor(Marker).
policy_anchor("ok").

Decl class_bar(Class, Bar).
class_bar("billing", 800) :- policy_anchor("ok").
class_bar("technical_support", 800) :- policy_anchor("ok").
class_bar("account_access", 950) :- policy_anchor("ok").

Decl choice(Option).
Decl score_pm(Option, Score).

halt("Req", "low_confidence_send_to_review", "T1") :-
    action_operation("Req", "route_ticket"),
    choice(C), score_pm(C, P), class_bar(C, T), :lt(P, T).
`

type spikeInput struct {
	Ticket string
}

func spikeClient(t *testing.T) (*manglekit.Client, *int) {
	t.Helper()
	ctx := context.Background()
	client, err := manglekit.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Shutdown(ctx) })
	if err := client.LoadPolicy(ctx, perClassPolicy); err != nil {
		t.Fatalf("load policy: %v", err)
	}
	dispatched := 0
	client.RegisterSupervised("route_ticket", function.New("route_ticket",
		func(context.Context, spikeInput) (string, error) {
			dispatched++
			return "ok", nil
		}))
	return client, &dispatched
}

func spikeRun(ctx context.Context, t *testing.T, client *manglekit.Client, choice string, pm int) error {
	t.Helper()
	env := core.NewEnvelope(spikeInput{Ticket: "T-1"})
	env.Facts = append(env.Facts,
		`choice("`+choice+`").`,
		`score_pm("`+choice+`", `+itoa(pm)+`).`,
	)
	_, err := client.ExecuteByName(ctx, "route_ticket", env)
	return err
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// TestPolicyCanCompareAcrossFacts is the spike proper.
func TestPolicyCanCompareAcrossFacts(t *testing.T) {
	ctx := context.Background()
	client, dispatched := spikeClient(t)

	cases := []struct {
		name     string
		choice   string
		pm       int
		wantDeny bool
		wantRule string
	}{
		// account_access bar is 950: 900 fails, 950 passes (boundary is >=).
		{"risky class below its own bar", "account_access", 900, true, "low_confidence_send_to_review"},
		{"risky class exactly at bar", "account_access", 950, false, ""},
		{"risky class above bar", "account_access", 970, false, ""},
		// Same number, different class, different verdict — the whole point of
		// per-class bars: a scalar confidence could not express this.
		{"same score, ordinary class passes", "billing", 900, false, ""},
		{"ordinary class below its bar", "billing", 740, true, "low_confidence_send_to_review"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := *dispatched
			err := spikeRun(ctx, t, client, tc.choice, tc.pm)

			if !tc.wantDeny {
				if err != nil {
					t.Fatalf("expected allow, got deny: %v", err)
				}
				if *dispatched != before+1 {
					t.Fatalf("allowed call must dispatch (before=%d after=%d)", before, *dispatched)
				}
				return
			}

			var pve *core.PolicyViolationError
			if !core.IsPolicyViolationError(err) || !errors.As(err, &pve) {
				t.Fatalf("expected a policy deny, got %v", err)
			}
			if *dispatched != before {
				t.Fatalf("halt must mean no dispatch (before=%d after=%d)", before, *dispatched)
			}
			if !strings.Contains(pve.MatchedRule+pve.Violation+pve.RuleID, tc.wantRule) {
				t.Errorf("deny must name rule %q, got %+v", tc.wantRule, pve)
			}
		})
	}
}

// TestExplainShowsTheArithmetic: the proof tree must contain the comparison
// itself, not just a Go-asserted boolean. This is the audit win.
func TestExplainShowsTheArithmetic(t *testing.T) {
	ctx := context.Background()
	client, _ := spikeClient(t)

	expl, err := client.Explain(ctx, `halt("Req", Reason, Tier)`, []string{
		`action_operation("Req", "route_ticket").`,
		`choice("account_access").`,
		`score_pm("account_access", 900).`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !expl.Outcome {
		t.Fatal("halt must derive")
	}
	tree := expl.String()
	for _, want := range []string{`score_pm("account_access",900)`, `class_bar("account_access",950)`, "low_confidence_send_to_review"} {
		if !strings.Contains(tree, want) {
			t.Fatalf("proof tree must ground %q:\n%s", want, tree)
		}
	}
	t.Logf("proof tree:\n%s", tree)
}

// TestPerClassBarsAreHotSwappable: because the bars are rules, ReloadPolicy can
// change them mid-process and the same facts re-decide.
func TestPerClassBarsAreHotSwappable(t *testing.T) {
	ctx := context.Background()
	client, _ := spikeClient(t)

	relaxed := strings.Replace(perClassPolicy, `class_bar("account_access", 950)`, `class_bar("account_access", 850)`, 1)
	if err := client.ReloadPolicySource(ctx, relaxed); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if err := spikeRun(ctx, t, client, "account_access", 900); err != nil {
		t.Fatalf("900 must clear the relaxed 850 bar, got %v", err)
	}
}
