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

func TestSessionExistsByPath(t *testing.T) {
	st, dir := newTestStore(t)
	defer st.Close()
	path := filepath.Join(dir, "s2.jsonl")
	if _, err := st.InsertImportedSession(SessionInput{
		SessionID:   "codex-test-2",
		SessionType: "codex",
		SessionPath: path,
	}, nil); err != nil {
		t.Fatalf("insert: %v", err)
	}
	exists, err := st.SessionExistsByPath(path)
	if err != nil {
		t.Fatalf("exists err: %v", err)
	}
	if !exists {
		t.Fatalf("expected path to exist")
	}
}

func TestContextStatsAndSummary(t *testing.T) {
	st, dir := newTestStore(t)
	defer st.Close()

	_, err := st.InsertImportedSession(SessionInput{
		SessionID:      "claude-stats-1",
		SessionType:    "claude",
		SessionPath:    filepath.Join(dir, "claude.jsonl"),
		WorkspacePath:  "/tmp/ws-a",
		SessionTitle:   "claude one",
		SessionSummary: "hello",
	}, []TurnInput{
		{UserMessage: "u1", AssistantSummary: "a1", Timestamp: "2026-01-01T00:00:01Z"},
		{UserMessage: "u2", AssistantSummary: "a2", Timestamp: "2026-01-01T00:00:02Z"},
	})
	if err != nil {
		t.Fatalf("insert 1: %v", err)
	}
	_, err = st.InsertImportedSession(SessionInput{
		SessionID:      "codex-stats-1",
		SessionType:    "codex",
		SessionPath:    filepath.Join(dir, "codex.jsonl"),
		WorkspacePath:  "/tmp/ws-a",
		SessionTitle:   "codex one",
		SessionSummary: "world",
	}, []TurnInput{
		{UserMessage: "u3", AssistantSummary: "a3", Timestamp: "2026-01-01T00:00:03Z"},
	})
	if err != nil {
		t.Fatalf("insert 2: %v", err)
	}

	stats, err := st.GetContextStats("default")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.SessionCount != 2 {
		t.Fatalf("session count = %d", stats.SessionCount)
	}
	if stats.TurnCount != 3 {
		t.Fatalf("turn count = %d", stats.TurnCount)
	}
	if len(stats.SourceCounts) != 2 {
		t.Fatalf("source counts len = %d", len(stats.SourceCounts))
	}
	if len(stats.WorkspaceCounts) < 1 || stats.WorkspaceCounts[0].Key != "/tmp/ws-a" {
		t.Fatalf("workspace counts unexpected: %+v", stats.WorkspaceCounts)
	}
	summary := BuildContextSummary(stats)
	if summary == "" {
		t.Fatalf("empty summary")
	}
	if err := st.RefreshContextSummary("default"); err != nil {
		t.Fatalf("refresh summary: %v", err)
	}
	ctx, _, err := st.GetContext("default")
	if err != nil {
		t.Fatalf("get context: %v", err)
	}
	if ctx.Summary == "" {
		t.Fatalf("context summary empty after refresh")
	}
}

func TestListAndGetSessions(t *testing.T) {
	st, dir := newTestStore(t)
	defer st.Close()

	_, err := st.InsertImportedSession(SessionInput{
		SessionID:      "claude-list-1",
		SessionType:    "claude",
		SessionPath:    filepath.Join(dir, "claude-list-1.jsonl"),
		WorkspacePath:  "/tmp/ws-list",
		StartedAt:      "2026-01-01T00:00:00Z",
		LastActivityAt: "2026-01-01T00:00:05Z",
		SessionTitle:   "list title",
		SessionSummary: "list summary",
		Metadata:       "{}",
	}, []TurnInput{
		{UserMessage: "hello", AssistantSummary: "world", Timestamp: "2026-01-01T00:00:01Z"},
		{UserMessage: "q2", AssistantSummary: "a2", Timestamp: "2026-01-01T00:00:02Z"},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	sessions, err := st.ListSessions(10)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("session len = %d", len(sessions))
	}
	if sessions[0].ID != "claude-list-1" || sessions[0].TurnCount != 2 {
		t.Fatalf("unexpected session row: %+v", sessions[0])
	}

	got, err := st.GetSession("claude-list-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.SessionPath == "" || got.SessionType != "claude" || got.TurnCount != 2 {
		t.Fatalf("unexpected session details: %+v", got)
	}

	turns, err := st.ListTurns("claude-list-1", 10)
	if err != nil {
		t.Fatalf("list turns: %v", err)
	}
	if len(turns) != 2 {
		t.Fatalf("turn len = %d", len(turns))
	}
	// newest first
	if turns[0].TurnNumber != 2 {
		t.Fatalf("unexpected turn ordering: %+v", turns)
	}
}
