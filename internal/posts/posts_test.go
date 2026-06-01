package posts

import (
	"strings"
	"testing"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/article"
)

func TestBuildFrontmatterDefaultsCanonical(t *testing.T) {
	fm := BuildFrontmatter(article.Meta{Title: "T", Date: "2026-01-02"}, "my-slug")
	if fm.CanonicalURL != CanonicalBase+"my-slug/" {
		t.Errorf("canonical = %q, want default", fm.CanonicalURL)
	}
	if fm.Slug != "my-slug" {
		t.Errorf("slug = %q", fm.Slug)
	}
}

func TestBuildFrontmatterKeepsExplicitCanonical(t *testing.T) {
	fm := BuildFrontmatter(article.Meta{CanonicalURL: "https://x/y/"}, "s")
	if fm.CanonicalURL != "https://x/y/" {
		t.Errorf("canonical overwritten: %q", fm.CanonicalURL)
	}
}

func TestRenderFrontmatterShape(t *testing.T) {
	fm := Frontmatter{
		Title:        `He said "hi"`,
		Date:         "2026-01-02",
		Slug:         "s",
		Summary:      "sum",
		Thumbnail:    "./images/t.png",
		Tags:         []string{"a", "b"},
		CanonicalURL: "https://ffreis.com/blog/s/",
	}
	out := fm.Render()
	wantContains := []string{
		`title: "He said \"hi\""`,
		`date: "2026-01-02"`,
		`tags: ["a", "b"]`,
		`thumbnail: "./images/t.png"`,
		`medium_published: false`,
		"---\n",
	}
	for _, w := range wantContains {
		if !strings.Contains(out, w) {
			t.Errorf("render missing %q in:\n%s", w, out)
		}
	}
}

func TestRenderFrontmatterOmitsEmptyOptionalFields(t *testing.T) {
	fm := Frontmatter{Title: "t", Date: "d", Slug: "s", CanonicalURL: "c"}
	out := fm.Render()
	if strings.Contains(out, "summary:") || strings.Contains(out, "tags:") || strings.Contains(out, "thumbnail:") {
		t.Errorf("unexpected optional field present:\n%s", out)
	}
}

func TestAssembleIndexMD(t *testing.T) {
	fm := Frontmatter{Title: "t", Date: "d", Slug: "s", CanonicalURL: "c"}
	got := AssembleIndexMD(fm, "\n\nHello body")
	if !strings.HasPrefix(got, "---\n") {
		t.Errorf("missing leading fence: %q", got)
	}
	if !strings.HasSuffix(got, "Hello body") {
		t.Errorf("body not appended: %q", got)
	}
}

func TestNormalizeImageLinks(t *testing.T) {
	cases := map[string]string{
		"![a](images/x.png)":   "![a](./images/x.png)",
		"![a](/images/x.png)":  "![a](./images/x.png)",
		"![a](./images/x.png)": "![a](./images/x.png)",
		"![a](other/x.png)":    "![a](other/x.png)",
	}
	for in, want := range cases {
		if got := NormalizeImageLinks(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
