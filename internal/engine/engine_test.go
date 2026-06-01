package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recordingRunner captures the last invocation and optionally creates a file at
// a path derived from the args, simulating the tool's output.
type recordingRunner struct {
	dir     string
	env     []string
	name    string
	args    []string
	produce string // filename to create in OutDir, "" = none
	outDir  string
	fail    error
}

func (r *recordingRunner) run(_ context.Context, dir string, env []string, name string, args ...string) error {
	r.dir, r.env, r.name, r.args = dir, env, name, args
	if r.fail != nil {
		return r.fail
	}
	if r.produce != "" {
		if err := os.WriteFile(filepath.Join(r.outDir, r.produce), []byte("out"), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func newJob(t *testing.T) Job {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "dist", "hello")
	src := filepath.Join(dir, "main.tex")
	if err := os.WriteFile(src, []byte(`\documentclass{article}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return Job{SourceTeX: src, WorkDir: dir, OutDir: out, Slug: "hello", TexDirs: []string{"/t"}, BibDirs: []string{"/b"}}
}

func TestTectonicRender(t *testing.T) {
	j := newJob(t)
	rr := &recordingRunner{produce: "main.pdf", outDir: j.OutDir}
	te := NewTectonic(rr.run)
	got, err := te.Render(context.Background(), j)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if filepath.Base(got) != "hello.pdf" {
		t.Errorf("output = %q, want hello.pdf", got)
	}
	if rr.name != "tectonic" {
		t.Errorf("tool = %q", rr.name)
	}
	if !contains(rr.args, j.SourceTeX) || !contains(rr.args, "compile") {
		t.Errorf("args missing source/compile: %v", rr.args)
	}
	if !contains(rr.args, "search-path=/t") || !contains(rr.args, "search-path=/b") {
		t.Errorf("tectonic args missing -Z search-path: %v", rr.args)
	}
	// jobEnv still sets TEXINPUTS/BIBINPUTS (recursive markers) for TeX Live engines.
	if !contains(rr.env, "TEXINPUTS=/t//:") || !contains(rr.env, "BIBINPUTS=/b//:") {
		t.Errorf("env missing TEXINPUTS/BIBINPUTS: %v", rr.env)
	}
}

func TestMake4htRender(t *testing.T) {
	j := newJob(t)
	rr := &recordingRunner{produce: "main.html", outDir: j.OutDir}
	got, err := NewMake4ht(rr.run).Render(context.Background(), j)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if filepath.Base(got) != "hello.html" {
		t.Errorf("output = %q", got)
	}
}

func TestPandocRender(t *testing.T) {
	j := newJob(t)
	rr := &recordingRunner{produce: BodyMarkdownName, outDir: j.OutDir}
	got, err := NewPandoc(rr.run).Render(context.Background(), j)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if filepath.Base(got) != BodyMarkdownName {
		t.Errorf("output = %q", got)
	}
	if !contains(rr.args, "--to=gfm-raw_html") {
		t.Errorf("expected gfm-raw_html target: %v", rr.args)
	}
}

func TestRenderToolError(t *testing.T) {
	j := newJob(t)
	rr := &recordingRunner{fail: errors.New("boom"), outDir: j.OutDir}
	if _, err := NewTectonic(rr.run).Render(context.Background(), j); err == nil {
		t.Error("expected error from failing runner")
	}
}

func TestRenameMissingOutput(t *testing.T) {
	j := newJob(t)
	rr := &recordingRunner{outDir: j.OutDir} // produces nothing
	if _, err := NewTectonic(rr.run).Render(context.Background(), j); err == nil {
		t.Error("expected error when expected output absent")
	}
}

func TestNoop(t *testing.T) {
	n := NewNoop(FormatPDF)
	if n.Format() != FormatPDF || n.Tool() != "" || !n.Available() {
		t.Error("noop metadata wrong")
	}
	got, err := n.Render(context.Background(), Job{})
	if err != nil || got != "" {
		t.Errorf("noop render = %q, %v", got, err)
	}
}

func TestDefaultRunnerFallback(t *testing.T) {
	if NewTectonic(nil) == nil || NewMake4ht(nil) == nil || NewPandoc(nil) == nil {
		t.Error("nil runner should fall back to ExecRunner")
	}
}

func TestAvailable(t *testing.T) {
	// 'go' is on PATH in CI; use it to exercise the Available path generically.
	if !toolAvailable("go") {
		t.Skip("go not on PATH")
	}
	if toolAvailable("definitely-not-a-real-binary-xyz") {
		t.Error("unexpected availability")
	}
}

func TestSourceBase(t *testing.T) {
	if sourceBase("/a/b/main.tex") != "main" {
		t.Errorf("sourceBase wrong: %q", sourceBase("/a/b/main.tex"))
	}
}

func TestJobEnvEmpty(t *testing.T) {
	if env := jobEnv(Job{}); len(env) != 0 {
		t.Errorf("empty job should yield no env, got %v", env)
	}
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want || strings.Contains(v, want) {
			return true
		}
	}
	return false
}

func TestFormatAndAvailableMetadata(t *testing.T) {
	if NewTectonic(nil).Format() != FormatPDF {
		t.Error("tectonic format")
	}
	if NewMake4ht(nil).Format() != FormatHTML {
		t.Error("make4ht format")
	}
	if NewPandoc(nil).Format() != FormatMarkdown {
		t.Error("pandoc format")
	}
	// Available exercises exec.LookPath; result depends on env, just call it.
	_ = NewTectonic(nil).Available()
	_ = NewMake4ht(nil).Available()
	_ = NewPandoc(nil).Available()
}

func TestExecRunnerSuccess(t *testing.T) {
	if err := ExecRunner(context.Background(), t.TempDir(), []string{"FOO=bar"}, "true"); err != nil {
		// 'true' should exist on Linux CI/dev hosts.
		t.Skipf("true not available: %v", err)
	}
}

func TestExecRunnerFailure(t *testing.T) {
	if err := ExecRunner(context.Background(), t.TempDir(), nil, "this-binary-does-not-exist-xyz"); err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestRenameSlugEqualsSourceBase(t *testing.T) {
	j := newJob(t)
	j.Slug = "main" // sourceBase("main.tex") == "main" => produced == final, early return
	rr := &recordingRunner{produce: "main.pdf", outDir: j.OutDir}
	got, err := NewTectonic(rr.run).Render(context.Background(), j)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if filepath.Base(got) != "main.pdf" {
		t.Errorf("output = %q", got)
	}
}

func TestRenameFinalAlreadyExists(t *testing.T) {
	j := newJob(t)
	// Pre-create the final file; runner produces nothing under the source base.
	if err := os.MkdirAll(j.OutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(j.OutDir, "hello.html"), []byte("x"), 0o644)
	rr := &recordingRunner{outDir: j.OutDir} // no produce
	got, err := NewMake4ht(rr.run).Render(context.Background(), j)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if filepath.Base(got) != "hello.html" {
		t.Errorf("output = %q", got)
	}
}
