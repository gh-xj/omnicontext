package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gh-xj/omnicontext/internal/lab"
)

func newLabCmd(dataDirProvider func() string) *cobra.Command {
	labCmd := &cobra.Command{Use: "lab", Short: "Multi-agent experiment runner with verification loop"}

	var runDir string
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize lab workspace and sample config",
		RunE: func(cmd *cobra.Command, args []string) error {
			d := resolveLabDir(dataDirProvider(), runDir)
			if err := os.MkdirAll(d, 0o755); err != nil {
				return err
			}
			sample := lab.Config{
				Goal:               "Fix failing tests and ship a safe patch",
				MaxIterations:      3,
				ContextDesigner:    "printf '# Experiment Context\\n\\n- control variables fixed\\n- acceptance explicit\\n' > \"$OCX_LAB_CONTEXT_PACK_FILE\"",
				LauncherCommand:    "echo 'launcher placeholder (invoke external agent here)' > \"$OCX_LAB_ITER_DIR/outbox/launcher.log\"",
				PlannerCommand:     "echo '- inspect failures\n- patch minimal\n- re-verify' > \"$OCX_LAB_PLAN_FILE\"",
				ImplementerCommand: "echo 'implementer placeholder' > \"$OCX_LAB_IMPL_FILE\"",
				VerifyCommand:      "go test ./...",
				InspectorCommand:   "cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'\n{\"verdict\":\"QUALIFIED\",\"reasons\":[\"sample passed\"],\"patch_hints\":[],\"confidence\":0.90}\nJSON",
				JudgeCommand:       "if grep -q 'FAIL' \"$OCX_LAB_ITER_DIR/outbox/verify.log\"; then echo NOT_QUALIFIED; exit 1; else echo QUALIFIED; fi",
			}
			b, _ := json.MarshalIndent(sample, "", "  ")
			cfgPath := filepath.Join(d, "sample-config.json")
			if err := os.WriteFile(cfgPath, b, 0o644); err != nil {
				return err
			}
			fmt.Printf("initialized lab dir: %s\n", d)
			fmt.Printf("sample config: %s\n", cfgPath)
			return nil
		},
	}
	initCmd.Flags().StringVar(&runDir, "run-dir", "", "Lab base dir (default: <data-dir>/lab/runs)")
	labCmd.AddCommand(initCmd)

	var goal string
	var maxIterations int
	var contextDesigner string
	var launcher string
	var planner string
	var implementer string
	var verify string
	var inspector string
	var judge string
	var shell string
	var configPath string
	var workspace string
	var jsonOut bool
	var runDirRun string

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run multi-agent loop until qualified or max iterations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := lab.Config{}
			if configPath != "" {
				b, err := os.ReadFile(configPath)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(b, &cfg); err != nil {
					return err
				}
			}
			if goal != "" {
				cfg.Goal = goal
			}
			if maxIterations > 0 {
				cfg.MaxIterations = maxIterations
			}
			if contextDesigner != "" {
				cfg.ContextDesigner = contextDesigner
			}
			if launcher != "" {
				cfg.LauncherCommand = launcher
			}
			if planner != "" {
				cfg.PlannerCommand = planner
			}
			if implementer != "" {
				cfg.ImplementerCommand = implementer
			}
			if verify != "" {
				cfg.VerifyCommand = verify
			}
			if inspector != "" {
				cfg.InspectorCommand = inspector
			}
			if judge != "" {
				cfg.JudgeCommand = judge
			}
			if shell != "" {
				cfg.Shell = shell
			}
			if workspace != "" {
				cfg.Workspace = workspace
			}
			if cfg.Goal == "" {
				return errors.New("goal is required (use --goal or --config)")
			}
			if cfg.VerifyCommand == "" {
				return errors.New("verify command is required (use --verify or --config)")
			}

			report, err := lab.Run(resolveLabDir(dataDirProvider(), runDirRun), cfg)
			if err != nil {
				return err
			}
			if jsonOut {
				b, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(b))
			} else {
				fmt.Printf("run_id: %s\n", report.RunID)
				fmt.Printf("run_dir: %s\n", report.RunDir)
				fmt.Printf("qualified: %v\n", report.Qualified)
				fmt.Printf("result: %s\n", report.FinalMessage)
				fmt.Printf("report: %s\n", filepath.Join(report.RunDir, "report.md"))
			}
			if !report.Qualified {
				return fmt.Errorf("not qualified")
			}
			return nil
		},
	}
	runCmd.Flags().StringVar(&configPath, "config", "", "Path to JSON config")
	runCmd.Flags().StringVar(&goal, "goal", "", "Run goal")
	runCmd.Flags().IntVar(&maxIterations, "max-iterations", 0, "Max iterations")
	runCmd.Flags().StringVar(&contextDesigner, "context-designer", "", "Context designer command")
	runCmd.Flags().StringVar(&launcher, "launcher", "", "Agent launcher command")
	runCmd.Flags().StringVar(&planner, "planner", "", "Planner command")
	runCmd.Flags().StringVar(&implementer, "implementer", "", "Implementer command")
	runCmd.Flags().StringVar(&verify, "verify", "", "Verification command (required)")
	runCmd.Flags().StringVar(&inspector, "inspector", "", "Inspector command (required)")
	runCmd.Flags().StringVar(&judge, "judge", "", "Judge command (outputs QUALIFIED / NOT_QUALIFIED)")
	runCmd.Flags().StringVar(&shell, "shell", "", "Shell executable for command execution")
	runCmd.Flags().StringVar(&workspace, "workspace", "", "Workspace to run commands in (default: cwd)")
	runCmd.Flags().BoolVar(&jsonOut, "json", false, "Emit JSON report to stdout")
	runCmd.Flags().StringVar(&runDirRun, "run-dir", "", "Lab base dir (default: <data-dir>/lab/runs)")
	labCmd.AddCommand(runCmd)

	return labCmd
}

func resolveLabDir(dataDir, runDir string) string {
	if runDir != "" {
		return runDir
	}
	return filepath.Join(dataDir, "lab", "runs")
}
