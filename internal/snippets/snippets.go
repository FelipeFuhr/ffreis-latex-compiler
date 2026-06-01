// Package snippets models the ffreis-snippets repository: a library of reusable
// LaTeX fragments (preambles, classes, macros, bib, figures) that articles pull
// in by name. It produces the TEXINPUTS/BIBINPUTS search paths the TeX engines
// need, and resolves local \input/\include references for validation.
package snippets

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Standard top-level directories inside a ffreis-snippets repo.
var (
	TeXDirs = []string{"preambles", "classes", "macros", "figures"}
	BibDirs = []string{"bib"}
)

// Repo is a ffreis-snippets checkout rooted at Root. A zero/empty Root means
// "no snippets repo configured" — TexInputs/BibInputs then contribute nothing
// beyond the caller-supplied extras.
type Repo struct {
	Root string
}

// texPathListSep is the separator TeX uses inside TEXINPUTS/BIBINPUTS. It is ':'
// on POSIX; the toolchain runs on Linux (native or container) so ':' is correct.
const texPathListSep = ":"

// TexInputs builds a TEXINPUTS value: each extra dir first (recursive), then the
// snippets repo root (recursive via the trailing '//'), then a trailing empty
// entry so the engine still searches its built-in/default trees.
func (r Repo) TexInputs(extra ...string) string {
	return r.pathList(r.Root, extra)
}

// BibInputs builds a BIBINPUTS value covering the snippets bib/ directory plus
// any extra dirs (recursive), with a trailing empty entry for defaults.
func (r Repo) BibInputs(extra ...string) string {
	var root string
	if r.Root != "" {
		root = filepath.Join(r.Root, "bib")
	}
	return r.pathList(root, extra)
}

func (r Repo) pathList(root string, extra []string) string {
	parts := make([]string, 0, len(extra)+2)
	for _, e := range extra {
		if e != "" {
			parts = append(parts, recursive(e))
		}
	}
	if root != "" {
		parts = append(parts, recursive(root))
	}
	// Trailing empty entry => append the engine's default search path.
	parts = append(parts, "")
	return strings.Join(parts, texPathListSep)
}

// recursive appends TeX's '//' recursive-search marker to a directory.
func recursive(dir string) string {
	return strings.TrimRight(dir, string(os.PathSeparator)) + "//"
}

// inputRefRE matches \input{name}, \include{name}, and \subfile{name}.
var inputRefRE = regexp.MustCompile(`\\(?:input|include|subfile)\{([^}]+)\}`)

// LocalRefs extracts the targets of \input/\include/\subfile from LaTeX source.
// Targets are returned verbatim (without the implicit .tex extension TeX adds).
func LocalRefs(texSource string) []string {
	matches := inputRefRE.FindAllStringSubmatch(texSource, -1)
	refs := make([]string, 0, len(matches))
	for _, m := range matches {
		refs = append(refs, strings.TrimSpace(m[1]))
	}
	return refs
}

// Resolve searches the snippets repo and the given local dirs (e.g. the article
// directory) for a referenced \input target, trying both the literal name and
// name+".tex". It returns the resolved absolute path and true on success.
func (r Repo) Resolve(ref string, localDirs ...string) (string, bool) {
	candidates := []string{ref}
	if filepath.Ext(ref) == "" {
		candidates = append(candidates, ref+".tex")
	}

	var roots []string
	roots = append(roots, localDirs...)
	if r.Root != "" {
		for _, d := range TeXDirs {
			roots = append(roots, filepath.Join(r.Root, d))
		}
		roots = append(roots, r.Root)
	}

	for _, root := range roots {
		for _, c := range candidates {
			// Direct join (handles refs like "macros/math").
			if p := filepath.Join(root, c); fileExists(p) {
				return p, true
			}
		}
	}
	return "", false
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
