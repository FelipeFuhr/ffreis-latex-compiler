package snippets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTexInputsFormat(t *testing.T) {
	r := Repo{Root: "/snips"}
	got := r.TexInputs("/article")
	// article dir first (recursive), then snippets root (recursive), then empty.
	if !strings.HasPrefix(got, "/article//:") {
		t.Errorf("article dir not first: %q", got)
	}
	if !strings.Contains(got, "/snips//") {
		t.Errorf("snippets root missing: %q", got)
	}
	if !strings.HasSuffix(got, ":") {
		t.Errorf("missing trailing empty entry: %q", got)
	}
}

func TestTexInputsEmptyRoot(t *testing.T) {
	r := Repo{}
	got := r.TexInputs()
	if got != "" {
		t.Errorf("empty repo, no extras should yield empty trailing entry only, got %q", got)
	}
}

func TestBibInputs(t *testing.T) {
	r := Repo{Root: "/snips"}
	got := r.BibInputs()
	if !strings.Contains(got, filepath.Join("/snips", "bib")+"//") {
		t.Errorf("bib dir missing: %q", got)
	}
}

func TestLocalRefs(t *testing.T) {
	src := `\input{macros/math}
\include{chapters/intro}
\subfile{part}
\usepackage{amsmath}`
	refs := LocalRefs(src)
	want := map[string]bool{"macros/math": true, "chapters/intro": true, "part": true}
	if len(refs) != 3 {
		t.Fatalf("got %d refs: %v", len(refs), refs)
	}
	for _, r := range refs {
		if !want[r] {
			t.Errorf("unexpected ref %q", r)
		}
	}
}

func TestResolve(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "macros", "math.tex"))
	r := Repo{Root: root}

	if _, ok := r.Resolve("macros/math"); !ok {
		t.Error("expected to resolve macros/math via .tex")
	}
	if _, ok := r.Resolve("nope/missing"); ok {
		t.Error("did not expect to resolve missing ref")
	}

	// resolve from a local article dir
	art := t.TempDir()
	touch(t, filepath.Join(art, "local.tex"))
	if _, ok := r.Resolve("local", art); !ok {
		t.Error("expected to resolve from local dir")
	}
}
