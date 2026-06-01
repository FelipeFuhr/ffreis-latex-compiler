// Package testutil provides helpers for building temporary article/snippets
// trees in tests.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteFile writes content to path under dir, creating parents. Returns the
// full path.
func WriteFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
	return full
}

// Article writes a minimal valid article (main.tex + meta.yaml) under
// root/articles/<slug>/ and returns root. Extra files may be added by the
// caller via WriteFile.
func Article(t *testing.T, root, slug, meta, tex string) {
	t.Helper()
	base := filepath.Join("articles", slug)
	WriteFile(t, root, filepath.Join(base, "meta.yaml"), meta)
	WriteFile(t, root, filepath.Join(base, "main.tex"), tex)
}
