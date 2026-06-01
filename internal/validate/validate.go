// Package validate checks article sources before they are compiled or
// promoted. It applies the shared ffreis-posts field rules (via posts.CheckFields)
// plus LaTeX-source-specific checks: the slug shape and that every local
// \input/\include/\subfile reference resolves against the snippets repo or the
// article directory.
package validate

import (
	"fmt"
	"os"
	"regexp"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/article"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/posts"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/snippets"
)

var slugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Options configures a validate run.
type Options struct {
	ArticlesRoot string
	SnippetsRoot string
	Slug         string // single article; "" = all
}

// Run validates the selected article(s) and returns one Result each.
func Run(opts Options) ([]posts.Result, error) {
	var articles []*article.Article
	if opts.Slug != "" {
		a, err := article.Load(opts.ArticlesRoot, opts.Slug)
		if err != nil {
			return nil, err
		}
		articles = []*article.Article{a}
	} else {
		all, err := article.LoadAll(opts.ArticlesRoot)
		if err != nil {
			return nil, err
		}
		articles = all
	}

	snip := snippets.Repo{Root: opts.SnippetsRoot}
	results := make([]posts.Result, 0, len(articles))
	for _, a := range articles {
		results = append(results, validateArticle(a, snip))
	}
	return results, nil
}

func validateArticle(a *article.Article, snip snippets.Repo) posts.Result {
	res := posts.Result{Slug: a.Slug}

	if !slugRE.MatchString(a.Slug) {
		res.Errors = append(res.Errors,
			fmt.Sprintf("directory name must be lowercase letters, numbers, and hyphens (got %q)", a.Slug))
	}

	errs, warns := posts.CheckFields(posts.Fields{
		Slug:         a.Slug,
		FrontSlug:    a.Meta.Slug,
		Title:        a.Meta.Title,
		Date:         a.Meta.Date,
		Summary:      a.Meta.Summary,
		Thumbnail:    a.Meta.Thumbnail,
		CanonicalURL: a.Meta.CanonicalURL,
		Tags:         a.Meta.Tags,
		Dir:          a.Dir,
		// Body left empty: LaTeX source is not Markdown, so the body image and
		// raw-HTML checks don't apply here (they run post-build on index.md).
	})
	res.Errors = append(res.Errors, errs...)
	res.Warnings = append(res.Warnings, warns...)

	res.Errors = append(res.Errors, checkSnippetRefs(a, snip)...)
	return res
}

func checkSnippetRefs(a *article.Article, snip snippets.Repo) []string {
	src, err := os.ReadFile(a.MainTeX) //nolint:gosec // MainTeX is under a trusted articles root
	if err != nil {
		return []string{fmt.Sprintf("cannot read %s: %v", article.MainTeXName, err)}
	}
	var errs []string
	for _, ref := range snippets.LocalRefs(string(src)) {
		if _, ok := snip.Resolve(ref, a.Dir); !ok {
			errs = append(errs, fmt.Sprintf("unresolved \\input/\\include reference %q (not found in snippets repo or article dir)", ref))
		}
	}
	return errs
}
