package posts

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/testutil"
)

func errStr(errs []string) string { return strings.Join(errs, "\n") }

func TestCheckFieldsValid(t *testing.T) {
	errs, warns := CheckFields(Fields{
		Slug: "s", Title: "Hello", Date: "2026-01-02",
		Summary: "x", CanonicalURL: CanonicalBase + "s/",
	})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
}

func TestCheckFieldsErrors(t *testing.T) {
	errs, _ := CheckFields(Fields{
		Slug:         "s",
		Title:        strings.Repeat("x", MaxTitle+1),
		Date:         "2026-13-99",
		FrontSlug:    "other",
		Tags:         []string{"1", "2", "3", "4", "5", "6"},
		CanonicalURL: "https://wrong/",
	})
	joined := errStr(errs)
	for _, want := range []string{"title", "date", "slug", "tags", "canonical_url"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q error in: %s", want, joined)
		}
	}
}

func TestCheckDateNotReal(t *testing.T) {
	errs, _ := CheckFields(Fields{Slug: "s", Title: "t", Date: "2026-02-30", Summary: "x", CanonicalURL: "c"})
	if !strings.Contains(errStr(errs), "not a real calendar date") {
		t.Errorf("expected calendar error, got %v", errs)
	}
}

func TestCheckFieldsBodyChecks(t *testing.T) {
	dir := t.TempDir()
	errs, _ := CheckFields(Fields{
		Slug: "s", Title: "t", Date: "2026-01-02", Summary: "x", CanonicalURL: "c",
		Dir:  dir,
		Body: "text ![](./images/missing.png) more <div>nope</div>",
	})
	joined := errStr(errs)
	if !strings.Contains(joined, "body image not found") {
		t.Errorf("missing body image error: %s", joined)
	}
	if !strings.Contains(joined, "raw HTML") {
		t.Errorf("missing raw HTML error: %s", joined)
	}
}

func TestCheckThumbnailShapeAndExistence(t *testing.T) {
	if got := checkThumbnail("", "cover.png"); !strings.Contains(got, "relative path") {
		t.Errorf("expected shape error, got %q", got)
	}
	dir := t.TempDir()
	if got := checkThumbnail(dir, "./images/x.png"); !strings.Contains(got, "not found") {
		t.Errorf("expected not-found, got %q", got)
	}
	testutil.WriteFile(t, dir, "images/x.png", "data")
	if got := checkThumbnail(dir, "./images/x.png"); got != "" {
		t.Errorf("expected ok, got %q", got)
	}
}

func TestValidateGeneratedPost(t *testing.T) {
	postsBase := t.TempDir()
	slug := "good-post"
	testutil.WriteFile(t, postsBase, filepath.Join(slug, "index.md"),
		"---\ntitle: \"Good\"\ndate: \"2026-01-02\"\nslug: \"good-post\"\nsummary: \"s\"\ncanonical_url: \""+CanonicalBase+"good-post/\"\n---\n\nBody text.\n")
	res := Validate(postsBase, slug)
	if !res.OK() {
		t.Fatalf("expected valid, got errors %v", res.Errors)
	}
}

func TestValidateMissingIndex(t *testing.T) {
	res := Validate(t.TempDir(), "nope")
	if res.OK() || !strings.Contains(errStr(res.Errors), "index.md not found") {
		t.Errorf("expected missing index error, got %v", res.Errors)
	}
}

func TestValidateBadSlugAndNoFrontmatter(t *testing.T) {
	postsBase := t.TempDir()
	slug := "Bad_Slug"
	testutil.WriteFile(t, postsBase, filepath.Join(slug, "index.md"), "no frontmatter here")
	res := Validate(postsBase, slug)
	joined := errStr(res.Errors)
	if !strings.Contains(joined, "lowercase") {
		t.Errorf("expected slug error: %s", joined)
	}
	if !strings.Contains(joined, "no YAML frontmatter") {
		t.Errorf("expected frontmatter error: %s", joined)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	meta, body, ok := splitFrontmatter("---\na: 1\n---\nhello\n")
	if !ok || strings.TrimSpace(meta) != "a: 1" || strings.TrimSpace(body) != "hello" {
		t.Errorf("split failed: meta=%q body=%q ok=%v", meta, body, ok)
	}
	if _, _, ok := splitFrontmatter("no fence"); ok {
		t.Error("expected ok=false without fence")
	}
	if _, _, ok := splitFrontmatter("---\nunterminated\n"); ok {
		t.Error("expected ok=false for unterminated frontmatter")
	}
}

func TestCheckFieldsEmptyRequiredAndWarnings(t *testing.T) {
	errs, warns := CheckFields(Fields{Slug: "s"}) // empty title, date, summary, canonical
	joined := errStr(errs)
	if !strings.Contains(joined, "'title' is required") {
		t.Errorf("missing empty-title error: %s", joined)
	}
	if !strings.Contains(joined, "'date' is required") {
		t.Errorf("missing empty-date error: %s", joined)
	}
	w := strings.Join(warns, "\n")
	if !strings.Contains(w, "summary") || !strings.Contains(w, "canonical_url") {
		t.Errorf("missing warnings: %s", w)
	}
}
