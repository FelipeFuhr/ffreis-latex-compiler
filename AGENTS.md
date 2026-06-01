# Agent Context

**This repo:** `ffreis-latex-compiler` — Go CLI that compiles LaTeX articles to
PDF, HTML, and Medium-safe Markdown. It is the LaTeX analog of
`ffreis-website-compiler` and follows the same conventions (handwritten CLI
dispatch on `os.Args[1]`, `log/slog` via `internal/logx`, ports-and-adapters).

## How it fits the fleet

- **`ffreis-articles`** — article **sources** (`articles/<slug>/main.tex` + `meta.yaml`).
- **`ffreis-snippets`** — reusable LaTeX fragments (preambles, classes, macros,
  bib, figures), added to `TEXINPUTS`/`BIBINPUTS` at compile time.
- **`ffreis-posts`** — finished blog Markdown. The compiler can **promote** a
  compiled article into it (manual only).

## Toolchain (ports + adapters, `internal/engine`)

Each format sits behind a `Renderer`; adapters shell out via an injectable
`Runner` (so command/env construction is unit-tested without the binaries):

| Format | Tool | Notes |
|---|---|---|
| PDF | `tectonic` | auto-downloads CTAN packages on demand — the "behind the scenes" core |
| HTML | `make4ht` (tex4ht) | ships with TeX Live, which Tectonic deliberately omits |
| Markdown | `pandoc` | `--to=gfm-raw_html` ⇒ no raw HTML, Medium-safe |

**These three do not coexist in one lightweight package**: Tectonic is a single
binary; tex4ht needs a TeX Live install. The full toolchain therefore lives in
`containers/Dockerfile.cli` (TeX Live base + tectonic + pandoc + the Go binary).
`make build` runs the compiler **inside that podman image**, so the host needs
nothing but a container runtime. `make doctor` reports native tool availability;
`make build-native` runs against host-installed tools.

## Commands

```bash
go run ./cmd/ffreis-latex-compiler build    -articles-root ../ffreis-articles -snippets-root ../ffreis-snippets -slug <slug> -formats pdf,html,md
go run ./cmd/ffreis-latex-compiler validate -articles-root ../ffreis-articles -snippets-root ../ffreis-snippets
go run ./cmd/ffreis-latex-compiler promote  -articles-root ../ffreis-articles -out dist -posts-dir ../ffreis-posts -slug <slug> [-open-pr|-dry-run]
go run ./cmd/ffreis-latex-compiler doctor
```

Output per article: `dist/<slug>/{<slug>.pdf, <slug>.html, index.md, images/}`.
The `index.md` + `images/` subtree is shaped exactly like a `ffreis-posts/posts/<slug>/`
dir, so `promote` is a copy. `internal/posts` re-implements the ffreis-posts
validation rules (`validate-posts.py`) so a promoted post can't bounce in CI.

## Promotion is manual, never automatic

- `make promote SLUG=… [OPEN_PR=1] [DRY_RUN=1]` (pure Go; needs only pandoc, not
  the TeX toolchain — promotion uses the Markdown output).
- `.github/workflows/promote-to-posts.yml` is **`workflow_dispatch`-only**: it
  compiles one slug's Markdown, opens a **draft PR** in `ffreis-posts`, and never
  merges. Needs secret `FLEET_CONTENT_PAT` (read on articles/snippets, PR-write
  on posts). `openPullRequest` refuses to commit onto `main`/`develop`.

## Medium caveat

Markdown is best-effort for prose. Medium renders no LaTeX math or complex
floats — those stay high-fidelity only in PDF/HTML. The validator flags raw HTML
(`<div>` etc.), title >250 chars, >5 tags, missing/mismatched slug, and missing
images — the same gate `ffreis-posts` enforces.

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
