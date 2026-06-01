package buildcmd

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/engine"
)

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestParseFormats(t *testing.T) {
	got, err := ParseFormats("pdf, html ,md")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %v", got)
	}
}

func TestParseFormatsDedup(t *testing.T) {
	got, err := ParseFormats("pdf,pdf,pdf")
	if err != nil || len(got) != 1 || got[0] != engine.FormatPDF {
		t.Errorf("dedup failed: %v %v", got, err)
	}
}

func TestParseFormatsUnknown(t *testing.T) {
	if _, err := ParseFormats("pdf,docx"); err == nil {
		t.Error("expected unknown-format error")
	}
}

func TestParseFormatsEmpty(t *testing.T) {
	if _, err := ParseFormats(" , "); err == nil {
		t.Error("expected empty-selection error")
	}
}

func TestRunBadFlag(t *testing.T) {
	if err := Run([]string{"-nope"}, nil); err == nil {
		t.Error("expected flag parse error")
	}
}

func TestRunDelegatesToBuild(t *testing.T) {
	// Empty articles dir => build.Run returns a "no articles" error before any
	// external tool is invoked. Exercises the delegation path in Run.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "articles"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Run([]string{"-articles-root", root, "-out", t.TempDir(), "-formats", "md"}, quietLogger())
	if err == nil {
		t.Error("expected no-articles error from build.Run")
	}
}
