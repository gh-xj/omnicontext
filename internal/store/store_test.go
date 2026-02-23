package store

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := st.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := st.EnsureDefaultContext(); err != nil {
		t.Fatalf("default context: %v", err)
	}
	return st, dir
}

func TestInsertImportedSessionWithTurns(t *testing.T) {
	st, dir := newTestStore(t)
	defer st.Close()

	sid, err := st.InsertImportedSession(SessionInput{
		SessionID:      "claude-test-1",
		SessionType:    "claude",
		SessionPath:    filepath.Join(dir, "s1.jsonl"),
		WorkspacePath:  dir,
		StartedAt:      "2026-01-01T00:00:00Z",
		LastActivityAt: "2026-01-01T00:00:10Z",
		SessionTitle:   "test",
		SessionSummary: "summary",
		Metadata:       "{}",
	}, []TurnInput{{UserMessage: "u1", AssistantSummary: "a1", Timestamp: "2026-01-01T00:00:01Z"}, {UserMessage: "u2", AssistantSummary: "a2", Timestamp: "2026-01-01T00:00:02Z"}})
	if err != nil {
		t.Fatalf("insert imported session: %v", err)
	}

	n, err := st.CountSessions()
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if n != 1 {
		t.Fatalf("sessions = %d", n)
	}
	turns, err := st.CountTurnsForSession(sid)
	if err != nil {
		t.Fatalf("count turns: %v", err)
	}
	if turns != 2 {
		t.Fatalf("turns = %d", turns)
	}
}
