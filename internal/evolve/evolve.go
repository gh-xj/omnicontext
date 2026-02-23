package evolve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-xj/omnicontext/internal/lab"
)

type Config struct {
	Workspace          string
	DataDir            string
	Goal               string
	MaxIterations      int
	VerifyCommand      string
	PlannerCommand     string
	ImplementerCommand string
	JudgeCommand       string
	Branch             string
	AllowDirty         bool
	AutoCommit         bool
	CommitMessage      string
	OpenDraftPR        bool
	PRTitle            string
}

type Report struct {
	RunID         string `json:"run_id"`
	RunDir        string `json:"run_dir"`
	Branch        string `json:"branch"`
	Qualified     bool   `json:"qualified"`
	CommitHash    string `json:"commit_hash,omitempty"`
	PRTitle       string `json:"pr_title"`
	PRBodyFile    string `json:"pr_body_file"`
	PRTitleFile   string `json:"pr_title_file"`
	LabReportFile string `json:"lab_report_file"`
	Message       string `json:"message"`
}

func Run(cfg Config) (Report, error) {
	if strings.TrimSpace(cfg.Goal) == "" {
		return Report{}, errors.New("goal is required")
	}
	if strings.TrimSpace(cfg.VerifyCommand) == "" {
		cfg.VerifyCommand = "go vet ./... && go test ./... && go build ./cmd/ocx"
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 3
	}
	if cfg.Workspace == "" {
		wd, _ := os.Getwd()
		cfg.Workspace = wd
	}
	if cfg.DataDir == "" {
		h, _ := os.UserHomeDir()
		cfg.DataDir = filepath.Join(h, ".ocx")
	}
	if cfg.CommitMessage == "" {
		cfg.CommitMessage = "evolve: " + strings.TrimSpace(cfg.Goal)
	}

	if err := ensureGitRepo(cfg.Workspace); err != nil {
		return Report{}, err
	}
	if !cfg.AllowDirty {
		dirty, err := isDirty(cfg.Workspace)
		if err != nil {
			return Report{}, err
		}
		if dirty {
			return Report{}, errors.New("workspace has uncommitted changes; commit/stash first or use --allow-dirty")
		}
	}

	branch := cfg.Branch
	if strings.TrimSpace(branch) == "" {
		branch = fmt.Sprintf("evolve/%s", time.Now().UTC().Format("20060102-150405"))
	}
	if err := createBranch(cfg.Workspace, branch); err != nil {
		return Report{}, err
	}

	runsBase := filepath.Join(cfg.DataDir, "evolve", "runs")
	if err := os.MkdirAll(runsBase, 0o755); err != nil {
		return Report{}, err
	}

	judge := cfg.JudgeCommand
	if strings.TrimSpace(judge) == "" {
		judge = `if grep -q "FAIL" "$OCX_LAB_ITER_DIR/outbox/verify.log"; then echo NOT_QUALIFIED; exit 1; else echo QUALIFIED; fi`
	}
	labReport, err := lab.Run(runsBase, lab.Config{
		Workspace:          cfg.Workspace,
		Goal:               cfg.Goal,
		MaxIterations:      cfg.MaxIterations,
		PlannerCommand:     cfg.PlannerCommand,
		ImplementerCommand: cfg.ImplementerCommand,
		VerifyCommand:      cfg.VerifyCommand,
		JudgeCommand:       judge,
	})
	if err != nil {
		return Report{}, err
	}

	r := Report{
		RunID:         labReport.RunID,
		RunDir:        labReport.RunDir,
		Branch:        branch,
		Qualified:     labReport.Qualified,
		LabReportFile: filepath.Join(labReport.RunDir, "report.md"),
	}
	if !labReport.Qualified {
		r.Message = "not qualified; see lab report"
		_ = writeEvolveSummary(r)
		return r, errors.New("not qualified")
	}

	changed, err := changedFiles(cfg.Workspace)
	if err != nil {
		return r, err
	}
	if len(changed) == 0 {
		r.Message = "qualified but no code changes detected"
		_ = writeEvolveSummary(r)
		return r, nil
	}

	if cfg.AutoCommit {
		hash, err := commitAll(cfg.Workspace, cfg.CommitMessage)
		if err != nil {
			return r, err
		}
		r.CommitHash = hash
	}

	prDir := filepath.Join(labReport.RunDir, "pr")
	if err := os.MkdirAll(prDir, 0o755); err != nil {
		return r, err
	}
	title := cfg.PRTitle
	if strings.TrimSpace(title) == "" {
		title = defaultPRTitle(cfg.Goal)
	}
	body := defaultPRBody(cfg.Goal, cfg.VerifyCommand, r.LabReportFile, changed, r.CommitHash)
	titleFile := filepath.Join(prDir, "pr-title.txt")
	bodyFile := filepath.Join(prDir, "pr-body.md")
	if err := os.WriteFile(titleFile, []byte(title+"\n"), 0o644); err != nil {
		return r, err
	}
	if err := os.WriteFile(bodyFile, []byte(body), 0o644); err != nil {
		return r, err
	}
	r.PRTitle = title
	r.PRTitleFile = titleFile
	r.PRBodyFile = bodyFile

	if cfg.OpenDraftPR {
		if err := createDraftPR(cfg.Workspace, title, bodyFile, branch); err != nil {
			return r, err
		}
	}

	r.Message = "qualified; PR artifacts ready for human review"
	if err := writeEvolveSummary(r); err != nil {
		return r, err
	}
	return r, nil
}

