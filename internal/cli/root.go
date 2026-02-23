package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	cmd := &cobra.Command{
		Use:   "auto",
		Short: "Auto-ingest from ~/.claude/projects and ~/.codex/sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
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
				inserted, parsed, skipped, err := importFromPathWithOptions(st, src.kind, src.path, dryRun)
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
	return cmd
}

func importFromPath(st *store.Store, kind, p string) (inserted int, parsed int, skipped int, err error) {
	return importFromPathWithOptions(st, kind, p, false)
}

func importFromPathWithOptions(st *store.Store, kind, p string, dryRun bool) (inserted int, parsed int, skipped int, err error) {
	sessions, err := adapters.Parse(kind, p)
	if err != nil {
		return 0, 0, 0, err
	}
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
		if dryRun {
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
	if !dryRun {
		_ = st.RefreshContextSummary("default")
	}
	return inserted, parsed, skipped, nil
}
