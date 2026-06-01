package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAndArticle(t *testing.T) {
	root := t.TempDir()
	p := WriteFile(t, root, "sub/a.txt", "hello")
	if got, _ := os.ReadFile(p); string(got) != "hello" {
		t.Errorf("WriteFile content = %q", got)
	}

	Article(t, root, "demo", "title: \"T\"\n", "\\documentclass{article}")
	for _, rel := range []string{"articles/demo/meta.yaml", "articles/demo/main.tex"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}
