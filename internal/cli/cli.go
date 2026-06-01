// Package cli is the command dispatcher for ffreis-latex-compiler. It mirrors
// the handwritten dispatch style of ffreis-website-compiler: read os.Args[1] as
// the command, hand the rest to the matching command package.
package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/buildcmd"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/doctorcmd"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/logx"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/promotecmd"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/validatecmd"
)

// Run dispatches the CLI and exits the process with the resulting status code.
func Run(programName string) {
	os.Exit(run(programName, os.Args[1:], logx.New(programName)))
}

// run is the testable core: it dispatches args and returns a process exit code
// instead of calling os.Exit, so it can be exercised in unit tests.
func run(programName string, args []string, logger *slog.Logger) int {
	if len(args) == 0 {
		printUsage(programName)
		return 1
	}

	cmd := args[0]
	rest := args[1:]

	var err error
	switch cmd {
	case "build", "compile":
		err = buildcmd.Run(rest, logger)
	case "validate":
		err = validatecmd.Run(rest, logger)
	case "promote":
		err = promotecmd.Run(rest, logger)
	case "doctor":
		err = doctorcmd.Run(rest, logger)
	case "help", "-h", "--help":
		printUsage(programName)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage(programName)
		return 1
	}

	if err != nil {
		logger.Error("command failed", "command", cmd, "error", err)
		return 1
	}
	return 0
}

func printUsage(programName string) {
	fmt.Printf(`%s — compile LaTeX articles to PDF, HTML, and Medium-safe Markdown.

Usage:
  %s <command> [flags]

Commands:
  build, compile   Compile article(s) to dist/<slug>/ (pdf, html, index.md)
  validate         Validate article sources + meta.yaml + snippet references
  promote          Stage a compiled article into a Markdown blog repo checkout (optionally open a PR)
  doctor           Report toolchain availability (tectonic, make4ht, pandoc)

Examples:
  %s build -articles-root ../articles -snippets-root ../snippets -slug hello-latex
  %s validate -articles-root ../articles
  %s promote -slug hello-latex -posts-dir ../posts -open-pr
  %s doctor
`, programName, programName, programName, programName, programName, programName)
}
