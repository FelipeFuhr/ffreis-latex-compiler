// Package engine is the ports-and-adapters layer over the external LaTeX
// toolchain. Each output format sits behind a Renderer interface; concrete
// adapters shell out to tectonic (PDF), make4ht/tex4ht (HTML), and pandoc
// (Medium-safe Markdown). A Noop renderer stands in when a format is disabled,
// and any adapter whose tool is absent fails loudly (surfaced by `doctor`)
// rather than silently producing nothing.
package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Format identifies an output target.
type Format string

const (
	FormatPDF      Format = "pdf"
	FormatHTML     Format = "html"
	FormatMarkdown Format = "md"
)

// Job describes a single article render request. WorkDir is the article source
// directory (commands run there so relative \input/\graphicspath resolve);
// OutDir is the per-slug output directory under dist/.
type Job struct {
	SourceTeX string   // absolute path to main.tex
	WorkDir   string   // directory to run the tool in (article dir)
	OutDir    string   // output directory (dist/<slug>)
	Slug      string   // canonical slug; output artifacts are named <slug>.<ext>
	TexDirs   []string // directories to add to the TeX search path
	BibDirs   []string // directories to add to the BibTeX search path
}

// Renderer turns a Job into an artifact of a single Format.
type Renderer interface {
	Format() Format
	Tool() string    // underlying binary name ("" for Noop)
	Available() bool // whether the tool is invokable
	Render(context.Context, Job) (string, error)
}

// Runner executes an external command in dir with the given extra environment
// (appended to the current process env). It is injected so adapters can be
// unit-tested without the real toolchain installed.
type Runner func(ctx context.Context, dir string, env []string, name string, args ...string) error

// ExecRunner is the production Runner: it runs the command and streams its
// stderr through for diagnostics.
func ExecRunner(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // invoking the configured LaTeX toolchain binary is the purpose of this adapter
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stderr // tool chatter goes to stderr; stdout stays clean
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// toolAvailable reports whether name resolves on PATH.
func toolAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// jobEnv assembles the TEXINPUTS/BIBINPUTS environment slice for a Job. This is
// honoured by TeX Live engines (make4ht); tectonic ignores it and instead uses
// the `-Z search-path` flags built from the same directories.
func jobEnv(j Job) []string {
	var env []string
	if e := texPathEnv("TEXINPUTS", j.TexDirs); e != "" {
		env = append(env, e)
	}
	if e := texPathEnv("BIBINPUTS", j.BibDirs); e != "" {
		env = append(env, e)
	}
	return env
}

// texPathEnv builds a "VAR=<dir>//:<dir>//:" value: each directory is marked
// recursive ('//') and a trailing empty entry preserves the engine's defaults.
func texPathEnv(name string, dirs []string) string {
	if len(dirs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(dirs)+1)
	for _, d := range dirs {
		parts = append(parts, strings.TrimRight(d, "/")+"//")
	}
	parts = append(parts, "") // trailing empty => append the engine's default path
	return name + "=" + strings.Join(parts, ":")
}
