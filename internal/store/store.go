package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

type Context struct {
	ID        string
	Name      string
	Summary   string
	CreatedAt string
	UpdatedAt string
}

type SessionInput struct {
	SessionID      string
	SessionType    string
	SessionPath    string
	WorkspacePath  string
	StartedAt      string
	LastActivityAt string
	SessionTitle   string
	SessionSummary string
	Metadata       string
}

type TurnInput struct {
	UserMessage      string
	AssistantSummary string
	Timestamp        string
}

type ContextStats struct {
	ContextID       string
	SessionCount    int
	TurnCount       int
	LastActivityAt  string
	SourceCounts    []KVCount
	WorkspaceCounts []KVCount
}

type KVCount struct {
	Key   string
	Count int
}

type SessionRow struct {
	ID             string
	SessionType    string
	SessionPath    string
	WorkspacePath  string
	StartedAt      string
	LastActivityAt string
	SessionTitle   string
	SessionSummary string
	TurnCount      int
}

type TurnRow struct {
	TurnNumber       int
	UserMessage      string
	AssistantSummary string
	Timestamp        string
}

func DefaultDataDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ".ocx"
	}
	return filepath.Join(h, ".ocx")
}

func Open(dataDir string) (*Store, error) {
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "db"), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "db", "ocx.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) Migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at TEXT DEFAULT (datetime('now')),
			description TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			session_type TEXT NOT NULL,
			session_path TEXT NOT NULL,
			workspace_path TEXT,
			started_at TEXT,
			last_activity_at TEXT,
			session_title TEXT,
			session_summary TEXT,
			metadata TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS turns (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			turn_number INTEGER NOT NULL,
			user_message TEXT,
			assistant_summary TEXT,
			timestamp TEXT,
			content_hash TEXT,
			UNIQUE(session_id, turn_number)
		);`,
		`CREATE TABLE IF NOT EXISTS contexts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			summary TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS context_sessions (
			context_id TEXT NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
			session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			added_at TEXT DEFAULT (datetime('now')),
			PRIMARY KEY (context_id, session_id)
		);`,
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			created_at TEXT DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			payload TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		);`,
		`INSERT OR IGNORE INTO schema_version(version, description) VALUES (1, 'v1 bootstrap schema');`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) EnsureDefaultContext() error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO contexts(id, name, summary) VALUES (?, ?, ?)`, "default", "default", "Auto-linked imported sessions")
	return err
}

func (s *Store) UpsertImportedSession(kind, p string) (string, error) {
	if kind != "claude" && kind != "codex" {
		return "", errors.New("unsupported session type")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sid := fmt.Sprintf("%s-%d", kind, time.Now().UTC().UnixNano())
	if _, err := s.DB.Exec(`
		INSERT INTO sessions(id, session_type, session_path, workspace_path, started_at, last_activity_at, session_title, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, sid, kind, p, p, now, now, "imported session", "{}"); err != nil {
		return "", err
	}
	if _, err := s.DB.Exec(`INSERT OR IGNORE INTO context_sessions(context_id, session_id) VALUES ('default', ?)`, sid); err != nil {
		return "", err
	}
	return sid, nil
}

