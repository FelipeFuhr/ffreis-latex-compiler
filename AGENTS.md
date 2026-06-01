# Agent Context

**This repo:** `ffreis-latex-compiler` — Go CLI that compiles LaTeX articles to
PDF, HTML, and Medium-safe Markdown. It is the LaTeX analog of
`ffreis-website-compiler` and follows the same conventions (handwritten CLI
dispatch on `os.Args[1]`, `log/slog` via `internal/logx`, ports-and-adapters).

## Public repo — private-repo hygiene

This is a **public** GitHub repository. When writing commit messages, PR titles,
PR descriptions, code comments, or any other user-visible text, **never name
private repos** — the article-source, snippets, or blog/posts repos that consume
this tool, or any internal infra. Use generic terms: "the articles repo", "the
snippets repo", "the posts/blog repo", "a private consumer". The compiler core is
deliberately generic: it takes `-articles-root`, `-snippets-root`, and
`-posts-dir` as flags and hardcodes no consumer.

## Inputs and outputs

The tool is consumer-agnostic. It expects:

- an **articles** root containing `articles/<slug>/{main.tex, meta.yaml, images/, refs.bib}`;
- an optional **snippets** root (reusable LaTeX fragments: `preambles/ classes/
  macros/ bib/ figures/`), added to `TEXINPUTS`/`BIBINPUTS` at compile time;
- for `promote`, a checkout of a **Markdown blog repo** (the `posts/<slug>/` layout).

`meta.yaml` is a superset of a typical blog frontmatter schema, so a compiled
article promotes into a blog without reshaping metadata.

## Toolchain (ports + adapters, `internal/engine`)

Each format sits behind a `Renderer`; adapters shell out via an injectable
`Runner` (so command/env construction is unit-tested without the binaries):

| Format | Tool | Notes |
|---|---|---|
| PDF | `tectonic` | auto-downloads CTAN packages on demand — the "behind the scenes" core |
| HTML | `make4ht` (tex4ht) | ships with TeX Live, which Tectonic deliberately omits |
| Markdown | `pandoc` | `--to=gfm-raw_html` ⇒ no raw HTML, Medium-safe |

**Search paths.** Tectonic ignores `TEXINPUTS` by design, so snippet directories are
passed to it as `-Z search-path=<dir>` (one per dir); make4ht/TeX Live gets the same dirs
via the `TEXINPUTS`/`BIBINPUTS` env. `engine.Job` therefore carries raw `TexDirs`/`BibDirs`
(from `snippets.Repo.TexDirs/BibDirs`), and each adapter formats them itself.

**These three do not coexist in one lightweight package**: Tectonic is a single
binary; tex4ht needs a TeX Live install. The full toolchain therefore lives in
`containers/Dockerfile.cli` (TeX Live base + tectonic + pandoc + the Go binary).
`make build` runs the compiler **inside that podman image**, so the host needs
nothing but a container runtime. `make doctor` reports native tool availability;
`make build-native` runs against host-installed tools.

## Commands

```bash
go run ./cmd/ffreis-latex-compiler build    -articles-root ../articles -snippets-root ../snippets -slug <slug> -formats pdf,html,md
go run ./cmd/ffreis-latex-compiler validate -articles-root ../articles -snippets-root ../snippets
go run ./cmd/ffreis-latex-compiler promote  -articles-root ../articles -out dist -posts-dir ../posts -slug <slug> [-open-pr|-dry-run]
go run ./cmd/ffreis-latex-compiler doctor
```

Output per article: `dist/<slug>/{<slug>.pdf, <slug>.html, index.md, images/}`.
The `index.md` + `images/` subtree is shaped exactly like a blog repo's
`posts/<slug>/` dir, so `promote` is a copy. `internal/posts` re-implements a
typical blog's `validate-posts.py` rules so a promoted post can't bounce in CI.

## Promotion is manual, never automatic

`promote` only ever stages a compiled post and (with `-open-pr`) opens a **PR** in
the target blog repo — it never auto-merges, and `openPullRequest` refuses to
commit onto `main`/`develop`.

- `make promote SLUG=… [OPEN_PR=1] [DRY_RUN=1]` (pure Go; needs only pandoc, not
  the TeX toolchain — promotion uses the Markdown output).
- **The CI glue that wires this to specific private repos lives in the private
  article-source repo, not here** (a `workflow_dispatch` workflow that checks out
  this public compiler + the private content repos and runs `build` + `promote
  -open-pr`). Keeping that out of this public repo is why no private repo names or
  fleet token secrets appear in this tree.

## Medium caveat

Markdown is best-effort for prose. Medium renders no LaTeX math or complex
floats — those stay high-fidelity only in PDF/HTML. The validator flags raw HTML
(`<div>` etc.), title >250 chars, >5 tags, missing/mismatched slug, and missing
images — the same gate the downstream blog repo enforces.

## Local toolchain note (sandbox)

The Go toolchain auto-downloaded by `GOTOOLCHAIN=auto` (module distribution) ships
without `covdata`, so `go test -race -coverprofile` over no-test packages errors
locally. `make coverage-gate` runs coverage **without** `-race` and works; real CI
(setup-go) has `covdata`. Use `GOTOOLCHAIN=go1.25.10` locally.

## Non-obvious facts

- **Test coverage minimum:** `COVERAGE_MIN=90` — enforced locally via
  `make coverage-gate` (pre-push) and in CI via `coverage.yml`.
- **Mutation testing:** Runs monthly against `./internal/...` with a
  `60%` efficacy threshold. Triggered by `mutation.yml`.
- **lefthook hooks** pull from `ffreis-platform-standards` (pinned SHA) and run
  `make quality-gates` on pre-push. Install with `make setup`.
- **Renovate** keeps Go module dependencies and GitHub Actions SHAs updated automatically.
  It extends `ffreis-platform-standards:renovate/go`.

## Structure

```
cmd/ffreis-latex-compiler/    ← CLI entry point
internal/               ← feature packages (tested, linted, covered)
scripts/hooks/          ← pre-commit and pre-push hook scripts
.github/workflows/      ← CI/CD workflows
```

## Build and run

```bash
make setup              # install lefthook git hooks
make test               # run unit tests
make coverage-gate      # run tests + enforce coverage minimum
make quality-gates      # full pre-push gate: test + race + coverage + govulncheck
make lint               # run golangci-lint
go run ./cmd/ffreis-latex-compiler --help
```

## Keeping this file current

- **If you discover a fact not reflected here:** add it before finishing your task.
- **If something here is wrong or outdated:** correct it in the same commit as the code change.
- **If you rename a file, command, or concept referenced here:** update the reference.
