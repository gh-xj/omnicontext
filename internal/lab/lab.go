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
	cfg = normalizeConfig(cfg)
	if err := validateConfig(cfg); err != nil {
		return RunReport{}, err
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
			"OCX_LAB_INSPECTOR_JSON_FILE": inspectorPath,
			"OCX_LAB_INSPECTOR_LOG_FILE":  inspectorLogPath,
			"OCX_LAB_VERIFY_LOG_FILE":     verifyLogPath,
			"OCX_LAB_SESSION_REF_FILE":    sessionRefPath,
			"OCX_LAB_SESSION_PATH_FILE":   sessionPathHintPath,
		}

		iter.ContextCode = runOptionalCommand(cfg.Shell, cfg.Workspace, cfg.ContextDesigner, env, filepath.Join(iterDir, "outbox", "context-designer.log"))
		iter.ContextPack = contextPackPath
		iter.ContextHash = hashFileSHA256(contextPackPath)
		if iter.ContextHash != "" {
			env["OCX_LAB_CONTEXT_PACK_SHA256"] = iter.ContextHash
		}

		iter.LauncherCode = runOptionalCommand(cfg.Shell, cfg.Workspace, cfg.LauncherCommand, env, filepath.Join(iterDir, "outbox", "launcher.log"))

		iter.PlannerCode = runOptionalCommand(cfg.Shell, cfg.Workspace, cfg.PlannerCommand, env, filepath.Join(iterDir, "outbox", "planner.log"))

		iter.ImplementCode = runOptionalCommand(cfg.Shell, cfg.Workspace, cfg.ImplementerCommand, env, filepath.Join(iterDir, "outbox", "implementer.log"))

		vcode, vout := runShell(cfg.Shell, cfg.Workspace, cfg.VerifyCommand, env)
		iter.VerifyCode = vcode
		_ = os.WriteFile(verifyLogPath, vout, 0o644)
		sessionPath := readSessionPathHint(sessionPathHintPath)
		_ = writeSessionRef(sessionRefPath, cfg.Workspace, runID, i, sessionPath)
		icode, iout := runShell(cfg.Shell, cfg.Workspace, cfg.InspectorCommand, env)
		iter.InspectorCode = icode
		_ = os.WriteFile(inspectorLogPath, iout, 0o644)
		iter.InspectorPath = inspectorPath
		inspector, inspectorErr := parseInspectorOutput(inspectorPath)
		if inspectorErr == nil {
			iter.InspectorOK = true
			iter.Inspector = inspector.Verdict
		}

		qualified, reason := evaluateQualification(vcode, iter.InspectorCode, inspector, inspectorErr)
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

func normalizeConfig(cfg Config) Config {
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 3
	}
	if strings.TrimSpace(cfg.Shell) == "" {
		cfg.Shell = os.Getenv("SHELL")
		if cfg.Shell == "" {
			cfg.Shell = "sh"
		}
	}
	if strings.TrimSpace(cfg.Workspace) == "" {
		wd, _ := os.Getwd()
		cfg.Workspace = wd
	}
	return cfg
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Goal) == "" {
		return fmt.Errorf("goal is required")
	}
	if strings.TrimSpace(cfg.VerifyCommand) == "" {
		return fmt.Errorf("verify command is required")
	}
	if strings.TrimSpace(cfg.InspectorCommand) == "" {
		return fmt.Errorf("inspector command is required")
	}
	return nil
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

func runOptionalCommand(shell, cwd, command string, env map[string]string, logPath string) int {
	if strings.TrimSpace(command) == "" {
		return 0
	}
	code, out := runShell(shell, cwd, command, env)
	_ = os.WriteFile(logPath, out, 0o644)
	return code
}

func evaluateQualification(verifyCode, inspectorCode int, inspector inspectorOutput, inspectorErr error) (bool, string) {
	if verifyCode == 0 && inspectorCode == 0 && inspectorErr == nil && inspector.Verdict == "QUALIFIED" {
		return true, "verification and inspector qualified"
	}
	switch {
	case verifyCode != 0 && inspectorCode != 0:
		return false, fmt.Sprintf("verification failed and inspector command failed (exit %d)", inspectorCode)
	case verifyCode != 0 && inspectorErr != nil:
		return false, fmt.Sprintf("verification failed and inspector schema invalid: %v", inspectorErr)
	case verifyCode != 0:
		return false, "verification failed"
	case inspectorCode != 0:
		return false, fmt.Sprintf("inspector command failed (exit %d)", inspectorCode)
	case inspectorErr != nil:
		return false, fmt.Sprintf("inspector schema invalid: %v", inspectorErr)
	default:
		return false, fmt.Sprintf("inspector verdict %s", inspector.Verdict)
	}
}

func writeIterationSummary(iterDir string, iter IterationResult) {
	_ = os.WriteFile(filepath.Join(iterDir, "summary.md"), []byte(fmt.Sprintf(
		"# Iteration %d\n\n- Qualified: %v\n- Reason: %s\n- Context designer exit: %d\n- Launcher exit: %d\n- Planner exit: %d\n- Implementer exit: %d\n- Verify exit: %d\n- Inspector exit: %d\n- Inspector schema valid: %v\n- Inspector verdict: %s\n- Inspector file: %s\n- Context pack: %s\n- Context SHA256: %s\n",
		iter.Iteration, iter.Qualified, iter.Reason, iter.ContextCode, iter.LauncherCode, iter.PlannerCode, iter.ImplementCode, iter.VerifyCode, iter.InspectorCode, iter.InspectorOK, iter.Inspector, iter.InspectorPath, iter.ContextPack, iter.ContextHash,
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
		md = append(md, []byte(fmt.Sprintf("- Iteration %d: qualified=%v, reason=%s, verify=%d, inspector=%d\n", it.Iteration, it.Qualified, it.Reason, it.VerifyCode, it.InspectorCode))...)
	}
	if err := os.WriteFile(filepath.Join(runDir, "report.md"), md, 0o644); err != nil {
		return err
	}
	return writeReviewChecklist(runDir, report)
}

func writeReviewChecklist(runDir string, report RunReport) error {
	var b strings.Builder
	b.WriteString("# Human Review Checklist\n\n")
	b.WriteString("- Confirm goal aligns with requested scope.\n")
	b.WriteString("- Confirm last iteration is qualified by verifier + inspector.\n")
	b.WriteString("- Inspect inspector verdict/reasons/patch_hints/confidence.\n")
	b.WriteString("- Inspect verifier log for the same iteration.\n")
	b.WriteString("- Verify changed files are intentional before PR/merge.\n\n")
	if len(report.Iterations) > 0 {
		last := report.Iterations[len(report.Iterations)-1]
		iterDir := filepath.Join(runDir, fmt.Sprintf("iter-%03d", last.Iteration))
		b.WriteString("## Evidence Paths\n")
		b.WriteString("- Iteration summary: `" + filepath.Join(iterDir, "summary.md") + "`\n")
		b.WriteString("- Inspector JSON: `" + last.InspectorPath + "`\n")
		b.WriteString("- Inspector log: `" + filepath.Join(iterDir, "outbox", "inspector.log") + "`\n")
		b.WriteString("- Verifier log: `" + filepath.Join(iterDir, "outbox", "verify.log") + "`\n")
		b.WriteString("- Session ref: `" + filepath.Join(iterDir, "inbox", "session-ref.json") + "`\n")
	}
	return os.WriteFile(filepath.Join(runDir, "review-checklist.md"), []byte(b.String()), 0o644)
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
