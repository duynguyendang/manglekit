package bm25_test

import (
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/v1/internal/registry"
)

func TestMain(m *testing.M) {
	registry.ResetForTest()
	os.Exit(m.Run())
}
