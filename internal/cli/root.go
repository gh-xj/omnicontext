package cli

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gh-xj/omnicontext/internal/adapters"
	"github.com/gh-xj/omnicontext/internal/share"
	"github.com/gh-xj/omnicontext/internal/store"
	"github.com/gh-xj/omnicontext/internal/tui"
)

func NewRootCmd() *cobra.Command {
	var dataDir string
	root := &cobra.Command{
		Use:   "ocx",
		Short: "OmniContext CLI (OSS MVP)",
	}
	root.PersistentFlags().StringVar(&dataDir, "data-dir", store.DefaultDataDir(), "Data directory (default: ~/.ocx)")

	openStore := func() (*store.Store, error) {
		st, err := store.Open(dataDir)
		if err != nil {
			return nil, err
		}
		if err := st.Migrate(); err != nil {
			_ = st.Close()
			return nil, err
		}
		if err := st.EnsureDefaultContext(); err != nil {
			_ = st.Close()
			return nil, err
		}
		return st, nil
	}

	root.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize local store",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			fmt.Printf("initialized: %s\n", filepath.Join(dataDir, "db", "ocx.db"))
			return nil
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Basic health checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			fmt.Printf("ok: db=%s\n", filepath.Join(dataDir, "db", "ocx.db"))
			fmt.Println("ok: schema migrated")
			return nil
		},
	})

	importCmd := &cobra.Command{Use: "import", Short: "Import sessions"}
	importCmd.AddCommand(newImportSubCmd("claude", openStore))
	importCmd.AddCommand(newImportSubCmd("codex", openStore))
	root.AddCommand(importCmd)

	ingestCmd := &cobra.Command{Use: "ingest", Short: "Ingest sessions from default sources"}
	ingestCmd.AddCommand(newIngestAutoCmd(openStore))
	root.AddCommand(ingestCmd)

	contextCmd := &cobra.Command{Use: "context", Short: "Context operations"}
	contextCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			items, err := st.ListContexts()
			if err != nil {
				return err
			}
			for _, c := range items {
				fmt.Printf("%s\t%s\t%s\n", c.ID, c.Name, c.Summary)
			}
			return nil
		},
	})
	contextCmd.AddCommand(&cobra.Command{
		Use:   "show <context-id>",
		Short: "Show one context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			_ = st.RefreshContextSummary(args[0])
			ctx, count, err := st.GetContext(args[0])
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("context not found: %s", args[0])
				}
				return err
			}
			fmt.Printf("id: %s\nname: %s\nsummary: %s\nsessions: %d\n", ctx.ID, ctx.Name, ctx.Summary, count)
			sessions, err := st.SessionsForContext(ctx.ID)
			if err != nil {
				return err
			}
			for _, s := range sessions {
				fmt.Printf("- %s\t%s\t%s\n", s["id"], s["type"], s["path"])
			}
			return nil
		},
	})
	contextCmd.AddCommand(&cobra.Command{
		Use:   "stats <context-id>",
		Short: "Show aggregated stats for a context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			stats, err := st.GetContextStats(args[0])
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("context not found: %s", args[0])
				}
				return err
			}
			fmt.Printf("context: %s\nsessions: %d\nturns: %d\nlast_activity: %s\n", stats.ContextID, stats.SessionCount, stats.TurnCount, stats.LastActivityAt)
			fmt.Printf("summary: %s\n", store.BuildContextSummary(stats))
			fmt.Println("sources:")
			if len(stats.SourceCounts) == 0 {
				fmt.Println("- (none)")
			}
			for _, kv := range stats.SourceCounts {
				fmt.Printf("- %s\t%d\n", kv.Key, kv.Count)
			}
			fmt.Println("workspaces:")
			if len(stats.WorkspaceCounts) == 0 {
				fmt.Println("- (none)")
			}
			for i, kv := range stats.WorkspaceCounts {
				if i >= 10 {
					break
				}
				fmt.Printf("- %s\t%d\n", kv.Key, kv.Count)
			}
			return nil
		},
	})
	contextExportCmd := &cobra.Command{Use: "export", Short: "Export context data"}
	var contextCSVOut string
	contextExportCSV := &cobra.Command{
		Use:   "csv <context-id>",
		Short: "Export context sessions as CSV",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if contextCSVOut == "" {
				return errors.New("--out is required")
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if _, _, err := st.GetContext(args[0]); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("context not found: %s", args[0])
				}
				return err
			}
			rows, err := st.SessionsForContextRows(args[0])
			if err != nil {
				return err
			}
			if err := writeSessionsCSV(contextCSVOut, rows); err != nil {
				return err
			}
			fmt.Printf("exported context csv: %s (%d rows)\n", contextCSVOut, len(rows))
			return nil
		},
	}
	contextExportCSV.Flags().StringVar(&contextCSVOut, "out", "", "Output CSV file path")
	contextExportCmd.AddCommand(contextExportCSV)
	contextCmd.AddCommand(contextExportCmd)
	root.AddCommand(contextCmd)

	sessionCmd := &cobra.Command{Use: "session", Short: "Session operations"}
	var sessionListLimit int
	sessionList := &cobra.Command{
		Use:   "list",
		Short: "List imported sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			items, err := st.ListSessions(sessionListLimit)
			if err != nil {
				return err
			}
			for _, s := range items {
				fmt.Printf("%s\t%s\tturns=%d\t%s\n", s.ID, s.SessionType, s.TurnCount, s.SessionPath)
			}
			return nil
		},
	}
	sessionList.Flags().IntVar(&sessionListLimit, "limit", 50, "Max sessions to show")
	sessionCmd.AddCommand(sessionList)

	var turnLimit int
	sessionShow := &cobra.Command{
		Use:   "show <session-id>",
		Short: "Show session details and recent turns",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			s, err := st.GetSession(args[0])
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("session not found: %s", args[0])
				}
				return err
			}
			fmt.Printf("id: %s\n", s.ID)
			fmt.Printf("type: %s\n", s.SessionType)
			fmt.Printf("path: %s\n", s.SessionPath)
			fmt.Printf("workspace: %s\n", s.WorkspacePath)
			fmt.Printf("started_at: %s\n", s.StartedAt)
			fmt.Printf("last_activity: %s\n", s.LastActivityAt)
			fmt.Printf("title: %s\n", s.SessionTitle)
			fmt.Printf("summary: %s\n", s.SessionSummary)
			fmt.Printf("turns: %d\n", s.TurnCount)

			turns, err := st.ListTurns(s.ID, turnLimit)
			if err != nil {
				return err
			}
			fmt.Printf("recent_turns (newest first, limit=%d):\n", turnLimit)
			if len(turns) == 0 {
				fmt.Println("- (none)")
				return nil
			}
			for _, t := range turns {
				user := t.UserMessage
				if len(user) > 120 {
					user = user[:120] + "..."
				}
				asst := t.AssistantSummary
				if len(asst) > 120 {
					asst = asst[:120] + "..."
				}
				fmt.Printf("- #%d @ %s\n  user: %s\n  assistant: %s\n", t.TurnNumber, t.Timestamp, user, asst)
			}
			return nil
		},
	}
	sessionShow.Flags().IntVar(&turnLimit, "turn-limit", 10, "Number of recent turns to include")
	sessionCmd.AddCommand(sessionShow)

	var sessionSearchLimit int
	var sessionSearchQuery string
	sessionSearch := &cobra.Command{
		Use:   "search",
		Short: "Search sessions by id/title/summary/path/workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sessionSearchQuery) == "" {
				return errors.New("--query is required")
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			items, err := st.SearchSessions(sessionSearchQuery, sessionSearchLimit)
			if err != nil {
				return err
			}
			for _, s := range items {
				fmt.Printf("%s\t%s\tturns=%d\t%s\t%s\n", s.ID, s.SessionType, s.TurnCount, s.SessionTitle, s.SessionPath)
			}
			return nil
		},
	}
	sessionSearch.Flags().StringVar(&sessionSearchQuery, "query", "", "Search text")
	sessionSearch.Flags().IntVar(&sessionSearchLimit, "limit", 50, "Max sessions to show")
	sessionCmd.AddCommand(sessionSearch)

	sessionExportCmd := &cobra.Command{Use: "export", Short: "Export session data"}
	var sessionCSVOut string
	var sessionCSVLimit int
	var sessionCSVQuery string
	sessionExportCSV := &cobra.Command{
		Use:   "csv",
		Short: "Export sessions as CSV",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionCSVOut == "" {
				return errors.New("--out is required")
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			var rows []store.SessionRow
			if strings.TrimSpace(sessionCSVQuery) == "" {
				rows, err = st.ListSessions(sessionCSVLimit)
			} else {
				rows, err = st.SearchSessions(sessionCSVQuery, sessionCSVLimit)
			}
			if err != nil {
				return err
			}
			if err := writeSessionsCSV(sessionCSVOut, rows); err != nil {
				return err
			}
			fmt.Printf("exported session csv: %s (%d rows)\n", sessionCSVOut, len(rows))
			return nil
		},
	}
	sessionExportCSV.Flags().StringVar(&sessionCSVOut, "out", "", "Output CSV file path")
	sessionExportCSV.Flags().IntVar(&sessionCSVLimit, "limit", 1000, "Max sessions to export")
	sessionExportCSV.Flags().StringVar(&sessionCSVQuery, "query", "", "Optional search filter")
	sessionExportCmd.AddCommand(sessionExportCSV)
	sessionCmd.AddCommand(sessionExportCmd)
	root.AddCommand(sessionCmd)

	shareCmd := &cobra.Command{Use: "share", Short: "Local share pack operations"}
	exportCmd := &cobra.Command{
		Use:   "export <context-id>",
		Short: "Export context as .ocxpack zip",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, _ := cmd.Flags().GetString("out")
			if out == "" {
				out = "./context.ocxpack"
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			if err := share.ExportContext(st, args[0], out); err != nil {
				return err
			}
			fmt.Printf("exported: %s\n", out)
			return nil
		},
	}
	exportCmd.Flags().String("out", "./context.ocxpack", "Output .ocxpack path")
	shareCmd.AddCommand(exportCmd)

	importShareCmd := &cobra.Command{
		Use:   "import <pack-file>",
		Short: "Import context from .ocxpack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			ctxID, n, err := share.ImportContext(st, args[0])
			if err != nil {
				return err
			}
			_ = st.RefreshContextSummary(ctxID)
			_ = st.RefreshContextSummary("default")
			fmt.Printf("imported context: %s (sessions=%d)\n", ctxID, n)
			return nil
		},
	}
	shareCmd.AddCommand(importShareCmd)
	root.AddCommand(shareCmd)

	root.AddCommand(&cobra.Command{
		Use:   "dashboard",
		Short: "Open MVP Bubble Tea dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			return tui.RunDashboard(st)
		},
	})
	dataDirProvider := func() string { return dataDir }
	root.AddCommand(newLabCmd(dataDirProvider))
	root.AddCommand(newEvolveCmd(dataDirProvider))

	return root
}

