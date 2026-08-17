package kg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertTurtle(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "input.ttl")
	// Note: the knowledge-rdf turtle parser is prefix/dot naive, so use
	// simple prefixed terms (matching what `mkit eval --knowledge` accepts).
	ttl := "@prefix ex: <http://example.org/> .\nex:s ex:p \"value\" .\n"
	if err := os.WriteFile(in, []byte(ttl), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.nq")

	inFile, outFile, inFmt, outFmt = in, out, "", "nquads"
	if err := ConvertCmd.RunE(ConvertCmd, nil); err != nil {
		t.Fatalf("expected .ttl conversion to succeed, got: %v", err)
	}

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "<ex:s>") {
		t.Errorf("expected converted output to contain the subject, got: %q", string(got))
	}
	if !strings.Contains(string(got), "\"value\"") {
		t.Errorf("expected converted output to contain the object literal, got: %q", string(got))
	}
}

func TestConvertNQuadsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "input.nq")
	quad := "<http://example.org/s> <http://example.org/p> <http://example.org/o> .\n"
	if err := os.WriteFile(in, []byte(quad), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.nq")

	inFile, outFile, inFmt, outFmt = in, out, "nquads", "nquads"
	if err := ConvertCmd.RunE(ConvertCmd, nil); err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "<http://example.org/s>") {
		t.Errorf("expected converted output to contain the subject, got: %q", string(got))
	}
}

func TestConvertUnknownExtensionErrors(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "input.xyz")
	if err := os.WriteFile(in, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.nq")

	inFile, outFile, inFmt, outFmt = in, out, "", "nquads"
	err := ConvertCmd.RunE(ConvertCmd, nil)
	if err == nil {
		t.Fatal("expected error for unknown input extension")
	}
}
