package validatecmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/posts"
)

func TestReportAllValid(t *testing.T) {
	var buf bytes.Buffer
	err := report(&buf, []posts.Result{{Slug: "a"}, {Slug: "b"}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(buf.String(), "All articles valid.") {
		t.Errorf("output: %s", buf.String())
	}
}

func TestReportWithErrors(t *testing.T) {
	var buf bytes.Buffer
	err := report(&buf, []posts.Result{{Slug: "a", Errors: []string{"bad title"}}})
	if err == nil {
		t.Fatal("expected error return when an article has errors")
	}
	if !strings.Contains(buf.String(), "✗") || !strings.Contains(buf.String(), "bad title") {
		t.Errorf("output: %s", buf.String())
	}
}

func TestReportWithWarningsOnly(t *testing.T) {
	var buf bytes.Buffer
	err := report(&buf, []posts.Result{{Slug: "a", Warnings: []string{"no summary"}}})
	if err != nil {
		t.Fatalf("warnings should not fail: %v", err)
	}
	if !strings.Contains(buf.String(), "⚠") {
		t.Errorf("output: %s", buf.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	if err := Run([]string{"-bogus"}, nil); err == nil {
		t.Error("expected flag error")
	}
}

func TestRunEndToEnd(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "articles", "ok")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(base, "main.tex"), []byte("\\documentclass{article}"), 0o644)
	_ = os.WriteFile(filepath.Join(base, "meta.yaml"),
		[]byte("title: \"T\"\ndate: \"2026-01-02\"\nsummary: \"s\"\ncanonical_url: \"https://ffreis.com/blog/ok/\"\n"), 0o644)
	if err := Run([]string{"-articles-root", root}, nil); err != nil {
		t.Fatalf("Run valid: %v", err)
	}
}

func TestRunEndToEndInvalid(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "articles", "bad")
	_ = os.MkdirAll(base, 0o755)
	_ = os.WriteFile(filepath.Join(base, "main.tex"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(base, "meta.yaml"), []byte("date: \"2026-01-02\"\n"), 0o644) // no title
	if err := Run([]string{"-articles-root", root}, nil); err == nil {
		t.Error("expected error for invalid article")
	}
}
