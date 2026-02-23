package share

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gh-xj/omnicontext/internal/store"
)

type ExportPayload struct {
	Version    int                 `json:"version"`
	ContextID  string              `json:"context_id"`
	ExportedAt string              `json:"exported_at"`
	Context    *store.Context      `json:"context"`
	Sessions   []map[string]string `json:"sessions"`
}

func ExportContext(st *store.Store, contextID, outPath string) error {
	ctx, _, err := st.GetContext(contextID)
	if err != nil {
		return err
	}
	sessions, err := st.SessionsForContext(contextID)
	if err != nil {
		return err
	}
	payload := ExportPayload{
		Version:    1,
		ContextID:  contextID,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Context:    ctx,
		Sessions:   sessions,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	w, err := zw.Create("context.json")
	if err != nil {
		_ = zw.Close()
		return err
	}
	if _, err := w.Write(data); err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

func ImportContext(st *store.Store, packPath string) (string, int, error) {
	zr, err := zip.OpenReader(packPath)
	if err != nil {
		return "", 0, err
	}
	defer zr.Close()
	var content []byte
	for _, f := range zr.File {
		if f.Name != "context.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", 0, err
		}
		content, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", 0, err
		}
	}
	if len(content) == 0 {
		return "", 0, fmt.Errorf("context.json not found in pack")
	}
	var payload ExportPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return "", 0, err
	}
	ctxID := payload.ContextID + "-imported"
	if payload.Context != nil && payload.Context.ID != "" {
		ctxID = payload.Context.ID + "-imported"
	}
	name := "imported context"
	summary := "Imported from .ocxpack"
	if payload.Context != nil {
		if payload.Context.Name != "" {
			name = payload.Context.Name + " (imported)"
		}
		if payload.Context.Summary != "" {
			summary = payload.Context.Summary
		}
	}
	if _, err := st.DB.Exec(`INSERT OR REPLACE INTO contexts(id, name, summary, updated_at) VALUES (?, ?, ?, datetime('now'))`, ctxID, name, summary); err != nil {
		return "", 0, err
	}
	inserted := 0
	for _, s := range payload.Sessions {
		typ := s["type"]
		p := s["path"]
		if typ == "" || p == "" {
			continue
		}
		sid, err := st.InsertImportedSession(store.SessionInput{
			SessionType:    typ,
			SessionPath:    p,
			WorkspacePath:  p,
			SessionTitle:   "imported session",
			SessionSummary: "Imported from pack",
			Metadata:       "{}",
		}, nil)
		if err != nil {
			continue
		}
		_, _ = st.DB.Exec(`INSERT OR IGNORE INTO context_sessions(context_id, session_id) VALUES (?, ?)`, ctxID, sid)
		inserted++
	}
	return ctxID, inserted, nil
}
