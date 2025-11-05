package mangle

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRuleSet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mangle_rules_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ruleFile := filepath.Join(tempDir, "rules.dlog")
	err = os.WriteFile(ruleFile, []byte(`deny("test").`), 0644)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		opts := core.MangleOptions{
			Path: []string{tempDir},
		}
		rs, err := New(context.Background(), opts, nil)
		require.NoError(t, err)
		assert.NotNil(t, rs)
	})

	t.Run("no_path", func(t *testing.T) {
		opts := core.MangleOptions{}
		_, err := New(context.Background(), opts, nil)
		assert.Error(t, err)
	})

	t.Run("invalid_path", func(t *testing.T) {
		opts := core.MangleOptions{
			Path: []string{"/invalid/path"},
		}
		_, err := New(context.Background(), opts, nil)
		assert.Error(t, err)
	})

	t.Run("no_rules_found", func(t *testing.T) {
		emptyDir, err := os.MkdirTemp("", "empty_mangle_test")
		require.NoError(t, err)
		defer os.RemoveAll(emptyDir)

		opts := core.MangleOptions{
			Path: []string{emptyDir},
		}
		_, err = New(context.Background(), opts, nil)
		assert.Error(t, err)
	})
}