func newImportSubCmd(kind string, openStore func() (*store.Store, error)) *cobra.Command {
	var p string
	cmd := &cobra.Command{
		Use:   kind,
		Short: fmt.Sprintf("Import %s sessions from a path", kind),
		RunE: func(cmd *cobra.Command, args []string) error {
			if p == "" {
				return errors.New("--path is required")
			}
			if _, err := os.Stat(p); err != nil {
				return err
			}
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			inserted, parsed, skipped, err := importFromPath(st, kind, p)
			if err != nil {
				return err
			}
			fmt.Printf("imported %s sessions: %d/%d (skipped=%d)\n", kind, inserted, parsed, skipped)
			return nil
		},
	}
	cmd.Flags().StringVar(&p, "path", "", "Source session path")
	return cmd
}

func newIngestAutoCmd(openStore func() (*store.Store, error)) *cobra.Command {
	var claudePath string
	var codexPath string
	var dryRun bool
	var jsonOut bool
	var maxSessions int
	var since string
	cmd := &cobra.Command{
		Use:   "auto",
		Short: "Auto-ingest from ~/.claude/projects and ~/.codex/sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			sinceTime, err := parseSinceDate(since)
			if err != nil {
				return err
			}
			if claudePath == "" || codexPath == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				if claudePath == "" {
					claudePath = filepath.Join(home, ".claude", "projects")
				}
				if codexPath == "" {
					codexPath = filepath.Join(home, ".codex", "sessions")
				}
			}

			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			type result struct {
				kind     string
				path     string
				parsed   int
				inserted int
				skipped  int
				err      error
			}
			results := make([]result, 0, 2)
			for _, src := range []struct {
				kind string
				path string
			}{
				{kind: "claude", path: claudePath},
				{kind: "codex", path: codexPath},
			} {
				if _, err := os.Stat(src.path); err != nil {
					results = append(results, result{kind: src.kind, path: src.path, err: err})
					continue
				}
				inserted, parsed, skipped, err := importFromPathWithOptions(st, src.kind, src.path, importOptions{
					DryRun:      dryRun,
					MaxSessions: maxSessions,
					Since:       sinceTime,
				})
				results = append(results, result{
					kind:     src.kind,
					path:     src.path,
					parsed:   parsed,
					inserted: inserted,
					skipped:  skipped,
					err:      err,
				})
			}

			totalParsed, totalInserted, totalSkipped := 0, 0, 0
			if jsonOut {
				type item struct {
					Kind     string `json:"kind"`
					Path     string `json:"path"`
					Parsed   int    `json:"parsed"`
					Inserted int    `json:"inserted"`
					Skipped  int    `json:"skipped"`
					Error    string `json:"error,omitempty"`
				}
				out := struct {
					DryRun bool   `json:"dry_run"`
					Total  item   `json:"total"`
					Items  []item `json:"items"`
				}{
					DryRun: dryRun,
				}
				out.Items = make([]item, 0, len(results))
				for _, r := range results {
					it := item{
						Kind:     r.kind,
						Path:     r.path,
						Parsed:   r.parsed,
						Inserted: r.inserted,
						Skipped:  r.skipped,
					}
					if r.err != nil {
						it.Error = r.err.Error()
					}
					out.Items = append(out.Items, it)
					totalParsed += r.parsed
					totalInserted += r.inserted
					totalSkipped += r.skipped
				}
				out.Total = item{
					Kind:     "all",
					Path:     "",
					Parsed:   totalParsed,
					Inserted: totalInserted,
					Skipped:  totalSkipped,
				}
				b, err := json.MarshalIndent(out, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(b))
				return nil
			}

			for _, r := range results {
				if r.err != nil {
					fmt.Printf("%s: path=%s error=%v\n", r.kind, r.path, r.err)
					continue
				}
				fmt.Printf("%s: imported=%d/%d skipped=%d path=%s\n", r.kind, r.inserted, r.parsed, r.skipped, r.path)
				totalParsed += r.parsed
				totalInserted += r.inserted
				totalSkipped += r.skipped
			}
			if dryRun {
				fmt.Printf("total (dry-run): importable=%d/%d skipped=%d\n", totalInserted, totalParsed, totalSkipped)
			} else {
				fmt.Printf("total: imported=%d/%d skipped=%d\n", totalInserted, totalParsed, totalSkipped)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&claudePath, "claude-path", "", "Claude sessions root (default: ~/.claude/projects)")
	cmd.Flags().StringVar(&codexPath, "codex-path", "", "Codex sessions root (default: ~/.codex/sessions)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview import counts without writing to DB")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit machine-readable JSON output")
	cmd.Flags().IntVar(&maxSessions, "max-sessions", 0, "Max sessions per source to ingest after filtering")
	cmd.Flags().StringVar(&since, "since", "", "Only ingest sessions with activity on/after YYYY-MM-DD")
	return cmd
}

