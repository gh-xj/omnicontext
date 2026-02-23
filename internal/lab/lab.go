package lab

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	ContextDesigner    string `json:"context_designer_command"`
	LauncherCommand    string `json:"launcher_command"`
	PlannerCommand     string `json:"planner_command"`
	ImplementerCommand string `json:"implementer_command"`
	VerifyCommand      string `json:"verify_command"`
	InspectorCommand   string `json:"inspector_command"`
	JudgeCommand       string `json:"judge_command"`
	Shell              string `json:"shell"`
}

type IterationResult struct {
	Iteration     int       `json:"iteration"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	ContextCode   int       `json:"context_designer_code"`
	LauncherCode  int       `json:"launcher_code"`
	PlannerCode   int       `json:"planner_code"`
	ImplementCode int       `json:"implement_code"`
	VerifyCode    int       `json:"verify_code"`
	InspectorCode int       `json:"inspector_code"`
	JudgeCode     int       `json:"judge_code"`
	ContextPack   string    `json:"context_pack"`
	ContextHash   string    `json:"context_pack_sha256"`
	InspectorPath string    `json:"inspector_path"`
	InspectorOK   bool      `json:"inspector_ok"`
	Inspector     string    `json:"inspector_verdict"`
	Qualified     bool      `json:"qualified"`
	Reason        string    `json:"reason"`
}

type inspectorOutput struct {
	Verdict    string   `json:"verdict"`
	Reasons    []string `json:"reasons"`
	PatchHints []string `json:"patch_hints"`
	Confidence float64  `json:"confidence"`
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
	var nextStepHints []string

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
		if len(nextStepHints) > 0 {
			_ = writeNextStepHints(filepath.Join(iterDir, "inbox", "next-step-hints.md"), nextStepHints)
		}
		contextPackPath := filepath.Join(iterDir, "inbox", "context-pack.md")
		_ = os.WriteFile(contextPackPath, []byte(defaultContextPack(cfg.Goal, filepath.Join(iterDir, "inbox", "constraints.md"))), 0o644)
		verifyLogPath := filepath.Join(iterDir, "outbox", "verify.log")
		inspectorLogPath := filepath.Join(iterDir, "outbox", "inspector.log")
		inspectorPath := filepath.Join(iterDir, "outbox", "inspector.json")
		sessionRefPath := filepath.Join(iterDir, "inbox", "session-ref.json")
		sessionPathHintPath := filepath.Join(iterDir, "outbox", "session-path.txt")

		env := map[string]string{
			"OCX_LAB_RUN_DIR":             runDir,
			"OCX_LAB_ITER_DIR":            iterDir,
			"OCX_LAB_WORKSPACE":           cfg.Workspace,
			"OCX_LAB_GOAL_FILE":           filepath.Join(iterDir, "inbox", "goal.md"),
			"OCX_LAB_CONSTRAINTS_FILE":    filepath.Join(iterDir, "inbox", "constraints.md"),
			"OCX_LAB_CONTEXT_PACK_FILE":   contextPackPath,
			"OCX_LAB_PLAN_FILE":           filepath.Join(iterDir, "outbox", "planner.md"),
			"OCX_LAB_IMPL_FILE":           filepath.Join(iterDir, "outbox", "implementer.md"),
			"OCX_LAB_JUDGE_FILE":          filepath.Join(iterDir, "outbox", "judge.md"),
			"OCX_LAB_INSPECTOR_JSON_FILE": inspectorPath,
			"OCX_LAB_INSPECTOR_LOG_FILE":  inspectorLogPath,
			"OCX_LAB_VERIFY_LOG_FILE":     verifyLogPath,
			"OCX_LAB_SESSION_REF_FILE":    sessionRefPath,
			"OCX_LAB_SESSION_PATH_FILE":   sessionPathHintPath,
		}

		if strings.TrimSpace(cfg.ContextDesigner) != "" {
			code, out := runShell(cfg.Shell, cfg.Workspace, cfg.ContextDesigner, env)
			iter.ContextCode = code
			_ = os.WriteFile(filepath.Join(iterDir, "outbox", "context-designer.log"), out, 0o644)
		}
		iter.ContextPack = contextPackPath
		iter.ContextHash = hashFileSHA256(contextPackPath)
		if iter.ContextHash != "" {
			env["OCX_LAB_CONTEXT_PACK_SHA256"] = iter.ContextHash
		}

		if strings.TrimSpace(cfg.LauncherCommand) != "" {
			code, out := runShell(cfg.Shell, cfg.Workspace, cfg.LauncherCommand, env)
			iter.LauncherCode = code
			_ = os.WriteFile(filepath.Join(iterDir, "outbox", "launcher.log"), out, 0o644)
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
		_ = os.WriteFile(verifyLogPath, vout, 0o644)
		sessionPath := readSessionPathHint(sessionPathHintPath)
		_ = writeSessionRef(sessionRefPath, cfg.Workspace, runID, i, sessionPath)
		if strings.TrimSpace(cfg.InspectorCommand) != "" {
			icode, iout := runShell(cfg.Shell, cfg.Workspace, cfg.InspectorCommand, env)
			iter.InspectorCode = icode
			_ = os.WriteFile(inspectorLogPath, iout, 0o644)
		} else {
			_ = os.WriteFile(inspectorLogPath, []byte("inspector command not configured\n"), 0o644)
		}
		iter.InspectorPath = inspectorPath
		inspector, inspectorErr := parseInspectorOutput(inspectorPath)
		if inspectorErr == nil {
			iter.InspectorOK = true
			iter.Inspector = inspector.Verdict
		}

		qualified := false
		reason := "verification failed"
		if strings.TrimSpace(cfg.JudgeCommand) != "" {
			jcode, jout := runShell(cfg.Shell, cfg.Workspace, cfg.JudgeCommand, env)
			iter.JudgeCode = jcode
			_ = os.WriteFile(filepath.Join(iterDir, "outbox", "judge.log"), jout, 0o644)
		}

		if vcode == 0 && iter.InspectorCode == 0 && inspectorErr == nil && inspector.Verdict == "QUALIFIED" {
			qualified = true
			reason = "verification and inspector qualified"
		} else {
			switch {
			case vcode != 0 && iter.InspectorCode != 0:
				reason = fmt.Sprintf("verification failed and inspector command failed (exit %d)", iter.InspectorCode)
			case vcode != 0 && inspectorErr != nil:
				reason = fmt.Sprintf("verification failed and inspector schema invalid: %v", inspectorErr)
			case vcode != 0:
				reason = "verification failed"
			case iter.InspectorCode != 0:
				reason = fmt.Sprintf("inspector command failed (exit %d)", iter.InspectorCode)
			case inspectorErr != nil:
				reason = fmt.Sprintf("inspector schema invalid: %v", inspectorErr)
			case inspector.Verdict != "QUALIFIED":
				reason = fmt.Sprintf("inspector verdict %s", inspector.Verdict)
			}
		}
		if !qualified && inspectorErr == nil && len(inspector.PatchHints) > 0 {
			nextStepHints = append([]string{}, inspector.PatchHints...)
		} else {
			nextStepHints = nil
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
		"# Iteration %d\n\n- Qualified: %v\n- Reason: %s\n- Context designer exit: %d\n- Launcher exit: %d\n- Planner exit: %d\n- Implementer exit: %d\n- Verify exit: %d\n- Inspector exit: %d\n- Judge exit: %d\n- Inspector schema valid: %v\n- Inspector verdict: %s\n- Inspector file: %s\n- Context pack: %s\n- Context SHA256: %s\n",
		iter.Iteration, iter.Qualified, iter.Reason, iter.ContextCode, iter.LauncherCode, iter.PlannerCode, iter.ImplementCode, iter.VerifyCode, iter.InspectorCode, iter.JudgeCode, iter.InspectorOK, iter.Inspector, iter.InspectorPath, iter.ContextPack, iter.ContextHash,
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

func hashFileSHA256(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func defaultContextPack(goal, constraintsPath string) string {
	return fmt.Sprintf("# Context Pack\n\n## Goal\n%s\n\n## Constraints File\n%s\n\n## Acceptance\n- Verification command exits 0.\n- Inspector emits QUALIFIED with valid schema.\n", goal, constraintsPath)
}

func parseInspectorOutput(path string) (inspectorOutput, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return inspectorOutput{}, fmt.Errorf("read %s: %w", path, err)
	}
	var out inspectorOutput
	if err := json.Unmarshal(b, &out); err != nil {
		return inspectorOutput{}, fmt.Errorf("parse %s: %w", path, err)
	}
	out.Verdict = strings.ToUpper(strings.TrimSpace(out.Verdict))
	switch out.Verdict {
	case "QUALIFIED", "NOT_QUALIFIED":
	default:
		return inspectorOutput{}, fmt.Errorf("verdict must be QUALIFIED or NOT_QUALIFIED")
	}
	if out.Reasons == nil {
		return inspectorOutput{}, fmt.Errorf("reasons is required")
	}
	if out.PatchHints == nil {
		return inspectorOutput{}, fmt.Errorf("patch_hints is required")
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		return inspectorOutput{}, fmt.Errorf("confidence must be between 0 and 1")
	}
	return out, nil
}

func writeNextStepHints(path string, hints []string) error {
	var b strings.Builder
	b.WriteString("# Next Step Hints\n\n")
	for _, hint := range hints {
		trimmed := strings.TrimSpace(hint)
		if trimmed == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(trimmed)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeSessionRef(path, workspace, runID string, iteration int, sessionPath string) error {
	payload := struct {
		Workspace   string `json:"workspace"`
		RunID       string `json:"run_id"`
		Iteration   int    `json:"iteration"`
		SessionPath string `json:"session_path"`
	}{
		Workspace:   workspace,
		RunID:       runID,
		Iteration:   iteration,
		SessionPath: sessionPath,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func readSessionPathHint(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
