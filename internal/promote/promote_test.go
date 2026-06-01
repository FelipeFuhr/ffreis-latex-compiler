package promote

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/testutil"
)

// stage builds an articles root + a dist with a compiled post for slug.
func stage(t *testing.T, postSlug string) (articlesRoot, outRoot string) {
	t.Helper()
	articlesRoot = t.TempDir()
	metaPostSlug := ""
	if postSlug != "hello" {
		metaPostSlug = "post_slug: \"" + postSlug + "\"\n"
	}
	testutil.Article(t, articlesRoot, "hello",
		"title: \"Hi\"\ndate: \"2026-01-02\"\nsummary: \"s\"\n"+metaPostSlug, "tex")

	outRoot = t.TempDir()
	distDir := filepath.Join(outRoot, "hello")
	_ = os.MkdirAll(filepath.Join(distDir, "images"), 0o755)
	index := "---\ntitle: \"Hi\"\ndate: \"2026-01-02\"\nslug: \"" + postSlug + "\"\nsummary: \"s\"\ncanonical_url: \"https://ffreis.com/blog/" + postSlug + "/\"\n---\n\nBody.\n"
	_ = os.WriteFile(filepath.Join(distDir, "index.md"), []byte(index), 0o644)
	_ = os.WriteFile(filepath.Join(distDir, "images", "fig.png"), []byte("x"), 0o644)
	return articlesRoot, outRoot
}

func TestPromoteReal(t *testing.T) {
	articlesRoot, outRoot := stage(t, "hello")
	postsDir := t.TempDir()
	out, err := Run(Options{ArticlesRoot: articlesRoot, OutRoot: outRoot, PostsDir: postsDir, Slug: "hello"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(postsDir, "posts", "hello", "index.md")); err != nil {
		t.Errorf("index.md not staged: %v", err)
	}
	if _, err := os.Stat(filepath.Join(postsDir, "posts", "hello", "images", "fig.png")); err != nil {
		t.Errorf("image not staged: %v", err)
	}
	if !out.Validation.OK() {
		t.Errorf("validation failed: %v", out.Validation.Errors)
	}
}

func TestPromotePostSlugOverride(t *testing.T) {
	articlesRoot, outRoot := stage(t, "custom-post")
	postsDir := t.TempDir()
	out, err := Run(Options{ArticlesRoot: articlesRoot, OutRoot: outRoot, PostsDir: postsDir, Slug: "hello"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.PostSlug != "custom-post" {
		t.Errorf("post slug = %q", out.PostSlug)
	}
	if _, err := os.Stat(filepath.Join(postsDir, "posts", "custom-post", "index.md")); err != nil {
		t.Errorf("not staged under post_slug: %v", err)
	}
}

func TestPromoteDryRun(t *testing.T) {
	articlesRoot, outRoot := stage(t, "hello")
	postsDir := t.TempDir()
	out, err := Run(Options{ArticlesRoot: articlesRoot, OutRoot: outRoot, PostsDir: postsDir, Slug: "hello", DryRun: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(postsDir, "posts")); !os.IsNotExist(err) {
		t.Error("dry run should not write anything")
	}
	if len(out.CopiedFiles) == 0 {
		t.Error("dry run should report planned files")
	}
}

func TestPromoteMissingCompiled(t *testing.T) {
	articlesRoot := t.TempDir()
	testutil.Article(t, articlesRoot, "hello", "title: \"Hi\"\ndate: \"2026-01-02\"\n", "tex")
	_, err := Run(Options{ArticlesRoot: articlesRoot, OutRoot: t.TempDir(), PostsDir: t.TempDir(), Slug: "hello"})
	if err == nil {
		t.Error("expected error when no compiled Markdown present")
	}
}

func TestPromoteStagedInvalidFailsValidation(t *testing.T) {
	articlesRoot := t.TempDir()
	testutil.Article(t, articlesRoot, "hello", "title: \"Hi\"\ndate: \"2026-01-02\"\n", "tex")
	outRoot := t.TempDir()
	distDir := filepath.Join(outRoot, "hello")
	_ = os.MkdirAll(distDir, 0o755)
	// Invalid: bad date and mismatched slug in frontmatter.
	bad := "---\ntitle: \"Hi\"\ndate: \"not-a-date\"\nslug: \"hello\"\n---\nbody\n"
	_ = os.WriteFile(filepath.Join(distDir, "index.md"), []byte(bad), 0o644)
	postsDir := t.TempDir()
	out, err := Run(Options{ArticlesRoot: articlesRoot, OutRoot: outRoot, PostsDir: postsDir, Slug: "hello"})
	if err == nil {
		t.Fatal("expected validation failure error")
	}
	if out == nil || out.Validation.OK() {
		t.Error("expected non-OK validation in outcome")
	}
}
