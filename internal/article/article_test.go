package article

import (
	"strings"
	"testing"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/testutil"
)

const validMeta = "title: \"Hello\"\ndate: \"2026-01-02\"\nslug: \"hello\"\ntags: [a, b]\npost_slug: \"hello-post\"\n"

func TestLoadHappy(t *testing.T) {
	root := t.TempDir()
	testutil.Article(t, root, "hello", validMeta, `\documentclass{article}`)
	testutil.WriteFile(t, root, "articles/hello/images/fig.png", "x")

	a, err := Load(root, "hello")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if a.Meta.Title != "Hello" || a.Meta.Date != "2026-01-02" {
		t.Errorf("meta wrong: %+v", a.Meta)
	}
	if len(a.Meta.Tags) != 2 {
		t.Errorf("tags: %v", a.Meta.Tags)
	}
	if a.PostSlug() != "hello-post" {
		t.Errorf("post slug = %q", a.PostSlug())
	}
	if len(a.Images) != 1 || a.Images[0] != "fig.png" {
		t.Errorf("images = %v", a.Images)
	}
}

func TestPostSlugDefaultsToDir(t *testing.T) {
	root := t.TempDir()
	testutil.Article(t, root, "plain", "title: \"T\"\ndate: \"2026-01-02\"\n", "x")
	a, err := Load(root, "plain")
	if err != nil {
		t.Fatal(err)
	}
	if a.PostSlug() != "plain" {
		t.Errorf("post slug = %q, want plain", a.PostSlug())
	}
}

func TestLoadMissingMainTeX(t *testing.T) {
	root := t.TempDir()
	testutil.WriteFile(t, root, "articles/x/meta.yaml", validMeta)
	if _, err := Load(root, "x"); err == nil || !strings.Contains(err.Error(), "main.tex") {
		t.Errorf("expected main.tex error, got %v", err)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	root := t.TempDir()
	testutil.Article(t, root, "x", "title: \"unterminated\n: : :", "tex")
	if _, err := Load(root, "x"); err == nil || !strings.Contains(err.Error(), "invalid YAML") {
		t.Errorf("expected YAML error, got %v", err)
	}
}

func TestLoadMissingDir(t *testing.T) {
	if _, err := Load(t.TempDir(), "ghost"); err == nil {
		t.Error("expected error for missing article")
	}
}

func TestLoadAll(t *testing.T) {
	root := t.TempDir()
	testutil.Article(t, root, "b-second", validMeta, "x")
	testutil.Article(t, root, "a-first", validMeta, "x")
	all, err := LoadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Slug != "a-first" || all[1].Slug != "b-second" {
		t.Errorf("LoadAll order/count wrong: %v", all)
	}
}

func TestLoadAllMissingDir(t *testing.T) {
	if _, err := LoadAll(t.TempDir()); err == nil {
		t.Error("expected error when articles/ absent")
	}
}
