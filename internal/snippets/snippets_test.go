package snippets

import (
	"os"
	"path/filepath"
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

func TestTexDirs(t *testing.T) {
	r := Repo{Root: "/snips"}
	got := r.TexDirs("/article")
	// article dir first, then snippets root, then each standard subdir.
	if len(got) == 0 || got[0] != "/article" {
		t.Errorf("article dir not first: %v", got)
	}
	want := map[string]bool{
		"/snips": true, "/snips/preambles": true, "/snips/classes": true,
		"/snips/macros": true, "/snips/figures": true,
	}
	for w := range want {
		found := false
		for _, d := range got {
			if d == w {
				found = true
			}
		}
		if !found {
			t.Errorf("missing search dir %q in %v", w, got)
		}
	}
}

func TestTexDirsEmptyRoot(t *testing.T) {
	if got := (Repo{}).TexDirs(); len(got) != 0 {
		t.Errorf("empty repo with no extras should yield no dirs, got %v", got)
	}
	// Empty extras are dropped.
	if got := (Repo{}).TexDirs(""); len(got) != 0 {
		t.Errorf("empty extra should be dropped, got %v", got)
	}
}

func TestBibDirs(t *testing.T) {
	got := Repo{Root: "/snips"}.BibDirs("/article")
	if len(got) != 2 || got[0] != "/article" || got[1] != filepath.Join("/snips", "bib") {
		t.Errorf("unexpected bib dirs: %v", got)
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
