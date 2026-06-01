// Package promotecmd parses flags for the `promote` command. It stages a
// compiled article into a ffreis-posts checkout (via internal/promote) and,
// with -open-pr, branches + commits + opens a pull request. It never pushes to
// main and never auto-merges.
package promotecmd

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/promote"
)

// GitRunner runs a command in dir and returns combined output. Injected for
// tests; defaults to execGit.
type GitRunner func(ctx context.Context, dir, name string, args ...string) (string, error)

// Run executes the promote command.
func Run(args []string, logger *slog.Logger) error {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	articlesRoot := fs.String("articles-root", ".", "root containing articles/<slug>/")
	out := fs.String("out", "dist", "build output directory")
	postsDir := fs.String("posts-dir", "", "ffreis-posts checkout root (required)")
	slug := fs.String("slug", "", "article slug to promote (required)")
	openPR := fs.Bool("open-pr", false, "branch, commit, push, and open a PR in the posts repo")
	dryRun := fs.Bool("dry-run", false, "report what would happen without writing")
	branch := fs.String("branch", "", "branch name for the PR (default: promote/<post-slug>)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *slug == "" {
		return fmt.Errorf("-slug is required")
	}
	if *postsDir == "" {
		return fmt.Errorf("-posts-dir is required")
	}

	outcome, err := promote.Run(promote.Options{
		ArticlesRoot: *articlesRoot,
		OutRoot:      *out,
		PostsDir:     *postsDir,
		Slug:         *slug,
		DryRun:       *dryRun,
	})
	if outcome != nil {
		printOutcome(logger, outcome, *dryRun)
	}
	if err != nil {
		return err
	}

	if *dryRun || !*openPR {
		return nil
	}

	br := *branch
	if br == "" {
		br = "promote/" + outcome.PostSlug
	}
	return openPullRequest(context.Background(), logger, *postsDir, br, outcome.PostSlug, execGit)
}

func printOutcome(logger *slog.Logger, o *promote.Outcome, dryRun bool) {
	verb := "staged"
	if dryRun {
		verb = "would stage"
	}
	logger.Info(verb+" post", "post_slug", o.PostSlug, "target", o.TargetDir, "files", len(o.CopiedFiles))
	for _, w := range o.Validation.Warnings {
		logger.Warn("post warning", "msg", w)
	}
}

// openPullRequest creates a branch in the posts repo, commits the staged post,
// pushes, and opens a draft PR via gh. Guards against operating on main.
func openPullRequest(ctx context.Context, logger *slog.Logger, postsDir, branch, postSlug string, git GitRunner) error {
	if branch == "main" || branch == "develop" {
		return fmt.Errorf("refusing to commit promotion onto %q — use a feature branch", branch)
	}
	postPath := filepath.Join("posts", postSlug)

	steps := [][]string{
		{"git", "checkout", "-b", branch},
		{"git", "add", postPath},
		{"git", "commit", "-m", "post: add " + postSlug + " (promoted from ffreis-articles)"},
		{"git", "push", "-u", "origin", branch},
	}
	for _, s := range steps {
		out, err := git(ctx, postsDir, s[0], s[1:]...)
		if err != nil {
			return fmt.Errorf("%v: %w\n%s", s, err, out)
		}
	}

	title := fmt.Sprintf("post: %s", postSlug)
	const body = "Promoted from ffreis-articles by ffreis-latex-compiler.\n\nReview the rendered Markdown before merging. Auto-merge is intentionally disabled."
	out, err := git(ctx, postsDir, "gh", "pr", "create", "--draft", "--title", title, "--body", body)
	if err != nil {
		return fmt.Errorf("gh pr create: %w\n%s", err, out)
	}
	logger.Info("opened draft PR", "post_slug", postSlug, "branch", branch, "output", out)
	return nil
}

func execGit(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	b, err := cmd.CombinedOutput()
	return string(b), err
}
