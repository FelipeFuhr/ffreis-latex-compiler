package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// BodyMarkdownName is the raw Markdown body pandoc emits, before frontmatter is
// prepended by the posts package.
const BodyMarkdownName = "body.md"

// Pandoc renders a Job's LaTeX source to GitHub-Flavored Markdown suitable for
// ffreis-posts and Medium. It extracts referenced graphics into OutDir/images
// and emits no raw HTML wrappers, keeping the output Medium-safe.
type Pandoc struct {
	run Runner
}

// NewPandoc returns a Pandoc renderer. A nil runner falls back to ExecRunner.
func NewPandoc(run Runner) *Pandoc {
	if run == nil {
		run = ExecRunner
	}
	return &Pandoc{run: run}
}

func (p *Pandoc) Format() Format  { return FormatMarkdown }
func (p *Pandoc) Tool() string    { return "pandoc" }
func (p *Pandoc) Available() bool { return toolAvailable(p.Tool()) }

// Render converts main.tex to OutDir/body.md and returns its path. Figures are
// left as authored (e.g. images/foo.png); the build step copies the article's
// images/ dir alongside and the posts package normalises the link prefix to the
// ./images/ convention ffreis-posts uses.
func (p *Pandoc) Render(ctx context.Context, j Job) (string, error) {
	if err := os.MkdirAll(j.OutDir, 0o755); err != nil {
		return "", fmt.Errorf("pandoc: mkdir outdir: %w", err)
	}
	body := filepath.Join(j.OutDir, BodyMarkdownName)
	args := []string{
		j.SourceTeX,
		"--from=latex",
		// gfm with the raw_html extension DISABLED: pandoc then renders figures,
		// images, and cross-references as pure Markdown instead of emitting
		// <figure>/<img>/<a> HTML, keeping the output Medium-safe.
		"--to=gfm-raw_html",
		"--wrap=none",
		"--markdown-headings=atx",
		"-o", body,
	}
	if err := p.run(ctx, j.WorkDir, jobEnv(j), p.Tool(), args...); err != nil {
		return "", err
	}
	return body, nil
}
