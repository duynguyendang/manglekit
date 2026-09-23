package scoring_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit"
	function "github.com/duynguyendang/manglekit/adapters/func"
	"github.com/duynguyendang/manglekit/adapters/scoring"
	"github.com/duynguyendang/manglekit/core"
)

const (
	policyStrict  = "testdata/routing_policy.dl"
	policyRelaxed = "testdata/routing_policy_relaxed.dl"
	policyBroken  = "testdata/routing_policy_broken.dl"
)

type routeReq struct {
	Ticket string
	Choice string
}

type dispatcher struct {
	t            *testing.T
	calls        int
	client       *manglekit.Client
	executed     int
	dispatchedTo []string
}

func (d *dispatcher) run(_ context.Context, in routeReq) (string, error) {
	d.calls++
	d.executed++
	d.dispatchedTo = append(d.dispatchedTo, in.Choice)
	return "dispatched:" + in.Choice, nil
}

// stubScorer is a deterministic scoring.Scorer standing in for a model: it
// replays one probability vector per call.
type stubScorer struct {
	probs []float64
	err   error
}

func (s *stubScorer) Score(_ context.Context, prompt string, labels, options []string) (*scoring.Decision, error) {
	if s.err != nil {
		return nil, s.err
	}
	return scoring.Build(labels, options, s.probs)
}

func newGate(t *testing.T, policy string) (*dispatcher, func()) {
	t.Helper()
	ctx := context.Background()
	client, err := manglekit.QuickClient(ctx, policy)
	if err != nil {
		t.Fatalf("QuickClient(%s): %v", policy, err)
	}
	d := &dispatcher{t: t, client: client}
	client.RegisterSupervised("route_ticket", function.New("route_ticket", d.run))
	return d, func() { _ = client.Shutdown(ctx) }
}

func route(ctx context.Context, d *dispatcher, th scoring.Thresholds, dec *scoring.Decision) error {
	_, err := d.client.ExecuteByName(ctx, "route_ticket", scoring.Request(routeReq{Ticket: "T-1"}, dec, th))
	return err
}

// TestScoringEndToEndNoDXChange is the claim this adapter exists for: a
// fixed-choice model read becomes a governed decision using the SAME client
// surface as every other supervised action — QuickClient, RegisterSupervised,
// ExecuteByName, IsPolicyViolationError. Nothing in this flow is scoring-
// specific except how the envelope is built.
func TestScoringEndToEndNoDXChange(t *testing.T) {
	ctx := context.Background()
	d, cleanup := newGate(t, policyStrict)
	defer cleanup()

	th, err := scoring.ThresholdsFromPolicy(ctx, d.client.Engine(), "threshold")
	if err != nil {
		t.Fatal(err)
	}
	if th.Auto != 0.80 || th.Margin != 0.20 || th.Review != 0.70 {
		t.Fatalf("thresholds must come from the policy: %+v", th)
	}

	cases := []struct {
		name      string
		probs     []float64
		wantAllow bool
		wantRule  string
	}{
		{"clears the bar", []float64{0.90, 0.07, 0.03}, true, ""},
		{"review band", []float64{0.74, 0.18, 0.08}, false, "low_confidence_send_to_review"},
		{"below floor", []float64{0.44, 0.41, 0.15}, false, "confidence_below_review_floor_escalate"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scorer := &stubScorer{probs: tc.probs}
			dec, err := scorer.Score(ctx, "Route this ticket", []string{"A", "B", "C"},
				[]string{"billing", "tech", "account"})
			if err != nil {
				t.Fatal(err)
			}
			before := d.executed
			err = route(ctx, d, th, dec)

			if tc.wantAllow {
				if err != nil {
					t.Fatalf("expected allow, got %v", err)
				}
				if d.executed != before+1 {
					t.Fatalf("allowed call must dispatch (before=%d after=%d)", before, d.executed)
				}
				return
			}

			var pve *core.PolicyViolationError
			if !core.IsPolicyViolationError(err) || !errors.As(err, &pve) {
				t.Fatalf("expected a policy deny, got %v", err)
			}
			// The Zero-Trust contract: a halt means the dispatch never happened.
			if d.executed != before {
				t.Fatalf("T1 halt must prevent dispatch (before=%d after=%d)", before, d.executed)
			}
			if !strings.Contains(pve.MatchedRule+pve.Violation+pve.RuleID, tc.wantRule) {
				t.Errorf("deny must name rule %q, got %+v", tc.wantRule, pve)
			}
		})
	}
}