func importFromPath(st *store.Store, kind, p string) (inserted int, parsed int, skipped int, err error) {
	return importFromPathWithOptions(st, kind, p, importOptions{})
}

type importOptions struct {
	DryRun      bool
	MaxSessions int
	Since       *time.Time
}

func importFromPathWithOptions(st *store.Store, kind, p string, opts importOptions) (inserted int, parsed int, skipped int, err error) {
	sessions, err := adapters.Parse(kind, p)
	if err != nil {
		return 0, 0, 0, err
	}
	sessions = filterSessionsForImport(sessions, opts)
	parsed = len(sessions)
	for _, s := range sessions {
		exists, exErr := st.SessionExistsByPath(s.SessionPath)
		if exErr != nil {
			skipped++
			continue
		}
		if exists {
			skipped++
			continue
		}
		if opts.DryRun {
			inserted++
			continue
		}
		turns := make([]store.TurnInput, 0, len(s.Turns))
		for _, t := range s.Turns {
			turns = append(turns, store.TurnInput{
				UserMessage:      t.UserMessage,
				AssistantSummary: t.AssistantSummary,
				Timestamp:        t.Timestamp,
			})
		}
		_, insErr := st.InsertImportedSession(store.SessionInput{
			SessionID:      s.SessionID,
			SessionType:    s.SessionType,
			SessionPath:    s.SessionPath,
			WorkspacePath:  s.WorkspacePath,
			StartedAt:      s.StartedAt,
			LastActivityAt: s.LastActivityAt,
			SessionTitle:   s.SessionTitle,
			SessionSummary: s.SessionSummary,
			Metadata:       s.Metadata,
		}, turns)
		if insErr != nil {
			skipped++
			continue
		}
		inserted++
	}
	if !opts.DryRun {
		_ = st.RefreshContextSummary("default")
	}
	return inserted, parsed, skipped, nil
}

