package logic

import (
	"math"
	"testing"
)

func TestCalculateEAST_ClampsInputs(t *testing.T) {
	// Negative logicSuccess clamps to 0: activation = e^1.
	got := CalculateEAST(-1.0, 1.0)
	want := math.Exp(1.0) / 1.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("CalculateEAST(-1,1) = %v, want %v", got, want)
	}

	// LogicSuccess > 1 clamps to 1: activation = e^0 = 1.
	got = CalculateEAST(5.0, 1.0)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("CalculateEAST(5,1) = %v, want 1", got)
	}
}

func TestCalculateEAST_DivisionByZeroGuard(t *testing.T) {
	// Coefficient <= 0.01 is floored to 0.01, never dividing by zero / inf.
	got := CalculateEAST(0.5, 0.0)
	if math.IsInf(got, 0) || math.IsNaN(got) {
		t.Errorf("CalculateEAST with zero coefficient returned %v (should be finite)", got)
	}
	// With coefficient 0.01 and logicSuccess 0.5: activation = e^0.5 / 0.01.
	want := math.Exp(0.5) / 0.01
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("CalculateEAST(0.5,0) = %v, want %v", got, want)
	}
}

func TestCalculateEAST_MonotonicInLogicSuccess(t *testing.T) {
	// Higher logic success => lower magnitude.
	low := CalculateEAST(0.9, 1.0)
	high := CalculateEAST(0.1, 1.0)
	if low >= high {
		t.Errorf("expected magnitude to decrease with logic success: low=%v high=%v", low, high)
	}
}

func TestGetSteeringPrompts_TierSelection(t *testing.T) {
	cases := []struct {
		magnitude float64
		header    string
		body      string
	}{
		{0.9, "prompt_header_paradox", "prompt_body_conservative"},
		{0.8, "prompt_header_paradox", "prompt_body_conservative"},
		{0.6, "prompt_header_strict", "prompt_body_structured"},
		{0.5, "prompt_header_strict", "prompt_body_structured"},
		{0.1, "prompt_header_default", "prompt_body_creative"},
		{0.0, "prompt_header_default", "prompt_body_creative"},
	}
	for _, c := range cases {
		h, b := GetSteeringPrompts(c.magnitude)
		if h != c.header || b != c.body {
			t.Errorf("GetSteeringPrompts(%v) = (%q,%q), want (%q,%q)",
				c.magnitude, h, b, c.header, c.body)
		}
	}
}

func TestDefaultParadoxThreshold(t *testing.T) {
	if DefaultParadoxThreshold != 0.8 {
		t.Errorf("DefaultParadoxThreshold = %v, want 0.8", DefaultParadoxThreshold)
	}
}
