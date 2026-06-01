// Package fsutil holds small filesystem helpers shared across commands:
// copying files and directory trees with sensible permissions.
package fsutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies src to dst, creating parent directories as needed.
func CopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	in, err := os.Open(src) //nolint:gosec // paths are derived from trusted local roots
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst) //nolint:gosec // paths are derived from trusted local roots
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// CopyDir recursively copies the directory tree at src into dst. A missing src
// is treated as a no-op (the source dir is optional). Symlinks are skipped.
func CopyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o750)
		case d.Type()&os.ModeSymlink != 0:
			return nil // skip symlinks
		default:
			return CopyFile(path, target)
		}
	})
}
