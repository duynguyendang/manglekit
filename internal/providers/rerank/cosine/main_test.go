//go:build testhooks
// +build testhooks

package cosine_test

import (
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/internal/registry"
)

func TestMain(m *testing.M) {
	registry.ResetForTest()
	os.Exit(m.Run())
}
