// Package posts turns a compiled article's raw Markdown body into a
// ffreis-posts-compatible index.md (YAML frontmatter + Medium-safe GFM), and
// re-implements the ffreis-posts validation rules so the compiler can guarantee
// a promoted post will pass downstream CI. The rule set intentionally mirrors
// ffreis-posts/scripts/validate-posts.py field-for-field.
package posts

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/article"
)

// Limits and shapes shared with ffreis-posts/scripts/validate-posts.py.
const (
	MaxTitle      = 250
	MaxTags       = 5
	CanonicalBase = "https://ffreis.com/blog/"
)

var (
	slugRE    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	dateRE    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	imgBodyRE = regexp.MustCompile(`!\[[^\]]*\]\((\./images/[^)]+)\)`)
	rawHTMLRE = regexp.MustCompile(`(?i)<(div|script|style|span|iframe|form)\b`)
	// pandoc emits links relative to the output file: images/<name>. Normalise
	// both the bare and the explicitly-relative forms to ./images/.
	extractedImgRE = regexp.MustCompile(`(!\[[^\]]*\]\()(\.?/?images/)`)
)

// Frontmatter is the ffreis-posts post header. Field order is fixed by Render.
type Frontmatter struct {
	Title           string
	Date            string
	Slug            string
	Summary         string
	Thumbnail       string
	Tags            []string
	CanonicalURL    string
	MediumPublished bool
}

// BuildFrontmatter derives a post header from article metadata. The canonical
// URL defaults to the ffreis.com blog URL for the slug when unset, so generated
// posts are always explicit (and never trip the missing-canonical warning).
func BuildFrontmatter(meta article.Meta, slug string) Frontmatter {
	canonical := meta.CanonicalURL
	if canonical == "" {
		canonical = CanonicalBase + slug + "/"
	}
	return Frontmatter{
		Title:        meta.Title,
		Date:         meta.Date,
		Slug:         slug,
		Summary:      meta.Summary,
		Thumbnail:    meta.Thumbnail,
		Tags:         meta.Tags,
		CanonicalURL: canonical,
	}
}

// Render serialises the frontmatter to the exact YAML shape validate-posts.py
// parses: double-quoted scalars, inline tag list, boolean medium_published.
func (f Frontmatter) Render() string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %s\n", quote(f.Title))
	fmt.Fprintf(&b, "date: %s\n", quote(f.Date))
	fmt.Fprintf(&b, "slug: %s\n", quote(f.Slug))
	if f.Summary != "" {
		fmt.Fprintf(&b, "summary: %s\n", quote(f.Summary))
	}
	if f.Thumbnail != "" {
		fmt.Fprintf(&b, "thumbnail: %s\n", quote(f.Thumbnail))
	}
	if len(f.Tags) > 0 {
		quoted := make([]string, len(f.Tags))
		for i, t := range f.Tags {
			quoted[i] = quote(t)
		}
		fmt.Fprintf(&b, "tags: [%s]\n", strings.Join(quoted, ", "))
	}
	fmt.Fprintf(&b, "canonical_url: %s\n", quote(f.CanonicalURL))
	fmt.Fprintf(&b, "medium_published: %t\n", f.MediumPublished)
	b.WriteString("---\n")
	return b.String()
}

// AssembleIndexMD prepends the frontmatter to a normalised Markdown body.
func AssembleIndexMD(f Frontmatter, body string) string {
	return f.Render() + "\n" + strings.TrimLeft(body, "\n")
}

// NormalizeImageLinks rewrites pandoc's extracted-media links (images/… or
// /images/…) to the ./images/… convention ffreis-posts expects.
func NormalizeImageLinks(body string) string {
	return extractedImgRE.ReplaceAllString(body, "$1./images/")
}

func quote(s string) string {
	// Escape embedded double quotes and backslashes for the simple parser.
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