func parseSinceDate(v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, fmt.Errorf("invalid --since value %q, expected YYYY-MM-DD", v)
	}
	return &t, nil
}

func filterSessionsForImport(in []adapters.Session, opts importOptions) []adapters.Session {
	out := make([]adapters.Session, 0, len(in))
	for _, s := range in {
		if opts.Since != nil {
			ok := false
			candidates := []string{s.LastActivityAt, s.StartedAt}
			for _, ts := range candidates {
				if ts == "" {
					continue
				}
				t, err := time.Parse(time.RFC3339, ts)
				if err != nil {
					continue
				}
				if !t.Before(*opts.Since) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		ti := sessionTimeForSort(out[i])
		tj := sessionTimeForSort(out[j])
		return ti.After(tj)
	})
	if opts.MaxSessions > 0 && len(out) > opts.MaxSessions {
		out = out[:opts.MaxSessions]
	}
	return out
}

func sessionTimeForSort(s adapters.Session) time.Time {
	candidates := []string{s.LastActivityAt, s.StartedAt}
	for _, ts := range candidates {
		if ts == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func writeSessionsCSV(outPath string, rows []store.SessionRow) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"id", "session_type", "session_path", "workspace_path", "started_at", "last_activity_at", "session_title", "session_summary", "turn_count"}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, s := range rows {
		rec := []string{
			s.ID,
			s.SessionType,
			s.SessionPath,
			s.WorkspacePath,
			s.StartedAt,
			s.LastActivityAt,
			s.SessionTitle,
			s.SessionSummary,
			fmt.Sprintf("%d", s.TurnCount),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return w.Error()
}
