package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Tectonic renders a Job to PDF via the tectonic engine, which auto-downloads
// the LaTeX packages each document needs on demand.
type Tectonic struct {
	run Runner
}

// NewTectonic returns a Tectonic renderer. A nil runner falls back to ExecRunner.
func NewTectonic(run Runner) *Tectonic {
	if run == nil {
		run = ExecRunner
	}
	return &Tectonic{run: run}
}

func (t *Tectonic) Format() Format  { return FormatPDF }
func (t *Tectonic) Tool() string    { return "tectonic" }
func (t *Tectonic) Available() bool { return toolAvailable(t.Tool()) }

// Render compiles main.tex and renames tectonic's basename-derived output to
// <slug>.pdf. Returns the absolute path to the produced PDF.
func (t *Tectonic) Render(ctx context.Context, j Job) (string, error) {
	if err := os.MkdirAll(j.OutDir, 0o750); err != nil {
		return "", fmt.Errorf("tectonic: mkdir outdir: %w", err)
	}
	args := []string{
		"-X", "compile",
		j.SourceTeX,
		"--outdir", j.OutDir,
		"--keep-logs",
	}
	// Tectonic ignores TEXINPUTS by design, so snippet directories are added via
	// its own `-Z search-path` flag (one per directory).
	for _, d := range j.TexDirs {
		args = append(args, "-Z", "search-path="+d)
	}
	for _, d := range j.BibDirs {
		args = append(args, "-Z", "search-path="+d)
	}
	if err := t.run(ctx, j.WorkDir, jobEnv(j), t.Tool(), args...); err != nil {
		return "", err
	}
	return renameToSlug(j, ".pdf")
}

// renameToSlug renames OutDir/<sourcebase><ext> to OutDir/<slug><ext> and
// returns the final path. If the source-base file is absent but the slug file
// already exists (some tools honour the slug name directly), it is accepted.
func renameToSlug(j Job, ext string) (string, error) {
	base := sourceBase(j.SourceTeX)
	produced := filepath.Join(j.OutDir, base+ext)
	final := filepath.Join(j.OutDir, j.Slug+ext)
	if produced == final {
		return final, nil
	}
	if _, err := os.Stat(produced); err != nil {
		if _, err2 := os.Stat(final); err2 == nil {
			return final, nil
		}
		return "", fmt.Errorf("expected output %s not found: %w", produced, err)
	}
	if err := os.Rename(produced, final); err != nil {
		return "", fmt.Errorf("rename %s -> %s: %w", produced, final, err)
	}
	return final, nil
}

func sourceBase(sourceTeX string) string {
	b := filepath.Base(sourceTeX)
	return b[:len(b)-len(filepath.Ext(b))]
}
