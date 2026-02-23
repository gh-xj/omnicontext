package cli

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

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
	root.AddCommand(contextCmd)

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
			sid, err := st.UpsertImportedSession(kind, p)
			if err != nil {
				return err
			}
			fmt.Printf("imported %s session: %s\n", kind, sid)
			return nil
		},
	}
	cmd.Flags().StringVar(&p, "path", "", "Source session path")
	return cmd
}
