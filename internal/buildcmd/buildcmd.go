// Package buildcmd parses flags for the `build`/`compile` command and delegates
// to internal/build.
package buildcmd

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/build"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/engine"
)

// Run executes the build command.
func Run(args []string, logger *slog.Logger) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	articlesRoot := fs.String("articles-root", ".", "root containing articles/<slug>/")
	snippetsRoot := fs.String("snippets-root", "", "root of the snippets repo (optional)")
	out := fs.String("out", "dist", "output directory")
	slug := fs.String("slug", "", "compile only this article slug (default: all)")
	formatsStr := fs.String("formats", "pdf,html,md", "comma-separated output formats: pdf,html,md")
	if err := fs.Parse(args); err != nil {
		return err
	}

	formats, err := ParseFormats(*formatsStr)
	if err != nil {
		return err
	}

	return build.Run(context.Background(), logger, build.Options{
		ArticlesRoot: *articlesRoot,
		SnippetsRoot: *snippetsRoot,
		OutRoot:      *out,
		Slug:         *slug,
		Formats:      formats,
	}, build.DefaultRenderers())
}

// ParseFormats parses a comma-separated format list into engine.Format values,
// rejecting unknown or empty entries.
func ParseFormats(s string) ([]engine.Format, error) {
	parts := strings.Split(s, ",")
	formats := make([]engine.Format, 0, len(parts))
	seen := map[engine.Format]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f := engine.Format(p)
		switch f {
		case engine.FormatPDF, engine.FormatHTML, engine.FormatMarkdown:
		default:
			return nil, fmt.Errorf("unknown format %q (valid: pdf, html, md)", p)
		}
		if !seen[f] {
			seen[f] = true
			formats = append(formats, f)
		}
	}
	if len(formats) == 0 {
		return nil, fmt.Errorf("no formats selected")
	}
	return formats, nil
}
