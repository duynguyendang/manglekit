//go:build testhooks

package registry_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/registry"
	"github.com/duynguyendang/manglekit/internal/testproviders/noop"
)

func TestRegistry_Smoke(t *testing.T) {
	t.Run("register_and_lookup", func(t *testing.T) {
		t.Cleanup(registry.ResetForTest)
		reg := registry.Global()

		err := manglekit.Register(reg, noop.NoopOptions{}, noop.New)
		if err != nil {
			t.Fatalf("failed to register noop provider: %v", err)
		}

		factory, err := reg.Get(core.KindSchemaParser, "noop")
		if err != nil {
			t.Fatalf("resolve noop factory: %v", err)
		}

		instance, err := factory.Build(context.Background(), nil, noop.NoopOptions{})
		if err != nil {
			t.Fatalf("build noop: %v", err)
		}

		tool, ok := instance.(core.Tool)
		if !ok {
			t.Fatalf("resolved instance is not a core.Tool")
		}

		execCtx := &core.ExecutionContext{Meta: make(map[string]any)}
		err = tool.Execute(context.Background(), execCtx)
		if err != nil {
			t.Fatalf("noop execute: %v", err)
		}

		if executed, _ := execCtx.Meta["noop_executed"].(bool); !executed {
			t.Fatalf("expected noop tool to mark context as executed")
		}
	})

	t.Run("duplicate_registration_fails", func(t *testing.T) {
		t.Cleanup(registry.ResetForTest)
		reg := registry.Global()

		if err := manglekit.Register(reg, noop.NoopOptions{}, noop.New); err != nil {
			t.Fatalf("first registration failed: %v", err)
		}
		err := manglekit.Register(reg, noop.NoopOptions{}, noop.New)
		if err == nil {
			t.Fatalf("expected error on duplicate registration, got nil")
		}
	})

	t.Run("reset_clears_registry", func(t *testing.T) {
		regBeforeReset := registry.Global()
		if err := manglekit.Register(regBeforeReset, noop.NoopOptions{}, noop.New); err != nil {
			t.Fatalf("registration failed: %v", err)
		}

		registry.ResetForTest()

		regAfterReset := registry.Global()
		_, err := regAfterReset.Get(core.KindSchemaParser, "noop")
		if err == nil {
			t.Fatalf("registry still contains 'noop' after reset")
		}
	})
}
