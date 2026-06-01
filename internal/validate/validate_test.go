package validate

import (
	"strings"
	"testing"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/testutil"
)

func TestValidateHappy(t *testing.T) {
	root := t.TempDir()
	testutil.Article(t, root, "good",
		"title: \"Good\"\ndate: \"2026-01-02\"\nsummary: \"s\"\ncanonical_url: \"https://ffreis.com/blog/good/\"\n",
		"\\documentclass{article}")
	results, err := Run(Options{ArticlesRoot: root, Slug: "good"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].OK() {
		t.Errorf("expected valid, got %+v", results)
	}
}

func TestValidateUnresolvedSnippet(t *testing.T) {
	root := t.TempDir()
	testutil.Article(t, root, "bad",
		"title: \"T\"\ndate: \"2026-01-02\"\nsummary: \"s\"\ncanonical_url: \"https://ffreis.com/blog/bad/\"\n",
		"\\input{macros/missing}\n")
	results, err := Run(Options{ArticlesRoot: root, Slug: "bad", SnippetsRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].OK() || !strings.Contains(strings.Join(results[0].Errors, "\n"), "unresolved") {
		t.Errorf("expected unresolved snippet error, got %+v", results[0])
	}
}

func TestValidateResolvedSnippet(t *testing.T) {
	root := t.TempDir()
	snip := t.TempDir()
	testutil.WriteFile(t, snip, "macros/math.tex", "\\newcommand{\\R}{\\mathbb{R}}")
	testutil.Article(t, root, "ok",
		"title: \"T\"\ndate: \"2026-01-02\"\nsummary: \"s\"\ncanonical_url: \"https://ffreis.com/blog/ok/\"\n",
		"\\input{macros/math}\n")
	results, err := Run(Options{ArticlesRoot: root, Slug: "ok", SnippetsRoot: snip})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].OK() {
		t.Errorf("expected valid with resolved snippet, got %+v", results[0])
	}
}

func TestValidateAllAndMissingMeta(t *testing.T) {
	root := t.TempDir()
	testutil.Article(t, root, "x", "date: \"2026-01-02\"\n", "x") // missing title
	results, err := Run(Options{ArticlesRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].OK() {
		t.Error("expected title error")
	}
}

func TestValidateLoadError(t *testing.T) {
	if _, err := Run(Options{ArticlesRoot: t.TempDir(), Slug: "ghost"}); err == nil {
		t.Error("expected error for missing article")
	}
}
