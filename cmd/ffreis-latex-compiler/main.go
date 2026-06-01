// Command ffreis-latex-compiler compiles LaTeX articles (from a
// ffreis-articles-shaped repo, using shared fragments from ffreis-snippets)
// into PDF, HTML, and Medium-safe Markdown for ffreis-posts.
package main

import "github.com/FelipeFuhr/ffreis-latex-compiler/internal/cli"

func main() {
	cli.Run("ffreis-latex-compiler")
}
