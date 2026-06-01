// Package validatecmd parses flags for the `validate` command, prints a
// per-article report, and exits non-zero when any article has errors.
package validatecmd

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/posts"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/validate"
)

// Run executes the validate command.
func Run(args []string, _ *slog.Logger) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	articlesRoot := fs.String("articles-root", ".", "root containing articles/<slug>/")
	snippetsRoot := fs.String("snippets-root", "", "root of the ffreis-snippets repo (optional)")
	slug := fs.String("slug", "", "validate only this article slug (default: all)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	results, err := validate.Run(validate.Options{
		ArticlesRoot: *articlesRoot,
		SnippetsRoot: *snippetsRoot,
		Slug:         *slug,
	})
	if err != nil {
		return err
	}
	return report(os.Stdout, results)
}

// report prints results and returns an error if any article has errors.
func report(w io.Writer, results []posts.Result) error {
	totalErr, totalWarn := 0, 0
	for _, r := range results {
		totalErr += len(r.Errors)
		totalWarn += len(r.Warnings)
		if r.OK() && len(r.Warnings) == 0 {
			fmt.Fprintf(w, "articles/%s/  ✓\n", r.Slug)
			continue
		}
		fmt.Fprintf(w, "\narticles/%s/\n", r.Slug)
		for _, m := range r.Errors {
			fmt.Fprintf(w, "  ✗  %s\n", m)
		}
		for _, m := range r.Warnings {
			fmt.Fprintf(w, "  ⚠  %s\n", m)
		}
	}
	fmt.Fprintln(w)
	if totalErr > 0 {
		fmt.Fprintf(w, "%d error(s), %d warning(s) — fix errors before compiling or promoting.\n", totalErr, totalWarn)
		return fmt.Errorf("%d article(s) failed validation", totalErr)
	}
	if totalWarn > 0 {
		fmt.Fprintf(w, "0 errors, %d warning(s).\n", totalWarn)
		return nil
	}
	fmt.Fprintln(w, "All articles valid.")
	return nil
}