// TestUnobservableDistributionIsGoverned: the refusal to auto-route on an
// unknowable number is a policy decision with its own proof, not a Go else.
func TestUnobservableDistributionIsGoverned(t *testing.T) {
	ctx := context.Background()
	d, cleanup := newGate(t, policyStrict)
	defer cleanup()
	th, err := scoring.ThresholdsFromPolicy(ctx, d.client.Engine(), "threshold")
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.client.ExecuteByName(ctx, "route_ticket",
		scoring.Request(routeReq{Ticket: "T-2"}, nil, th))
	if !core.IsPolicyViolationError(err) {
		t.Fatalf("unobserved distribution must be denied, got %v", err)
	}
	if !strings.Contains(err.Error(), "unobserved_distribution_escalate") {
		t.Errorf("wrong rule fired: %v", err)
	}
	if d.executed != 0 {
		t.Fatalf("nothing may dispatch, executed=%d", d.executed)
	}

	// A real scorer reporting truncation reaches the same path.
	scorer := &stubScorer{err: scoring.ErrLabelsUnobserved}
	if _, serr := scorer.Score(ctx, "p", []string{"A"}, []string{"a"}); !errors.Is(serr, scoring.ErrLabelsUnobserved) {
		t.Fatalf("stub wiring: %v", serr)
	}
}

// TestReloadChangesTheDecisionNotTheCode is the adapter's reason to read
// thresholds from the policy: the same Decision re-decides after a hot swap,
// and a broken program cannot take the running one down.
func TestReloadChangesTheDecisionNotTheCode(t *testing.T) {
	ctx := context.Background()
	d, cleanup := newGate(t, policyStrict)
	defer cleanup()

	dec, err := (&stubScorer{probs: []float64{0.71, 0.2, 0.09}}).Score(ctx, "p",
		[]string{"A", "B", "C"}, []string{"billing", "tech", "account"})
	if err != nil {
		t.Fatal(err)
	}

	th, err := scoring.ThresholdsFromPolicy(ctx, d.client.Engine(), "threshold")
	if err != nil {
		t.Fatal(err)
	}
	if err := route(ctx, d, th, dec); !core.IsPolicyViolationError(err) {
		t.Fatalf("P=0.71 under auto=0.80 must be denied, got %v", err)
	}

	if err := d.client.ReloadPolicy(ctx, policyRelaxed); err != nil {
		t.Fatalf("reload relaxed: %v", err)
	}
	th2, err := scoring.ThresholdsFromPolicy(ctx, d.client.Engine(), "threshold")
	if err != nil {
		t.Fatalf("re-read thresholds after reload: %v", err)
	}
	if th2.Auto != 0.70 || th2.Review != 0.60 {
		t.Fatalf("reload must move the bars, got %+v", th2)
	}
	if err := route(ctx, d, th2, dec); err != nil {
		t.Fatalf("same P under relaxed policy must be allowed, got %v", err)
	}
	if d.executed != 1 {
		t.Fatalf("exactly one dispatch expected, got %d", d.executed)
	}

	// Fail-safe: a broken program is rejected and the ACTIVE one keeps serving.
	if err := d.client.ReloadPolicy(ctx, policyBroken); err == nil {
		t.Fatal("broken policy must be rejected")
	}
	th3, err := scoring.ThresholdsFromPolicy(ctx, d.client.Engine(), "threshold")
	if err != nil {
		t.Fatal(err)
	}
	if th3.Auto != 0.70 {
		t.Fatalf("active policy must be unchanged after a failed reload, got %+v", th3)
	}
}

// TestAmbiguousThresholdIsRejected protects the one way a reload can go
// silently wrong: a bare fact survives the swap, so two definitions of the same
// bar become queryable. Returning the first would be a coin flip on safety.
func TestAmbiguousThresholdIsRejected(t *testing.T) {
	ctx := context.Background()
	d, cleanup := newGate(t, policyStrict)
	defer cleanup()

	if err := d.client.LoadPolicy(ctx, `threshold("auto", "0.50").`); err != nil {
		t.Fatalf("load conflicting bare fact: %v", err)
	}
	_, err := scoring.ThresholdsFromPolicy(ctx, d.client.Engine(), "threshold")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("two values for one threshold must error, got %v", err)
	}
}

// TestExplainGroundsTheNumbers proves the audit story survives the move into
// the kernel: the proof tree binds the model's probability against the
// policy's own bar.
func TestExplainGroundsTheNumbers(t *testing.T) {
	ctx := context.Background()
	d, cleanup := newGate(t, policyStrict)
	defer cleanup()

	expl, err := d.client.Explain(ctx, `halt("Req", Reason, Tier)`, []string{
		`action_operation("Req", "route_ticket").`,
		`meta("below_auto", "true").`,
		`confidence("0.740").`,
		`margin_value("0.560").`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !expl.Outcome {
		t.Fatal("halt must derive")
	}
	tree := expl.String()
	for _, want := range []string{"low_confidence_send_to_review", "0.740", "0.80"} {
		if !strings.Contains(tree, want) {
			t.Errorf("proof tree must ground %q:\n%s", want, tree)
		}
	}
}
