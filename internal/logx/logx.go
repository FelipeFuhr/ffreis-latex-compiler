// Package logx provides a thin structured-logging wrapper around log/slog,
// mirroring the convention used by ffreis-website-compiler so every command in
// the fleet logs the same way (text handler to stderr, tagged with the program
// name).
package logx

import (
	"log/slog"
	"os"
)

// New returns a *slog.Logger writing human-readable text to stderr, tagged with
// the given program name. The level honours the LATEX_COMPILER_LOG_LEVEL env
// var (debug|info|warn|error); it defaults to info.
func New(name string) *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: levelFromEnv(),
	})
	return slog.New(handler).With("app", name)
}

func levelFromEnv() slog.Level {
	switch os.Getenv("LATEX_COMPILER_LOG_LEVEL") {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "warn", "WARN":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
