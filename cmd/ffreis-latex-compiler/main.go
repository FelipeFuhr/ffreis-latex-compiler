// Command ffreis-latex-compiler compiles LaTeX articles (from an
// articles-style source repo, using shared fragments from a snippets repo)
// into PDF, HTML, and Medium-safe Markdown for a blog/posts repo.
package main

import "github.com/FelipeFuhr/ffreis-latex-compiler/internal/cli"

func main() {
	cli.Run("ffreis-latex-compiler")
}
