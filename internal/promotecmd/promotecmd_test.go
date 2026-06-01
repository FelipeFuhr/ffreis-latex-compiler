package promotecmd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestOpenPullRequestSuccess(t *testing.T) {
	var calls [][]string
	git := func(_ context.Context, dir, name string, args ...string) (string, error) {
		calls = append(calls, append([]string{name}, args...))
		return "ok", nil
	}
	err := openPullRequest(context.Background(), quiet(), "/posts", "promote/x", "x", git)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// Expect: checkout, add, commit, push, gh pr create => 5 calls.
	if len(calls) != 5 {
		t.Fatalf("expected 5 git/gh calls, got %d: %v", len(calls), calls)
	}
	if calls[0][0] != "git" || calls[0][1] != "checkout" || calls[0][3] != "promote/x" {
		t.Errorf("checkout call wrong: %v", calls[0])
	}
	last := calls[4]
	if last[0] != "gh" || last[1] != "pr" {
		t.Errorf("expected gh pr create, got %v", last)
	}
}

func TestOpenPullRequestRefusesMain(t *testing.T) {
	git := func(context.Context, string, string, ...string) (string, error) { return "", nil }
	if err := openPullRequest(context.Background(), quiet(), "/posts", "main", "x", git); err == nil {
		t.Error("expected refusal to commit onto main")
	}
}

func TestOpenPullRequestGitError(t *testing.T) {
	git := func(_ context.Context, _, name string, _ ...string) (string, error) {
		if name == "git" {
			return "fail", errors.New("boom")
		}
		return "", nil
	}
	err := openPullRequest(context.Background(), quiet(), "/posts", "promote/x", "x", git)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected git error, got %v", err)
	}
}

func TestRunRequiresSlugAndPostsDir(t *testing.T) {
	if err := Run([]string{"-posts-dir", "/p"}, quiet()); err == nil {
		t.Error("expected error when -slug missing")
	}
	if err := Run([]string{"-slug", "x"}, quiet()); err == nil {
		t.Error("expected error when -posts-dir missing")
	}
}

func TestRunDryRunPath(t *testing.T) {
	dir := t.TempDir()
	// articles/hello + meta
	_ = os.MkdirAll(filepath.Join(dir, "articles", "hello"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "articles", "hello", "main.tex"), []byte("tex"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "articles", "hello", "meta.yaml"),
		[]byte("title: \"Hi\"\ndate: \"2026-01-02\"\n"), 0o644)
	// dist/hello/index.md
	out := t.TempDir()
	_ = os.MkdirAll(filepath.Join(out, "hello"), 0o755)
	_ = os.WriteFile(filepath.Join(out, "hello", "index.md"),
		[]byte("---\ntitle: \"Hi\"\ndate: \"2026-01-02\"\nslug: \"hello\"\n---\nbody\n"), 0o644)

	postsDir := t.TempDir()
	err := Run([]string{
		"-articles-root", dir, "-out", out, "-posts-dir", postsDir,
		"-slug", "hello", "-dry-run",
	}, quiet())
	if err != nil {
		t.Fatalf("dry-run Run: %v", err)
	}
	// dry run must not write into posts/
	if _, err := os.Stat(filepath.Join(postsDir, "posts")); !os.IsNotExist(err) {
		t.Error("dry-run should not create posts/")
	}
}

func TestExecGit(t *testing.T) {
	out, err := execGit(context.Background(), t.TempDir(), "git", "--version")
	if err != nil {
		t.Skipf("git not available: %v", err)
	}
	if !strings.Contains(out, "git version") {
		t.Errorf("unexpected git output: %q", out)
	}
	if _, err := execGit(context.Background(), t.TempDir(), "definitely-not-a-binary-xyz"); err == nil {
		t.Error("expected error for missing binary")
	}
}
