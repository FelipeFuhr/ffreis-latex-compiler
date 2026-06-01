// Package snippets models the snippets repository: a library of reusable
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

// Standard top-level directories inside a snippets repo.
var (
	texSubdirs = []string{"preambles", "classes", "macros", "figures"}
	bibSubdir  = "bib"
)

// Repo is a snippets-repo checkout rooted at Root. A zero/empty Root means
// "no snippets repo configured" — TexDirs/BibDirs then contribute nothing
// beyond the caller-supplied extras.
type Repo struct {
	Root string
}

// TexDirs returns the raw directories to add to the TeX search path: the extra
// dirs (e.g. the article directory) first, then the snippets root and each of
// its standard subdirectories. Returning raw dirs (rather than a TEXINPUTS
// string) lets each engine adapter format them its own way — tectonic ignores
// TEXINPUTS and needs `-Z search-path=<dir>`, while make4ht/TeX Live wants the
// env var. Enumerating the subdirs covers both `\input{preambles/x}` (resolved
// against Root) and `\usepackage{cls-in-classes}` (resolved against Root/classes).
func (r Repo) TexDirs(extra ...string) []string {
	dirs := make([]string, 0, len(extra)+len(texSubdirs)+1)
	dirs = appendNonEmpty(dirs, extra...)
	if r.Root != "" {
		dirs = append(dirs, r.Root)
		for _, d := range texSubdirs {
			dirs = append(dirs, filepath.Join(r.Root, d))
		}
	}
	return dirs
}

// BibDirs returns the raw directories to add to the BibTeX search path.
func (r Repo) BibDirs(extra ...string) []string {
	dirs := make([]string, 0, len(extra)+1)
	dirs = appendNonEmpty(dirs, extra...)
	if r.Root != "" {
		dirs = append(dirs, filepath.Join(r.Root, bibSubdir))
	}
	return dirs
}

func appendNonEmpty(dst []string, vals ...string) []string {
	for _, v := range vals {
		if v != "" {
			dst = append(dst, v)
		}
	}
	return dst
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
		for _, d := range texSubdirs {
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
