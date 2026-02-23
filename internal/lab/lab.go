package lab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Workspace          string `json:"workspace"`
	Goal               string `json:"goal"`
	MaxIterations      int    `json:"max_iterations"`
	PlannerCommand     string `json:"planner_command"`
	ImplementerCommand string `json:"implementer_command"`
	VerifyCommand      string `json:"verify_command"`
	JudgeCommand       string `json:"judge_command"`
	Shell              string `json:"shell"`
}

type IterationResult struct {
	Iteration     int       `json:"iteration"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	PlannerCode   int       `json:"planner_code"`
	ImplementCode int       `json:"implement_code"`
	VerifyCode    int       `json:"verify_code"`
	JudgeCode     int       `json:"judge_code"`
	Qualified     bool      `json:"qualified"`
	Reason        string    `json:"reason"`
}

type RunReport struct {
	RunID        string            `json:"run_id"`
	RunDir       string            `json:"run_dir"`
	StartedAt    time.Time         `json:"started_at"`
	FinishedAt   time.Time         `json:"finished_at"`
	Qualified    bool              `json:"qualified"`
	Goal         string            `json:"goal"`
	Iterations   []IterationResult `json:"iterations"`
	Config       Config            `json:"config"`
	FinalMessage string            `json:"final_message"`
}

func Run(baseDir string, cfg Config) (RunReport, error) {
	if strings.TrimSpace(cfg.Goal) == "" {
		return RunReport{}, fmt.Errorf("goal is required")
	}
	if strings.TrimSpace(cfg.VerifyCommand) == "" {
		return RunReport{}, fmt.Errorf("verify command is required")
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 3
	}
	if cfg.Shell == "" {
		cfg.Shell = os.Getenv("SHELL")
		if cfg.Shell == "" {
			cfg.Shell = "sh"
		}
	}
	if cfg.Workspace == "" {
		wd, _ := os.Getwd()
		cfg.Workspace = wd
	}

	runID := time.Now().UTC().Format("20060102-150405")
	runDir := filepath.Join(baseDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return RunReport{}, err
	}

	report := RunReport{
		RunID:     runID,
		RunDir:    runDir,
		StartedAt: time.Now().UTC(),
		Goal:      cfg.Goal,
		Config:    cfg,
	}

	for i := 1; i <= cfg.MaxIterations; i++ {
		iter := IterationResult{Iteration: i, StartedAt: time.Now().UTC()}
		iterDir := filepath.Join(runDir, fmt.Sprintf("iter-%03d", i))
		if err := os.MkdirAll(filepath.Join(iterDir, "inbox"), 0o755); err != nil {
			return report, err
		}
		if err := os.MkdirAll(filepath.Join(iterDir, "outbox"), 0o755); err != nil {
			return report, err
		}

		_ = os.WriteFile(filepath.Join(iterDir, "inbox", "goal.md"), []byte(cfg.Goal+"\n"), 0o644)
		_ = os.WriteFile(filepath.Join(iterDir, "inbox", "constraints.md"), []byte("Keep changes minimal, pass verification, and stop only when qualified.\n"), 0o644)

		env := map[string]string{
			"OCX_LAB_RUN_DIR":    runDir,
			"OCX_LAB_ITER_DIR":   iterDir,
			"OCX_LAB_WORKSPACE":  cfg.Workspace,
			"OCX_LAB_GOAL_FILE":  filepath.Join(iterDir, "inbox", "goal.md"),
			"OCX_LAB_PLAN_FILE":  filepath.Join(iterDir, "outbox", "planner.md"),
			"OCX_LAB_IMPL_FILE":  filepath.Join(iterDir, "outbox", "implementer.md"),
			"OCX_LAB_JUDGE_FILE": filepath.Join(iterDir, "outbox", "judge.md"),
		}

		if strings.TrimSpace(cfg.PlannerCommand) != "" {
			code, out := runShell(cfg.Shell, cfg.Workspace, cfg.PlannerCommand, env)
			iter.PlannerCode = code
			_ = os.WriteFile(filepath.Join(iterDir, "outbox", "planner.log"), out, 0o644)
		}

		if strings.TrimSpace(cfg.ImplementerCommand) != "" {
			code, out := runShell(cfg.Shell, cfg.Workspace, cfg.ImplementerCommand, env)
			iter.ImplementCode = code
			_ = os.WriteFile(filepath.Join(iterDir, "outbox", "implementer.log"), out, 0o644)
		}

		vcode, vout := runShell(cfg.Shell, cfg.Workspace, cfg.VerifyCommand, env)
		iter.VerifyCode = vcode
		_ = os.WriteFile(filepath.Join(iterDir, "outbox", "verify.log"), vout, 0o644)

		qualified := false
		reason := "verification failed"
		if strings.TrimSpace(cfg.JudgeCommand) != "" {
			jcode, jout := runShell(cfg.Shell, cfg.Workspace, cfg.JudgeCommand, env)
			iter.JudgeCode = jcode
			_ = os.WriteFile(filepath.Join(iterDir, "outbox", "judge.log"), jout, 0o644)
			if jcode == 0 && strings.Contains(strings.ToUpper(string(jout)), "QUALIFIED") {
				qualified = true
				reason = "judge accepted"
			} else if vcode == 0 {
				reason = "verification passed but judge rejected"
			} else {
				reason = "verification and judge failed"
			}
		} else if vcode == 0 {
			qualified = true
			reason = "verification passed"
		}

		iter.Qualified = qualified
		iter.Reason = reason
		iter.FinishedAt = time.Now().UTC()
		report.Iterations = append(report.Iterations, iter)
		writeIterationSummary(iterDir, iter)

		if qualified {
			report.Qualified = true
			report.FinalMessage = fmt.Sprintf("qualified in %d iteration(s)", i)
			break
		}
	}

	if !report.Qualified {
		report.FinalMessage = fmt.Sprintf("not qualified after %d iteration(s)", cfg.MaxIterations)
	}
	report.FinishedAt = time.Now().UTC()
	if err := writeRunReport(runDir, report); err != nil {
		return report, err
	}
	return report, nil
}

func runShell(shell, cwd, command string, extraEnv map[string]string) (int, []byte) {
	cmd := exec.Command(shell, "-lc", command)
	cmd.Dir = cwd
	env := os.Environ()
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	err := cmd.Run()
	if err == nil {
		return 0, b.Bytes()
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), b.Bytes()
	}
	return 1, b.Bytes()
}

func writeIterationSummary(iterDir string, iter IterationResult) {
	_ = os.WriteFile(filepath.Join(iterDir, "summary.md"), []byte(fmt.Sprintf(
		"# Iteration %d\n\n- Qualified: %v\n- Reason: %s\n- Planner exit: %d\n- Implementer exit: %d\n- Verify exit: %d\n- Judge exit: %d\n",
		iter.Iteration, iter.Qualified, iter.Reason, iter.PlannerCode, iter.ImplementCode, iter.VerifyCode, iter.JudgeCode,
	)), 0o644)
}

func writeRunReport(runDir string, report RunReport) error {
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "report.json"), b, 0o644); err != nil {
		return err
	}
	md := []byte(fmt.Sprintf("# OCX Lab Report\n\n- Run ID: %s\n- Goal: %s\n- Qualified: %v\n- Result: %s\n\n## Iterations\n", report.RunID, report.Goal, report.Qualified, report.FinalMessage))
	for _, it := range report.Iterations {
		md = append(md, []byte(fmt.Sprintf("- Iteration %d: qualified=%v, reason=%s, verify=%d, judge=%d\n", it.Iteration, it.Qualified, it.Reason, it.VerifyCode, it.JudgeCode))...)
	}
	return os.WriteFile(filepath.Join(runDir, "report.md"), md, 0o644)
}
