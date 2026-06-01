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
	SourceTeX string // absolute path to main.tex
	WorkDir   string // directory to run the tool in (article dir)
	OutDir    string // output directory (dist/<slug>)
	Slug      string // canonical slug; output artifacts are named <slug>.<ext>
	TexInputs string // value for TEXINPUTS (may be empty)
	BibInputs string // value for BIBINPUTS (may be empty)
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
	cmd := exec.CommandContext(ctx, name, args...)
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

// jobEnv assembles the TEXINPUTS/BIBINPUTS environment slice for a Job.
func jobEnv(j Job) []string {
	var env []string
	if j.TexInputs != "" {
		env = append(env, "TEXINPUTS="+j.TexInputs)
	}
	if j.BibInputs != "" {
		env = append(env, "BIBINPUTS="+j.BibInputs)
	}
	return env
}
