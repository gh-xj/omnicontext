package lab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunQualifies(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:          ws,
		Goal:               "create marker and pass verify",
		MaxIterations:      2,
		ImplementerCommand: "touch marker.txt && echo done > \"$OCX_LAB_IMPL_FILE\"",
		VerifyCommand:      "test -f marker.txt",
		JudgeCommand:       "echo QUALIFIED",
		Shell:              "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !r.Qualified {
		t.Fatalf("expected qualified")
	}
	if len(r.Iterations) < 1 {
		t.Fatalf("expected iterations")
	}
	if _, err := os.Stat(filepath.Join(r.RunDir, "report.json")); err != nil {
		t.Fatalf("report.json missing: %v", err)
	}
}

func TestRunNotQualified(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:     ws,
		Goal:          "never pass",
		MaxIterations: 2,
		VerifyCommand: "exit 1",
		Shell:         "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if r.Qualified {
		t.Fatalf("expected not qualified")
	}
	if len(r.Iterations) != 2 {
		t.Fatalf("expected 2 iterations, got %d", len(r.Iterations))
	}
}
