package posts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Result is the outcome of validating a single post (or article) source.
type Result struct {
	Slug     string
	Errors   []string
	Warnings []string
}

// OK reports whether the source passed without errors.
func (r Result) OK() bool { return len(r.Errors) == 0 }

func (r *Result) addErr(msg string) {
	if msg != "" {
		r.Errors = append(r.Errors, msg)
	}
}

func (r *Result) addWarn(msg string) {
	if msg != "" {
		r.Warnings = append(r.Warnings, msg)
	}
}

// Fields are the metadata values the shared rule set inspects. Dir enables
// file-existence checks (thumbnail, body images); leave it "" to skip them.
// Body enables Markdown body checks (image refs, raw HTML); leave it "" to skip
// (e.g. when validating LaTeX article sources rather than generated posts).
type Fields struct {
	Slug         string
	FrontSlug    string
	Title        string
	Date         string
	Summary      string
	Thumbnail    string
	CanonicalURL string
	Tags         []string
	Dir          string
	Body         string
}

// CheckFields applies the posts-repo rule set to metadata field values and
// returns errors and warnings. It is the single source of truth shared by the
// disk-facing Validate (generated posts) and the article validator.
func CheckFields(f Fields) (errs, warns []string) {
	var res Result
	res.addErr(checkTitle(f.Title))
	res.addErr(checkDate(f.Date))
	res.addErr(checkSlug(f.FrontSlug, f.Slug))
	res.addErr(checkThumbnail(f.Dir, f.Thumbnail))
	res.addErr(checkCanonical(f.CanonicalURL, f.Slug))
	res.addErr(checkTags(f.Tags))
	if f.Dir != "" && f.Body != "" {
		res.addErr(checkBodyImages(f.Dir, f.Body))
	}
	if f.Body != "" {
		res.addErr(checkRawHTML(f.Body))
	}
	res.addWarn(warnSummary(f.Summary))
	res.addWarn(warnCanonical(f.CanonicalURL))
	return res.Errors, res.Warnings
}

// postMeta is the subset of frontmatter the disk validator parses.
type postMeta struct {
	Title        string   `yaml:"title"`
	Date         string   `yaml:"date"`
	Slug         string   `yaml:"slug"`
	Summary      string   `yaml:"summary"`
	Thumbnail    string   `yaml:"thumbnail"`
	Tags         []string `yaml:"tags"`
	CanonicalURL string   `yaml:"canonical_url"`
}

// Validate checks postsDir/<slug>/index.md against the posts-repo rules
// (mirrors the posts repo's validate-posts.py). It never returns an error;
// problems are reported as Errors/Warnings on the Result.
func Validate(postsDir, slug string) Result {
	res := Result{Slug: slug}
	dir := filepath.Join(postsDir, slug)
	indexMD := filepath.Join(dir, "index.md")

	if !slugRE.MatchString(slug) {
		res.addErr(fmt.Sprintf("directory name must be lowercase letters, numbers, and hyphens (got %q)", slug))
	}

	raw, err := os.ReadFile(indexMD) //nolint:gosec // indexMD is under a trusted posts dir
	if err != nil {
		res.addErr("index.md not found")
		return res
	}

	meta, body, ok := splitFrontmatter(string(raw))
	if !ok {
		res.addErr("no YAML frontmatter found — file must start with ---")
		return res
	}
	var m postMeta
	if err := yaml.Unmarshal([]byte(meta), &m); err != nil {
		res.addErr(fmt.Sprintf("invalid frontmatter YAML: %v", err))
		return res
	}

	errs, warns := CheckFields(Fields{
		Slug:         slug,
		FrontSlug:    m.Slug,
		Title:        m.Title,
		Date:         m.Date,
		Summary:      m.Summary,
		Thumbnail:    m.Thumbnail,
		CanonicalURL: m.CanonicalURL,
		Tags:         m.Tags,
		Dir:          dir,
		Body:         body,
	})
	res.Errors = append(res.Errors, errs...)
	res.Warnings = append(res.Warnings, warns...)
	return res
}

func checkTitle(title string) string {
	switch {
	case title == "":
		return "'title' is required and must be a non-empty string"
	case len(title) > MaxTitle:
		return fmt.Sprintf("'title' is %d chars; Medium's limit is %d", len(title), MaxTitle)
	}
	return ""
}

func checkDate(date string) string {
	switch {
	case date == "":
		return "'date' is required (format: YYYY-MM-DD)"
	case !dateRE.MatchString(date):
		return fmt.Sprintf("'date' must be YYYY-MM-DD, got %q", date)
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Sprintf("'date' is not a real calendar date: %q", date)
	}
	return ""
}

func checkSlug(frontSlug, slug string) string {
	if frontSlug != "" && frontSlug != slug {
		return fmt.Sprintf("frontmatter 'slug' (%q) does not match directory name (%q)", frontSlug, slug)
	}
	return ""
}

// checkThumbnail validates the thumbnail path shape always, and its existence
// only when dir is non-empty.
func checkThumbnail(dir, thumb string) string {
	if thumb == "" {
		return ""
	}
	if !strings.HasPrefix(thumb, "./images/") {
		return fmt.Sprintf("'thumbnail' must be a relative path like './images/<file>', got %q", thumb)
	}
	if dir != "" && !fileExists(filepath.Join(dir, thumb[2:])) {
		return fmt.Sprintf("thumbnail file not found: %s", filepath.Join(dir, thumb[2:]))
	}
	return ""
}

func checkBodyImages(dir, body string) string {
	for _, match := range imgBodyRE.FindAllStringSubmatch(body, -1) {
		ref := match[1] // ./images/<file>
		if !fileExists(filepath.Join(dir, ref[2:])) {
			return fmt.Sprintf("body image not found: %s (referenced as %q)", filepath.Join(dir, ref[2:]), ref)
		}
	}
	return ""
}

func checkCanonical(canonical, slug string) string {
	expected := CanonicalBase + slug + "/"
	if canonical != "" && canonical != expected {
		return fmt.Sprintf("'canonical_url' must be %q, got %q", expected, canonical)
	}
	return ""
}

func checkTags(tags []string) string {
	if len(tags) > MaxTags {
		return fmt.Sprintf("'tags' has %d items; Medium's limit is %d", len(tags), MaxTags)
	}
	return ""
}

func checkRawHTML(body string) string {
	if loc := rawHTMLRE.FindStringSubmatch(body); loc != nil {
		return fmt.Sprintf("raw HTML <%s> found in body — Medium strips unsupported HTML silently; use Markdown instead", loc[1])
	}
	return ""
}

func warnSummary(summary string) string {
	if summary == "" {
		return "'summary' not set — will be blank in the blog list, RSS excerpt, and social preview cards"
	}
	return ""
}

func warnCanonical(canonical string) string {
	if canonical == "" {
		return "'canonical_url' not set — will default to the ffreis.com URL but explicit is better for RSS and Medium import"
	}
	return ""
}

// splitFrontmatter separates a leading "---" YAML block from the body. The
// opening fence must be the first line; the block ends at the next line that is
// exactly "---" (trailing whitespace allowed). Returns (meta, body, true) when
// frontmatter is present.
func splitFrontmatter(raw string) (meta, body string, ok bool) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		return "", raw, false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == "---" {
			meta = strings.Join(lines[1:i], "\n")
			body = strings.Join(lines[i+1:], "\n")
			return meta, body, true
		}
	}
	return "", raw, false
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