func defaultPRTitle(goal string) string {
	g := strings.TrimSpace(goal)
	if g == "" {
		return "evolve: automated quality iteration"
	}
	if len(g) > 72 {
		g = g[:72]
	}
	return "evolve: " + g
}

func defaultPRBody(goal, verifyCmd, labReport string, changed []string, commitHash string) string {
	b := &strings.Builder{}
	b.WriteString("## Summary\n")
	b.WriteString("Automated evolve loop completed and reached qualified state.\n\n")
	b.WriteString("## Goal\n")
	b.WriteString("- " + goal + "\n\n")
	b.WriteString("## What Changed\n")
	for _, f := range changed {
		b.WriteString("- " + f + "\n")
	}
	if len(changed) == 0 {
		b.WriteString("- (no file-level changes captured)\n")
	}
	b.WriteString("\n## Verification\n")
	b.WriteString("```bash\n" + verifyCmd + "\n```\n\n")
	b.WriteString("## Evidence\n")
	b.WriteString("- Lab report: `" + labReport + "`\n")
	if commitHash != "" {
		b.WriteString("- Commit: `" + commitHash + "`\n")
	}
	b.WriteString("\n## Risk & Rollback\n")
	b.WriteString("- Risk: scoped to changed files listed above.\n")
	b.WriteString("- Rollback: revert this commit/branch if regression is found.\n")
	return b.String()
}

func ensureGitRepo(workspace string) error {
	code, out, err := run(workspace, "git", "rev-parse", "--is-inside-work-tree")
	if err != nil || code != 0 {
		return fmt.Errorf("not a git repo: %s", strings.TrimSpace(out))
	}
	return nil
}

func isDirty(workspace string) (bool, error) {
	code, out, err := run(workspace, "git", "status", "--porcelain")
	if err != nil || code != 0 {
		return false, fmt.Errorf("git status failed: %s", strings.TrimSpace(out))
	}
	return strings.TrimSpace(out) != "", nil
}

func createBranch(workspace, branch string) error {
	code, out, _ := run(workspace, "git", "checkout", "-b", branch)
	if code == 0 {
		return nil
	}
	code2, out2, _ := run(workspace, "git", "checkout", branch)
	if code2 == 0 {
		return nil
	}
	return fmt.Errorf("failed to switch branch %s: %s | %s", branch, strings.TrimSpace(out), strings.TrimSpace(out2))
}

func changedFiles(workspace string) ([]string, error) {
	code, out, err := run(workspace, "git", "status", "--porcelain")
	if err != nil || code != 0 {
		return nil, fmt.Errorf("git status failed: %s", strings.TrimSpace(out))
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		files = append(files, strings.TrimSpace(line[3:]))
	}
	return files, nil
}

func commitAll(workspace, msg string) (string, error) {
	if code, out, _ := run(workspace, "git", "add", "-A"); code != 0 {
		return "", fmt.Errorf("git add failed: %s", strings.TrimSpace(out))
	}
	if code, out, _ := run(workspace, "git", "commit", "-m", msg); code != 0 {
		return "", fmt.Errorf("git commit failed: %s", strings.TrimSpace(out))
	}
	code, out, _ := run(workspace, "git", "rev-parse", "HEAD")
	if code != 0 {
		return "", fmt.Errorf("rev-parse failed: %s", strings.TrimSpace(out))
	}
	return strings.TrimSpace(out), nil
}

func createDraftPR(workspace, title, bodyFile, branch string) error {
	code, out, _ := run(workspace, "git", "push", "-u", "origin", branch)
	if code != 0 {
		return fmt.Errorf("git push failed: %s", strings.TrimSpace(out))
	}
	code, out, _ = run(workspace, "gh", "pr", "create", "--draft", "--title", title, "--body-file", bodyFile)
	if code != 0 {
		return fmt.Errorf("gh pr create failed: %s", strings.TrimSpace(out))
	}
	return nil
}

func writeEvolveSummary(r Report) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(r.RunDir, "evolve-report.json")
	mdPath := filepath.Join(r.RunDir, "evolve-report.md")
	if err := os.WriteFile(jsonPath, b, 0o644); err != nil {
		return err
	}
	md := fmt.Sprintf("# Evolve Report\n\n- Run ID: %s\n- Branch: %s\n- Qualified: %v\n- Message: %s\n- Lab report: %s\n- PR title file: %s\n- PR body file: %s\n", r.RunID, r.Branch, r.Qualified, r.Message, r.LabReportFile, r.PRTitleFile, r.PRBodyFile)
	return os.WriteFile(mdPath, []byte(md), 0o644)
}

func run(workspace string, bin string, args ...string) (int, string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = workspace
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if err == nil {
		return 0, buf.String(), nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), buf.String(), nil
	}
	return 1, buf.String(), err
}
