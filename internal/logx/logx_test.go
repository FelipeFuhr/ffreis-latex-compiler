package logx

import (
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	if New("test") == nil {
		t.Fatal("New returned nil")
	}
}

func TestLevelFromEnv(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
		"":      slog.LevelInfo,
		"other": slog.LevelInfo,
	}
	for val, want := range cases {
		t.Setenv("LATEX_COMPILER_LOG_LEVEL", val)
		if got := levelFromEnv(); got != want {
			t.Errorf("level(%q) = %v, want %v", val, got, want)
		}
	}
}
