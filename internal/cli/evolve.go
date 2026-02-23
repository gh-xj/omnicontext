package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gh-xj/omnicontext/internal/evolve"
)

func newEvolveCmd(dataDirProvider func() string) *cobra.Command {
	var cfg evolve.Config
	var jsonOut bool

	cmd := &cobra.Command{Use: "evolve", Short: "Self-evolution harness with verification loop and PR handoff"}
	run := &cobra.Command{
		Use:   "run",
		Short: "Run local evolve loop, then prepare PR artifacts for human review",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Goal == "" {
				return errors.New("--goal is required")
			}
			if cfg.InspectorCommand == "" {
				return errors.New("--inspector is required")
			}
			if cfg.Workspace == "" {
				wd, _ := os.Getwd()
				cfg.Workspace = wd
			}
			cfg.DataDir = dataDirProvider()
			r, err := evolve.Run(cfg)
			if jsonOut {
				b, _ := json.MarshalIndent(r, "", "  ")
				fmt.Println(string(b))
			} else {
				fmt.Printf("run_id: %s\n", r.RunID)
				fmt.Printf("branch: %s\n", r.Branch)
				fmt.Printf("qualified: %v\n", r.Qualified)
				fmt.Printf("message: %s\n", r.Message)
				fmt.Printf("lab_report: %s\n", r.LabReportFile)
				if r.PRTitleFile != "" {
					fmt.Printf("pr_title: %s\n", r.PRTitleFile)
				}
				if r.PRBodyFile != "" {
					fmt.Printf("pr_body: %s\n", r.PRBodyFile)
				}
			}
			return err
		},
	}

	run.Flags().StringVar(&cfg.Goal, "goal", "", "Target goal for auto-improvement loop")
	run.Flags().IntVar(&cfg.MaxIterations, "max-iterations", 3, "Max loop iterations")
	run.Flags().StringVar(&cfg.ContextDesigner, "context-designer", "", "Context designer command (optional)")
	run.Flags().StringVar(&cfg.LauncherCommand, "launcher", "", "Agent launcher command (optional)")
	run.Flags().StringVar(&cfg.VerifyCommand, "verify", "go vet ./... && go test ./... && go build ./cmd/ocx", "Verification command")
	run.Flags().StringVar(&cfg.InspectorCommand, "inspector", "", "Inspector command (required)")
	run.Flags().StringVar(&cfg.PlannerCommand, "planner", "", "Planner command (optional)")
	run.Flags().StringVar(&cfg.ImplementerCommand, "implementer", "", "Implementer command (optional)")
	run.Flags().StringVar(&cfg.Workspace, "workspace", "", "Workspace directory (default: cwd)")
	run.Flags().StringVar(&cfg.Branch, "branch", "", "Branch name (default: evolve/<timestamp>)")
	run.Flags().BoolVar(&cfg.AllowDirty, "allow-dirty", false, "Allow running with dirty working tree")
	run.Flags().BoolVar(&cfg.AutoCommit, "auto-commit", true, "Commit qualified changes automatically")
	run.Flags().StringVar(&cfg.CommitMessage, "commit-message", "", "Commit message for auto-commit")
	run.Flags().BoolVar(&cfg.OpenDraftPR, "open-draft-pr", false, "Create draft PR automatically with gh")
	run.Flags().StringVar(&cfg.PRTitle, "pr-title", "", "PR title override")
	run.Flags().BoolVar(&jsonOut, "json", false, "Emit JSON output")

	cmd.AddCommand(run)
	return cmd
}
