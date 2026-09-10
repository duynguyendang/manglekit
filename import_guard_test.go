package manglekit_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestImportGuard_OODAOutOfCore enforces the ADR-004 layering rule with the
// import graph itself: nothing outside manglekit/x/ may import the x/
// extensions (governance core stays cognition-free), and the retired paths
// (sdk/ooda, manglekit/agents) must not be imported anywhere.
func TestImportGuard_OODAOutOfCore(t *testing.T) {
	const module = "github.com/duynguyendang/manglekit"
	fset := token.NewFileSet()

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || strings.HasPrefix(name, ".") && name != "." {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		inX := strings.HasPrefix(filepath.ToSlash(path), "x/")
		for _, imp := range f.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			switch {
			case !inX && strings.HasPrefix(ip, module+"/x/"):
				t.Errorf("%s imports %s: core/sdk/adapters/internal must not import x/ (ADR-004)", path, ip)
			case ip == module+"/sdk/ooda" || strings.HasPrefix(ip, module+"/sdk/ooda/"):
				t.Errorf("%s imports retired path %s (moved to x/ooda in v0.10)", path, ip)
			case ip == module+"/agents" || strings.HasPrefix(ip, module+"/agents/"):
				t.Errorf("%s imports retired path %s (moved to x/agents in v0.10)", path, ip)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
