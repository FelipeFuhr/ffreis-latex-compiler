package doctorcmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestCheckAllPresent(t *testing.T) {
	old := lookPath
	defer func() { lookPath = old }()
	lookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }

	var buf bytes.Buffer
	if err := check(&buf, true); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.Contains(buf.String(), "All tools present") {
		t.Errorf("output: %s", buf.String())
	}
}

func TestCheckMissingStrict(t *testing.T) {
	old := lookPath
	defer func() { lookPath = old }()
	lookPath = func(name string) (string, error) {
		if name == "pandoc" {
			return "/usr/bin/pandoc", nil
		}
		return "", errors.New("not found")
	}

	var buf bytes.Buffer
	err := check(&buf, true)
	if err == nil {
		t.Fatal("expected strict error for missing tools")
	}
	if !strings.Contains(buf.String(), "MISSING") {
		t.Errorf("output: %s", buf.String())
	}
}

func TestCheckMissingNonStrict(t *testing.T) {
	old := lookPath
	defer func() { lookPath = old }()
	lookPath = func(string) (string, error) { return "", errors.New("nope") }

	var buf bytes.Buffer
	if err := check(&buf, false); err != nil {
		t.Errorf("non-strict should not error, got %v", err)
	}
}

func TestRunBadFlag(t *testing.T) {
	if err := Run([]string{"-bogus"}, nil); err == nil {
		t.Error("expected flag error")
	}
}

func TestRunNonStrict(t *testing.T) {
	if err := Run(nil, nil); err != nil {
		t.Errorf("non-strict Run should not error, got %v", err)
	}
}
