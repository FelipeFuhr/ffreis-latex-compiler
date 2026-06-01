package cli

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRunHelpAndUnknown(t *testing.T) {
	// help => 0
	if code := run("prog", []string{"help"}, quiet()); code != 0 {
		t.Errorf("help exit = %d, want 0", code)
	}
	// no args => 1
	if code := run("prog", nil, quiet()); code != 1 {
		t.Errorf("no-args exit = %d, want 1", code)
	}
	// unknown => 1
	if code := run("prog", []string{"frobnicate"}, quiet()); code != 1 {
		t.Errorf("unknown exit = %d, want 1", code)
	}
}

func TestRunCommandError(t *testing.T) {
	// validate against a non-existent articles root for a specific slug errors => 1
	if code := run("prog", []string{"validate", "-slug", "ghost", "-articles-root", t.TempDir()}, quiet()); code != 1 {
		t.Errorf("expected exit 1 on command error, got %d", code)
	}
}

// captureStdout runs fn with os.Stdout redirected and returns what it wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestPrintUsage(t *testing.T) {
	out := captureStdout(t, func() { printUsage("ffreis-latex-compiler") })
	for _, want := range []string{"build", "validate", "promote", "doctor", "Usage:"} {
		if !strings.Contains(out, want) {
			t.Errorf("usage missing %q:\n%s", want, out)
		}
	}
}
