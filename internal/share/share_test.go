package share

import (
	"path/filepath"
	"testing"

	"github.com/gh-xj/omnicontext/internal/store"
)

func newShareStore(t *testing.T) (*store.Store, string) {
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
	return st, d
}

func TestExportImportRoundTrip(t *testing.T) {
	st, dir := newShareStore(t)
	defer st.Close()

	_, err := st.InsertImportedSession(store.SessionInput{
		SessionID:      "codex-test-1",
		SessionType:    "codex",
		SessionPath:    filepath.Join(dir, "codex1.jsonl"),
		WorkspacePath:  dir,
		StartedAt:      "2026-01-01T00:00:00Z",
		LastActivityAt: "2026-01-01T00:00:02Z",
		SessionTitle:   "codex test",
		SessionSummary: "summary",
		Metadata:       "{}",
	}, []store.TurnInput{{UserMessage: "u", AssistantSummary: "a", Timestamp: "2026-01-01T00:00:01Z"}})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	pack := filepath.Join(dir, "ctx.ocxpack")
	if err := ExportContext(st, "default", pack); err != nil {
		t.Fatalf("export: %v", err)
	}

	st2, _ := newShareStore(t)
	defer st2.Close()
	ctxID, n, err := ImportContext(st2, pack)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if ctxID == "" || n < 1 {
		t.Fatalf("unexpected import result: ctx=%q n=%d", ctxID, n)
	}
}
