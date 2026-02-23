package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
