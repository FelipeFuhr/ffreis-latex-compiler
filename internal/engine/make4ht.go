package engine

import (
	"context"
	"fmt"
	"os"
)

// Make4ht renders a Job to HTML via make4ht (the tex4ht build wrapper that
// ships with TeX Live). The generated CSS sits beside the HTML in OutDir.
type Make4ht struct {
	run Runner
}

// NewMake4ht returns a Make4ht renderer. A nil runner falls back to ExecRunner.
func NewMake4ht(run Runner) *Make4ht {
	if run == nil {
		run = ExecRunner
	}
	return &Make4ht{run: run}
}

func (m *Make4ht) Format() Format  { return FormatHTML }
func (m *Make4ht) Tool() string    { return "make4ht" }
func (m *Make4ht) Available() bool { return toolAvailable(m.Tool()) }

// Render builds <sourcebase>.html in OutDir, then renames it to <slug>.html.
// The companion <sourcebase>.css keeps its name and is still referenced by the
// renamed HTML, so no link rewriting is required.
func (m *Make4ht) Render(ctx context.Context, j Job) (string, error) {
	if err := os.MkdirAll(j.OutDir, 0o750); err != nil {
		return "", fmt.Errorf("make4ht: mkdir outdir: %w", err)
	}
	args := []string{
		"-d", j.OutDir, // output directory
		j.SourceTeX,
	}
	if err := m.run(ctx, j.WorkDir, jobEnv(j), m.Tool(), args...); err != nil {
		return "", err
	}
	return renameToSlug(j, ".html")
}
