package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "nested", "b.txt")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "hello" {
		t.Errorf("dst content = %q, %v", got, err)
	}
}

func TestCopyFileMissingSrc(t *testing.T) {
	if err := CopyFile(filepath.Join(t.TempDir(), "ghost"), filepath.Join(t.TempDir(), "x")); err == nil {
		t.Error("expected error for missing source")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(src, "a.png"), []byte("1"), 0o644)
	_ = os.WriteFile(filepath.Join(src, "sub", "b.png"), []byte("2"), 0o644)

	dst := filepath.Join(t.TempDir(), "out")
	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir: %v", err)
	}
	for _, rel := range []string{"a.png", "sub/b.png"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Errorf("missing copied file %s: %v", rel, err)
		}
	}
}

func TestCopyDirMissingIsNoop(t *testing.T) {
	if err := CopyDir(filepath.Join(t.TempDir(), "ghost"), t.TempDir()); err != nil {
		t.Errorf("missing src should be a no-op, got %v", err)
	}
}

func TestCopyDirNotADir(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file")
	_ = os.WriteFile(f, []byte("x"), 0o644)
	if err := CopyDir(f, t.TempDir()); err == nil {
		t.Error("expected error when src is a file")
	}
}

func TestCopyFileParentIsFile(t *testing.T) {
	// dst's parent is an existing regular file => MkdirAll fails.
	base := t.TempDir()
	parent := filepath.Join(base, "afile")
	if err := os.WriteFile(parent, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(base, "src.txt")
	_ = os.WriteFile(src, []byte("data"), 0o644)
	if err := CopyFile(src, filepath.Join(parent, "child.txt")); err == nil {
		t.Error("expected error when dst parent is a file")
	}
}
