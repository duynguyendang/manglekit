package engine_test

import (
	"context"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type User struct {
	Age int
}

func TestDualModeInput(t *testing.T) {
	// Scenario A: Typed Mode (Struct)
	t.Run("Typed Mode", func(t *testing.T) {
		e := engine.New()

		// Create temporary policy file with dummy fact for declaration
		policy := `
		age("init", 0).
		deny("Req") :- age("Req", 18).
		`
		f, err := os.CreateTemp("", "policy_typed_*.dl")
		require.NoError(t, err)
		defer os.Remove(f.Name())
		_, err = f.WriteString(policy)
		require.NoError(t, err)
		f.Close()

		// Load raw policy string instead of from file
		err = e.LoadPolicy(context.Background(), policy)
		require.NoError(t, err)

		env := core.NewEnvelope(User{Age: 18})
		// Default is TypeStruct

		err = e.Authorize(context.Background(), core.ActionMetadata{Name: "test"}, env)
		assert.ErrorIs(t, err, core.ErrAlignment)
	})

	// Scenario B: Dynamic Mode (JSON)
	t.Run("Dynamic Mode", func(t *testing.T) {
		e := engine.New()

		// Policy: deny if json_link(Req, "user", U), json_num(U, "age", 20)
		// Includes dummy facts for static analysis
		policy := `
		json_link("init", "init", "init").
		json_num("init", "init", 0).
		deny("Req") :-
			json_link("Req", "user", U),
			json_num(U, "age", 20).
		`
		// Load raw policy string instead of from file
		err := e.LoadPolicy(context.Background(), policy)
		require.NoError(t, err)

		// Input: JSON map
		data := map[string]any{
			"user": map[string]any{
				"age": 20,
			},
		}
		env := core.NewEnvelope(data)
		env.ContentType = core.TypeJSON

		err = e.Authorize(context.Background(), core.ActionMetadata{Name: "test-dynamic"}, env)
		assert.ErrorIs(t, err, core.ErrAlignment)
	})
}
