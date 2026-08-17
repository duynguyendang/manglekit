package manglekit_test

import (
	"context"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QuickClient constructs a client and loads the policy file (T-006).
func TestQuickClient(t *testing.T) {
	ctx := context.Background()

	c, err := manglekit.QuickClient(ctx, "sdk/testdata/policy.dl")
	require.NoError(t, err)
	defer c.Shutdown(ctx)
	assert.NotNil(t, c.Engine())

	_, err = manglekit.QuickClient(ctx, "no/such/file.dl")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read policy")
}

// MustReadFile must resolve relative paths against the caller's source
// directory when the cwd does not contain them (T-006). This test file lives
// at the repo root, so "go.mod" resolves via the caller frame even while the
// process cwd is a temp directory.
func TestMustReadFileCWDSafe(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	defer func() { _ = os.Chdir(oldWD) }()

	data := manglekit.MustReadFile("go.mod")
	assert.Contains(t, string(data), "module github.com/duynguyendang/manglekit")
}

// MustReadFile still panics for genuinely missing files.
func TestMustReadFilePanics(t *testing.T) {
	assert.Panics(t, func() {
		manglekit.MustReadFile("definitely/missing/file.txt")
	})
}
