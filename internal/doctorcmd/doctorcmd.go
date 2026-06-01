// Package doctorcmd implements the `doctor` command: it reports whether the
// external toolchain (tectonic, make4ht, pandoc) is invokable, so users know
// whether to run natively or via the bundled container image.
package doctorcmd

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
)

// tool describes a required external binary and which output format it backs.
type tool struct {
	name    string
	format  string
	purpose string
}

var tools = []tool{
	{name: "tectonic", format: "pdf", purpose: "PDF (auto-downloads LaTeX packages)"},
	{name: "make4ht", format: "html", purpose: "HTML (tex4ht; ships with TeX Live)"},
	{name: "pandoc", format: "md", purpose: "Medium-safe Markdown for ffreis-posts"},
}

// lookPath is overridable in tests.
var lookPath = exec.LookPath

// Run executes the doctor command.
func Run(args []string, _ *slog.Logger) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	strict := fs.Bool("strict", false, "exit non-zero if any tool is missing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return check(os.Stdout, *strict)
}

func check(w io.Writer, strict bool) error {
	fmt.Fprintln(w, "ffreis-latex-compiler toolchain:")
	missing := 0
	for _, t := range tools {
		path, err := lookPath(t.name)
		if err != nil {
			missing++
			fmt.Fprintf(w, "  ✗  %-9s MISSING — %s\n", t.name, t.purpose)
			continue
		}
		fmt.Fprintf(w, "  ✓  %-9s %s — %s\n", t.name, path, t.purpose)
	}
	fmt.Fprintln(w)
	if missing > 0 {
		fmt.Fprintf(w, "%d tool(s) missing. Install them, or run the compiler via the bundled\n", missing)
		fmt.Fprintln(w, "container image (`make build` uses podman) where the full toolchain is present.")
		if strict {
			return fmt.Errorf("%d required tool(s) missing", missing)
		}
		return nil
	}
	fmt.Fprintln(w, "All tools present — native compilation is available.")
	return nil
}
