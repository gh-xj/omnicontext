package cli

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gh-xj/omnicontext/internal/adapters"
	"github.com/gh-xj/omnicontext/internal/store"
)

func newCLIStore(t *testing.T) *store.Store {
	t.Helper()
	d := t.TempDir()
	st, err := store.Open(d)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := st.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := st.EnsureDefaultContext(); err != nil {
		t.Fatalf("default context: %v", err)
	}
	return st
}

func TestImportFromPathDedupe(t *testing.T) {
	st := newCLIStore(t)
	defer st.Close()

	fixtureDir := filepath.Join("..", "adapters", "testdata")
	inserted, parsed, skipped, err := importFromPath(st, "claude", fixtureDir)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if parsed < 1 || inserted < 1 {
		t.Fatalf("unexpected first import result: inserted=%d parsed=%d skipped=%d", inserted, parsed, skipped)
	}

	inserted2, parsed2, skipped2, err := importFromPath(st, "claude", fixtureDir)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if parsed2 < 1 || inserted2 != 0 || skipped2 < 1 {
		t.Fatalf("expected dedupe on second import: inserted=%d parsed=%d skipped=%d", inserted2, parsed2, skipped2)
	}
}

func TestImportFromPathDryRunDoesNotWrite(t *testing.T) {
	st := newCLIStore(t)
	defer st.Close()

	fixtureDir := filepath.Join("..", "adapters", "testdata")
	inserted, parsed, skipped, err := importFromPathWithOptions(st, "codex", fixtureDir, importOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run import: %v", err)
	}
	if parsed < 1 || inserted < 1 {
		t.Fatalf("unexpected dry-run result: inserted=%d parsed=%d skipped=%d", inserted, parsed, skipped)
	}
	n, err := st.CountSessions()
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if n != 0 {
		t.Fatalf("dry-run should not write sessions, got=%d", n)
	}
}

func TestFilterSessionsForImportMaxAndSince(t *testing.T) {
	s1 := adapters.Session{SessionID: "s1", LastActivityAt: "2026-01-01T00:00:00Z"}
	s2 := adapters.Session{SessionID: "s2", LastActivityAt: "2026-02-01T00:00:00Z"}
	s3 := adapters.Session{SessionID: "s3", LastActivityAt: "2026-03-01T00:00:00Z"}
	since := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	out := filterSessionsForImport([]adapters.Session{s1, s2, s3}, importOptions{
		Since:       &since,
		MaxSessions: 1,
	})
	if len(out) != 1 {
		t.Fatalf("len(out) = %d", len(out))
	}
	if out[0].SessionID != "s3" {
		t.Fatalf("expected most recent s3, got %s", out[0].SessionID)
	}
}

func TestParseSinceDate(t *testing.T) {
	if _, err := parseSinceDate("2026-02-01"); err != nil {
		t.Fatalf("expected valid date: %v", err)
	}
	if _, err := parseSinceDate("2026/02/01"); err == nil {
		t.Fatalf("expected invalid date error")
	}
}

func TestLabRunInspectorFlagRegistered(t *testing.T) {
	root := NewRootCmd()
	cmd, _, err := root.Find([]string{"lab", "run"})
	if err != nil {
		t.Fatalf("find lab run: %v", err)
	}
	if cmd.Flags().Lookup("inspector") == nil {
		t.Fatalf("expected --inspector flag on lab run")
	}
}

func TestEvolveRunInspectorFlagRegistered(t *testing.T) {
	root := NewRootCmd()
	cmd, _, err := root.Find([]string{"evolve", "run"})
	if err != nil {
		t.Fatalf("find evolve run: %v", err)
	}
	if cmd.Flags().Lookup("inspector") == nil {
		t.Fatalf("expected --inspector flag on evolve run")
	}
}

func TestLabRunRequiresInspectorCommand(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"lab", "run", "--goal", "g", "--verify", "exit 0"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected missing inspector error")
	}
	if !strings.Contains(err.Error(), "inspector command is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvolveRunRequiresInspectorFlag(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"evolve", "run", "--goal", "g"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected missing inspector error")
	}
	if !strings.Contains(err.Error(), "--inspector is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionCommandRegistered(t *testing.T) {
	root := NewRootCmd()
	cmd, _, err := root.Find([]string{"version"})
	if err != nil {
		t.Fatalf("find version: %v", err)
	}
	if cmd == nil || cmd.Name() != "version" {
		t.Fatalf("expected version command")
	}
}

func TestRootPersistentRuntimeFlagsRegistered(t *testing.T) {
	root := NewRootCmd()
	flags := []string{"verbose", "config", "no-color"}
	for _, name := range flags {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Fatalf("expected persistent flag %q", name)
		}
	}
}
