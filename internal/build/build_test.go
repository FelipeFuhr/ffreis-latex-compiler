package build

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/engine"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/testutil"
)

type fakeRenderer struct {
	format engine.Format
	fn     func(j engine.Job) (string, error)
}

func (f fakeRenderer) Format() engine.Format                                  { return f.format }
func (f fakeRenderer) Tool() string                                           { return "fake-" + string(f.format) }
func (f fakeRenderer) Available() bool                                        { return true }
func (f fakeRenderer) Render(_ context.Context, j engine.Job) (string, error) { return f.fn(j) }

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func setupArticle(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	testutil.Article(t, root, "hello",
		"title: \"Hello World\"\ndate: \"2026-01-02\"\nsummary: \"s\"\n",
		"\\documentclass{article}\\begin{document}hi\\end{document}")
	testutil.WriteFile(t, root, "articles/hello/images/fig.png", "imgdata")
	return root
}

func TestBuildMarkdownFinalization(t *testing.T) {
	root := setupArticle(t)
	out := filepath.Join(t.TempDir(), "dist")

	md := fakeRenderer{format: engine.FormatMarkdown, fn: func(j engine.Job) (string, error) {
		body := filepath.Join(j.OutDir, engine.BodyMarkdownName)
		_ = os.WriteFile(body, []byte("Body with ![](images/fig.png)\n"), 0o644)
		return body, nil
	}}

	err := Run(context.Background(), quietLogger(), Options{
		ArticlesRoot: root,
		OutRoot:      out,
		Slug:         "hello",
		Formats:      []engine.Format{engine.FormatMarkdown},
	}, RendererSet{MD: md})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	index, err := os.ReadFile(filepath.Join(out, "hello", "index.md"))
	if err != nil {
		t.Fatalf("index.md: %v", err)
	}
	s := string(index)
	if !strings.Contains(s, `title: "Hello World"`) {
		t.Errorf("frontmatter missing title:\n%s", s)
	}
	if !strings.Contains(s, "![](./images/fig.png)") {
		t.Errorf("image link not normalized:\n%s", s)
	}
	if _, err := os.Stat(filepath.Join(out, "hello", "images", "fig.png")); err != nil {
		t.Errorf("images not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "hello", engine.BodyMarkdownName)); !os.IsNotExist(err) {
		t.Errorf("intermediate body.md not removed")
	}
}

func TestBuildAllFormatsDispatch(t *testing.T) {
	root := setupArticle(t)
	out := filepath.Join(t.TempDir(), "dist")
	called := map[engine.Format]bool{}
	mk := func(f engine.Format) fakeRenderer {
		return fakeRenderer{format: f, fn: func(j engine.Job) (string, error) {
			called[f] = true
			p := filepath.Join(j.OutDir, j.Slug+"."+string(f))
			_ = os.WriteFile(p, []byte("x"), 0o644)
			return p, nil
		}}
	}
	err := Run(context.Background(), quietLogger(), Options{
		ArticlesRoot: root, OutRoot: out, Slug: "hello",
		Formats: []engine.Format{engine.FormatPDF, engine.FormatHTML},
	}, RendererSet{PDF: mk(engine.FormatPDF), HTML: mk(engine.FormatHTML)})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called[engine.FormatPDF] || !called[engine.FormatHTML] {
		t.Errorf("not all renderers called: %v", called)
	}
}

func TestBuildNoArticles(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "articles"), 0o755)
	err := Run(context.Background(), quietLogger(), Options{
		ArticlesRoot: root, OutRoot: t.TempDir(), Formats: []engine.Format{engine.FormatPDF},
	}, RendererSet{PDF: fakeRenderer{format: engine.FormatPDF, fn: func(engine.Job) (string, error) { return "", nil }}})
	if err == nil || !strings.Contains(err.Error(), "no articles") {
		t.Errorf("expected no-articles error, got %v", err)
	}
}

func TestBuildMissingRenderer(t *testing.T) {
	root := setupArticle(t)
	err := Run(context.Background(), quietLogger(), Options{
		ArticlesRoot: root, OutRoot: t.TempDir(), Slug: "hello",
		Formats: []engine.Format{engine.FormatPDF},
	}, RendererSet{}) // no PDF renderer
	if err == nil || !strings.Contains(err.Error(), "no renderer") {
		t.Errorf("expected missing-renderer error, got %v", err)
	}
}

func TestDefaultRenderersWired(t *testing.T) {
	rs := DefaultRenderers()
	if rs.PDF.Tool() != "tectonic" || rs.HTML.Tool() != "make4ht" || rs.MD.Tool() != "pandoc" {
		t.Errorf("default renderers mis-wired: %s/%s/%s", rs.PDF.Tool(), rs.HTML.Tool(), rs.MD.Tool())
	}
}

func TestBuildMarkdownBodyMissing(t *testing.T) {
	root := setupArticle(t)
	out := filepath.Join(t.TempDir(), "dist")
	// MD renderer returns a path that does not exist => finalizeMarkdown read fails.
	md := fakeRenderer{format: engine.FormatMarkdown, fn: func(j engine.Job) (string, error) {
		return filepath.Join(j.OutDir, "ghost.md"), nil
	}}
	err := Run(context.Background(), quietLogger(), Options{
		ArticlesRoot: root, OutRoot: out, Slug: "hello",
		Formats: []engine.Format{engine.FormatMarkdown},
	}, RendererSet{MD: md})
	if err == nil || !strings.Contains(err.Error(), "post-process") {
		t.Errorf("expected markdown post-process error, got %v", err)
	}
}

func TestBuildRelativeRootsAbsolutized(t *testing.T) {
	// Exercises absolutize with a relative articles root.
	root := setupArticle(t)
	rel, err := filepath.Rel(mustWD(t), root)
	if err != nil {
		t.Skip("cannot relativize temp dir")
	}
	out := filepath.Join(t.TempDir(), "dist")
	md := fakeRenderer{format: engine.FormatMarkdown, fn: func(j engine.Job) (string, error) {
		p := filepath.Join(j.OutDir, engine.BodyMarkdownName)
		_ = os.WriteFile(p, []byte("body"), 0o644)
		return p, nil
	}}
	if err := Run(context.Background(), quietLogger(), Options{
		ArticlesRoot: rel, OutRoot: out, Slug: "hello",
		Formats: []engine.Format{engine.FormatMarkdown},
	}, RendererSet{MD: md}); err != nil {
		t.Fatalf("Run with relative root: %v", err)
	}
}

func mustWD(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}