func (s *Store) InsertImportedSession(input SessionInput, turns []TurnInput) (string, error) {
	if input.SessionType != "claude" && input.SessionType != "codex" {
		return "", errors.New("unsupported session type")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sid := strings.TrimSpace(input.SessionID)
	if sid == "" {
		sid = fmt.Sprintf("%s-%d", input.SessionType, time.Now().UTC().UnixNano())
	}
	started := strings.TrimSpace(input.StartedAt)
	if started == "" {
		started = now
	}
	last := strings.TrimSpace(input.LastActivityAt)
	if last == "" {
		last = started
	}
	title := strings.TrimSpace(input.SessionTitle)
	if title == "" {
		title = "imported session"
	}
	metadata := strings.TrimSpace(input.Metadata)
	if metadata == "" {
		metadata = "{}"
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`
		INSERT INTO sessions(id, session_type, session_path, workspace_path, started_at, last_activity_at, session_title, session_summary, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sid, input.SessionType, input.SessionPath, input.WorkspacePath, started, last, title, input.SessionSummary, metadata); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO context_sessions(context_id, session_id) VALUES ('default', ?)`, sid); err != nil {
		return "", err
	}
	for i, t := range turns {
		turnTS := strings.TrimSpace(t.Timestamp)
		if turnTS == "" {
			turnTS = now
		}
		turnID := fmt.Sprintf("%s-turn-%d", sid, i+1)
		contentHash := fmt.Sprintf("%s-%d", sid, i+1)
		if _, err := tx.Exec(`
			INSERT INTO turns(id, session_id, turn_number, user_message, assistant_summary, timestamp, content_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, turnID, sid, i+1, t.UserMessage, t.AssistantSummary, turnTS, contentHash); err != nil {
			return "", err
		}
	}

	if _, err := tx.Exec(`UPDATE contexts SET updated_at = datetime('now') WHERE id = 'default'`); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	_ = s.RefreshContextSummary("default")
	return sid, nil
}

func (s *Store) CountSessions() (int, error) {
	var n int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) CountTurnsForSession(sessionID string) (int, error) {
	var n int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM turns WHERE session_id = ?`, sessionID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) SessionExistsByPath(sessionPath string) (bool, error) {
	var n int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE session_path = ?`, sessionPath).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) ListContexts() ([]Context, error) {
	rows, err := s.DB.Query(`SELECT id, name, COALESCE(summary,''), created_at, updated_at FROM contexts ORDER BY updated_at DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Context
	for rows.Next() {
		var c Context
		if err := rows.Scan(&c.ID, &c.Name, &c.Summary, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetContext(id string) (*Context, int, error) {
	var c Context
	err := s.DB.QueryRow(`SELECT id, name, COALESCE(summary,''), created_at, updated_at FROM contexts WHERE id = ?`, id).Scan(
		&c.ID, &c.Name, &c.Summary, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM context_sessions WHERE context_id = ?`, id).Scan(&count); err != nil {
		return nil, 0, err
	}
	return &c, count, nil
}

func (s *Store) SessionsForContext(id string) ([]map[string]string, error) {
	rows, err := s.DB.Query(`
		SELECT s.id, s.session_type, s.session_path, COALESCE(s.session_title,''), COALESCE(s.last_activity_at,'')
		FROM sessions s
		JOIN context_sessions cs ON cs.session_id = s.id
		WHERE cs.context_id = ?
		ORDER BY s.last_activity_at DESC, s.id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]string
	for rows.Next() {
		var id, typ, p, title, last string
		if err := rows.Scan(&id, &typ, &p, &title, &last); err != nil {
			return nil, err
		}
		out = append(out, map[string]string{
			"id": id, "type": typ, "path": p, "title": title, "last_activity": last,
		})
	}
	return out, rows.Err()
}

func (s *Store) SessionsForContextRows(contextID string) ([]SessionRow, error) {
	rows, err := s.DB.Query(`
		SELECT s.id, s.session_type, s.session_path, COALESCE(s.workspace_path,''), COALESCE(s.started_at,''), COALESCE(s.last_activity_at,''),
		       COALESCE(s.session_title,''), COALESCE(s.session_summary,''), COALESCE(t.turn_count,0)
		FROM context_sessions cs
		JOIN sessions s ON s.id = cs.session_id
		LEFT JOIN (
			SELECT session_id, COUNT(*) AS turn_count
			FROM turns
			GROUP BY session_id
		) t ON t.session_id = s.id
		WHERE cs.context_id = ?
		ORDER BY s.last_activity_at DESC, s.id
	`, contextID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SessionRow, 0)
	for rows.Next() {
		var srow SessionRow
		if err := rows.Scan(
			&srow.ID, &srow.SessionType, &srow.SessionPath, &srow.WorkspacePath, &srow.StartedAt,
			&srow.LastActivityAt, &srow.SessionTitle, &srow.SessionSummary, &srow.TurnCount,
		); err != nil {
			return nil, err
		}
		out = append(out, srow)
	}
	return out, rows.Err()
}

func (s *Store) ListSessions(limit int) ([]SessionRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB.Query(`
		SELECT s.id, s.session_type, s.session_path, COALESCE(s.workspace_path,''), COALESCE(s.started_at,''), COALESCE(s.last_activity_at,''),
		       COALESCE(s.session_title,''), COALESCE(s.session_summary,''), COALESCE(t.turn_count,0)
		FROM sessions s
		LEFT JOIN (
			SELECT session_id, COUNT(*) AS turn_count
			FROM turns
			GROUP BY session_id
		) t ON t.session_id = s.id
		ORDER BY s.last_activity_at DESC, s.id
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SessionRow, 0)
	for rows.Next() {
		var srow SessionRow
		if err := rows.Scan(
			&srow.ID, &srow.SessionType, &srow.SessionPath, &srow.WorkspacePath, &srow.StartedAt,
			&srow.LastActivityAt, &srow.SessionTitle, &srow.SessionSummary, &srow.TurnCount,
		); err != nil {
			return nil, err
		}
		out = append(out, srow)
	}
	return out, rows.Err()
}

func (s *Store) SearchSessions(query string, limit int) ([]SessionRow, error) {
	if limit <= 0 {
		limit = 100
	}
	q := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := s.DB.Query(`
		SELECT s.id, s.session_type, s.session_path, COALESCE(s.workspace_path,''), COALESCE(s.started_at,''), COALESCE(s.last_activity_at,''),
		       COALESCE(s.session_title,''), COALESCE(s.session_summary,''), COALESCE(t.turn_count,0)
		FROM sessions s
		LEFT JOIN (
			SELECT session_id, COUNT(*) AS turn_count
			FROM turns
			GROUP BY session_id
		) t ON t.session_id = s.id
		WHERE LOWER(s.id) LIKE ?
		   OR LOWER(COALESCE(s.session_title,'')) LIKE ?
		   OR LOWER(COALESCE(s.session_summary,'')) LIKE ?
		   OR LOWER(COALESCE(s.session_path,'')) LIKE ?
		   OR LOWER(COALESCE(s.workspace_path,'')) LIKE ?
		ORDER BY s.last_activity_at DESC, s.id
		LIMIT ?
	`, q, q, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SessionRow, 0)
	for rows.Next() {
		var srow SessionRow
		if err := rows.Scan(
			&srow.ID, &srow.SessionType, &srow.SessionPath, &srow.WorkspacePath, &srow.StartedAt,
			&srow.LastActivityAt, &srow.SessionTitle, &srow.SessionSummary, &srow.TurnCount,
		); err != nil {
			return nil, err
		}
		out = append(out, srow)
	}
	return out, rows.Err()
}

func (s *Store) GetSession(id string) (*SessionRow, error) {
	var srow SessionRow
	err := s.DB.QueryRow(`
		SELECT s.id, s.session_type, s.session_path, COALESCE(s.workspace_path,''), COALESCE(s.started_at,''), COALESCE(s.last_activity_at,''),
		       COALESCE(s.session_title,''), COALESCE(s.session_summary,''), COALESCE(t.turn_count,0)
		FROM sessions s
		LEFT JOIN (
			SELECT session_id, COUNT(*) AS turn_count
			FROM turns
			GROUP BY session_id
		) t ON t.session_id = s.id
		WHERE s.id = ?
	`, id).Scan(
		&srow.ID, &srow.SessionType, &srow.SessionPath, &srow.WorkspacePath, &srow.StartedAt,
		&srow.LastActivityAt, &srow.SessionTitle, &srow.SessionSummary, &srow.TurnCount,
	)
	if err != nil {
		return nil, err
	}
	return &srow, nil
}

func (s *Store) ListTurns(sessionID string, limit int) ([]TurnRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.DB.Query(`
		SELECT turn_number, COALESCE(user_message,''), COALESCE(assistant_summary,''), COALESCE(timestamp,'')
		FROM turns
		WHERE session_id = ?
		ORDER BY turn_number DESC
		LIMIT ?
	`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TurnRow, 0)
	for rows.Next() {
		var tr TurnRow
		if err := rows.Scan(&tr.TurnNumber, &tr.UserMessage, &tr.AssistantSummary, &tr.Timestamp); err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

func (s *Store) GetContextStats(contextID string) (ContextStats, error) {
	stats := ContextStats{ContextID: contextID}
	var exists int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM contexts WHERE id = ?`, contextID).Scan(&exists); err != nil {
		return ContextStats{}, err
	}
	if exists == 0 {
		return ContextStats{}, sql.ErrNoRows
	}

	if err := s.DB.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(ts.turn_count),0), COALESCE(MAX(s.last_activity_at),'')
		FROM context_sessions cs
		JOIN sessions s ON s.id = cs.session_id
		LEFT JOIN (
			SELECT session_id, COUNT(*) AS turn_count
			FROM turns
			GROUP BY session_id
		) ts ON ts.session_id = s.id
		WHERE cs.context_id = ?
	`, contextID).Scan(&stats.SessionCount, &stats.TurnCount, &stats.LastActivityAt); err != nil {
		return ContextStats{}, err
	}

	srcRows, err := s.DB.Query(`
		SELECT s.session_type, COUNT(*)
		FROM context_sessions cs
		JOIN sessions s ON s.id = cs.session_id
		WHERE cs.context_id = ?
		GROUP BY s.session_type
	`, contextID)
	if err != nil {
		return ContextStats{}, err
	}
	for srcRows.Next() {
		var k string
		var c int
		if err := srcRows.Scan(&k, &c); err != nil {
			srcRows.Close()
			return ContextStats{}, err
		}
		stats.SourceCounts = append(stats.SourceCounts, KVCount{Key: k, Count: c})
	}
	srcRows.Close()

	wsRows, err := s.DB.Query(`
		SELECT COALESCE(NULLIF(s.workspace_path,''),'(unknown)'), COUNT(*)
		FROM context_sessions cs
		JOIN sessions s ON s.id = cs.session_id
		WHERE cs.context_id = ?
		GROUP BY COALESCE(NULLIF(s.workspace_path,''),'(unknown)')
	`, contextID)
	if err != nil {
		return ContextStats{}, err
	}
	for wsRows.Next() {
		var k string
		var c int
		if err := wsRows.Scan(&k, &c); err != nil {
			wsRows.Close()
			return ContextStats{}, err
		}
		stats.WorkspaceCounts = append(stats.WorkspaceCounts, KVCount{Key: k, Count: c})
	}
	wsRows.Close()

	sort.Slice(stats.SourceCounts, func(i, j int) bool {
		if stats.SourceCounts[i].Count == stats.SourceCounts[j].Count {
			return stats.SourceCounts[i].Key < stats.SourceCounts[j].Key
		}
		return stats.SourceCounts[i].Count > stats.SourceCounts[j].Count
	})
	sort.Slice(stats.WorkspaceCounts, func(i, j int) bool {
		if stats.WorkspaceCounts[i].Count == stats.WorkspaceCounts[j].Count {
			return stats.WorkspaceCounts[i].Key < stats.WorkspaceCounts[j].Key
		}
		return stats.WorkspaceCounts[i].Count > stats.WorkspaceCounts[j].Count
	})

	return stats, nil
}

func BuildContextSummary(stats ContextStats) string {
	if stats.SessionCount == 0 {
		return "No sessions imported yet."
	}
	src := "sources: n/a"
	if len(stats.SourceCounts) > 0 {
		parts := make([]string, 0, len(stats.SourceCounts))
		for _, it := range stats.SourceCounts {
			parts = append(parts, fmt.Sprintf("%s=%d", it.Key, it.Count))
		}
		src = "sources: " + strings.Join(parts, ", ")
	}
	ws := ""
	if len(stats.WorkspaceCounts) > 0 {
		top := stats.WorkspaceCounts[0]
		ws = fmt.Sprintf("; top workspace: %s (%d)", top.Key, top.Count)
	}
	last := ""
	if stats.LastActivityAt != "" {
		last = fmt.Sprintf("; last activity: %s", stats.LastActivityAt)
	}
	return fmt.Sprintf("%d sessions, %d turns; %s%s%s", stats.SessionCount, stats.TurnCount, src, ws, last)
}

func (s *Store) RefreshContextSummary(contextID string) error {
	stats, err := s.GetContextStats(contextID)
	if err != nil {
		return err
	}
	summary := BuildContextSummary(stats)
	_, err = s.DB.Exec(`UPDATE contexts SET summary = ?, updated_at = datetime('now') WHERE id = ?`, summary, contextID)
	return err
}
