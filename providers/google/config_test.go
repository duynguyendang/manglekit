package google

import (
	"testing"
)

func TestConfigFromOptsMapsEveryKey(t *testing.T) {
	cfg, err := ConfigFromOpts(map[string]any{
		"api_key":     "k",
		"model":       "gemini-2.5-flash",
		"platform":    "vertex",
		"project":     "p",
		"location":    "us-east5",
		"api_version": "v1",
		"base_url":    "http://127.0.0.1:9999",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		APIKey: "k", ModelName: "gemini-2.5-flash", Platform: "vertex",
		Project: "p", Location: "us-east5", APIVersion: "v1", BaseURL: "http://127.0.0.1:9999",
	}
	if cfg != want {
		t.Fatalf("opts mapping lost fields:\n got %+v\nwant %+v", cfg, want)
	}
}

func TestConfigFromOptsDefaultsAndErrors(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	if _, err := ConfigFromOpts(map[string]any{"model": "x"}); err == nil {
		t.Fatal("missing api_key must error")
	}
	cfg, err := ConfigFromOpts(map[string]any{"api_key": "k"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ModelName != "gemini-1.5-flash" {
		t.Fatalf("model default changed: %q", cfg.ModelName)
	}
}

// TestVertexModeSelection pins BOTH the historical rule and the new explicit
// ones, so a config written before this change keeps hitting the same backend.
func TestVertexModeSelection(t *testing.T) {
	cases := []struct {
		name          string
		cfg           Config
		vertex        bool
		expressVertex bool
	}{
		{name: "plain api key", cfg: Config{APIKey: "k"}},
		{
			name:   "legacy: project implies vertex",
			cfg:    Config{Project: "p"},
			vertex: true,
		},
		{
			name:          "express: platform vertex, no project",
			cfg:           Config{APIKey: "k", Platform: "vertex"},
			vertex:        true,
			expressVertex: true,
		},
		{
			name:   "explicit platform googleai",
			cfg:    Config{APIKey: "k", Platform: "googleai"},
			vertex: false,
		},
		{
			name:   "explicit platform wins over a stray project",
			cfg:    Config{Platform: "googleai", Project: "p"},
			vertex: false,
		},
		{
			name:          "platform aliases resolve to express vertex",
			cfg:           Config{Platform: "VertexAI"},
			vertex:        true,
			expressVertex: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.isVertex(); got != tc.vertex {
				t.Errorf("isVertex() = %v, want %v", got, tc.vertex)
			}
			if got := tc.cfg.expressVertex(); got != tc.expressVertex {
				t.Errorf("expressVertex() = %v, want %v", got, tc.expressVertex)
			}
		})
	}
}

// TestExpressVertexNeedsAKey: choosing vertex with no project AND no key must
// fail loudly rather than silently falling back to ADC.
func TestExpressVertexNeedsAKey(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_API_KEY", "")
	_, err := InitWithConfig(t.Context(), nil, Config{ModelName: "m", Platform: "vertex"}, nil)
	if err == nil {
		t.Fatal("expected an error for keyless express mode")
	}
}
