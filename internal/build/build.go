// Package build orchestrates compilation: it loads article sources, wires up
// the snippets search paths, drives the per-format engines, and post-processes
// the Markdown target into a ffreis-posts-shaped index.md + images/ tree. It is
// deliberately decoupled from flag parsing (see internal/buildcmd) and from the
// concrete engines (injected as a RendererSet) so it can be unit-tested with
// fakes and without the real toolchain.
package build

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/article"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/engine"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/fsutil"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/posts"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/snippets"
)

// Options configures a build run.
type Options struct {
	ArticlesRoot string          // root containing articles/<slug>/
	SnippetsRoot string          // root of the ffreis-snippets repo ("" = none)
	OutRoot      string          // output root (dist)
	Slug         string          // single article slug; "" = all articles
	Formats      []engine.Format // formats to produce
}

// RendererSet holds one renderer per format. Any may be nil/Noop to disable it.
type RendererSet struct {
	PDF  engine.Renderer
	HTML engine.Renderer
	MD   engine.Renderer
}

// DefaultRenderers returns the production renderers backed by the real tools.
func DefaultRenderers() RendererSet {
	return RendererSet{
		PDF:  engine.NewTectonic(nil),
		HTML: engine.NewMake4ht(nil),
		MD:   engine.NewPandoc(nil),
	}
}

func (rs RendererSet) forFormat(f engine.Format) engine.Renderer {
	switch f {
	case engine.FormatPDF:
		return rs.PDF
	case engine.FormatHTML:
		return rs.HTML
	case engine.FormatMarkdown:
		return rs.MD
	default:
		return nil
	}
}

// Run compiles the selected article(s) into OutRoot.
func Run(ctx context.Context, logger *slog.Logger, opts Options, rs RendererSet) error {
	// Engines run with their working directory set to the article dir, so all
	// roots must be absolute or relative paths would resolve against the wrong
	// directory once the tool's CWD changes.
	opts, err := absolutize(opts)
	if err != nil {
		return err
	}

	articles, err := selectArticles(opts)
	if err != nil {
		return err
	}
	if len(articles) == 0 {
		return fmt.Errorf("no articles found under %s", filepath.Join(opts.ArticlesRoot, "articles"))
	}

	snip := snippets.Repo{Root: opts.SnippetsRoot}
	for _, a := range articles {
		if err := buildOne(ctx, logger, opts, rs, snip, a); err != nil {
			return fmt.Errorf("build %q: %w", a.Slug, err)
		}
	}
	return nil
}

// absolutize resolves the article, snippets, and output roots to absolute paths.
func absolutize(opts Options) (Options, error) {
	var err error
	if opts.ArticlesRoot, err = filepath.Abs(opts.ArticlesRoot); err != nil {
		return opts, err
	}
	if opts.OutRoot, err = filepath.Abs(opts.OutRoot); err != nil {
		return opts, err
	}
	if opts.SnippetsRoot != "" {
		if opts.SnippetsRoot, err = filepath.Abs(opts.SnippetsRoot); err != nil {
			return opts, err
		}
	}
	return opts, nil
}

func selectArticles(opts Options) ([]*article.Article, error) {
	if opts.Slug != "" {
		a, err := article.Load(opts.ArticlesRoot, opts.Slug)
		if err != nil {
			return nil, err
		}
		return []*article.Article{a}, nil
	}
	return article.LoadAll(opts.ArticlesRoot)
}

func buildOne(ctx context.Context, logger *slog.Logger, opts Options, rs RendererSet, snip snippets.Repo, a *article.Article) error {
	outDir := filepath.Join(opts.OutRoot, a.Slug)
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return err
	}
	job := engine.Job{
		SourceTeX: a.MainTeX,
		WorkDir:   a.Dir,
		OutDir:    outDir,
		Slug:      a.Slug,
		TexDirs:   snip.TexDirs(a.Dir),
		BibDirs:   snip.BibDirs(a.Dir),
	}

	for _, f := range opts.Formats {
		r := rs.forFormat(f)
		if r == nil {
			return fmt.Errorf("no renderer configured for format %q", f)
		}
		logger.Info("rendering", "slug", a.Slug, "format", string(f), "tool", r.Tool())
		out, err := r.Render(ctx, job)
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		if f == engine.FormatMarkdown && out != "" {
			if err := finalizeMarkdown(a, outDir, out); err != nil {
				return fmt.Errorf("markdown post-process: %w", err)
			}
		}
	}
	return nil
}

// finalizeMarkdown turns the raw pandoc body into a ffreis-posts index.md:
// normalise image links, prepend generated frontmatter, copy the article's
// images/ dir, and drop the intermediate body file.
func finalizeMarkdown(a *article.Article, outDir, bodyPath string) error {
	raw, err := os.ReadFile(bodyPath) //nolint:gosec // bodyPath is under the build OutDir
	if err != nil {
		return err
	}
	body := posts.NormalizeImageLinks(string(raw))
	fm := posts.BuildFrontmatter(a.Meta, a.PostSlug())
	index := posts.AssembleIndexMD(fm, body)

	if err := os.WriteFile(filepath.Join(outDir, "index.md"), []byte(index), 0o644); err != nil { //nolint:gosec // generated post is world-readable by design
		return err
	}
	if err := fsutil.CopyDir(filepath.Join(a.Dir, article.ImagesDir), filepath.Join(outDir, article.ImagesDir)); err != nil {
		return err
	}
	return os.Remove(bodyPath)
}
