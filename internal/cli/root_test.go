package cli

import (
	"path/filepath"
	"testing"

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
