package evolve

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultPRTitle(t *testing.T) {
	title := defaultPRTitle("Fix flaky tests in ingest")
	if title == "" || title[:8] != "evolve: " {
		t.Fatalf("unexpected title: %q", title)
	}
}

func TestDefaultPRBody(t *testing.T) {
	body := defaultPRBody("Improve parser", "go test ./...", "/tmp/report.md", []string{"a.go", "b.go"}, "abc123")
	if body == "" {
		t.Fatal("body empty")
	}
	mustContain := []string{"Improve parser", "go test ./...", "a.go", "abc123", "Risk & Rollback"}
	for _, m := range mustContain {
		if !strings.Contains(body, m) {
			t.Fatalf("body missing %q", m)
		}
	}
}

func TestRunRequiresInspectorCommand(t *testing.T) {
	_, err := Run(Config{
		Goal:          "require inspector",
		VerifyCommand: "exit 0",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "inspector command is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPassesInspectorCommandToLab(t *testing.T) {
	workspace := t.TempDir()
	dataDir := t.TempDir()

	runGit(t, workspace, "init")
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, workspace, "add", "README.md")
	runGit(t, workspace, "commit", "-m", "init")

	inspectorCmd := `cat > "$OCX_LAB_INSPECTOR_JSON_FILE" <<'JSON'
{"verdict":"QUALIFIED","reasons":["ok"],"patch_hints":[],"confidence":0.95}
JSON`
	r, err := Run(Config{
		Workspace:        workspace,
		DataDir:          dataDir,
		Goal:             "prove inspector passthrough",
		MaxIterations:    1,
		VerifyCommand:    "exit 0",
		InspectorCommand: inspectorCmd,
		AutoCommit:       false,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !r.Qualified {
		t.Fatalf("expected qualified")
	}
	if r.RunDir == "" {
		t.Fatalf("expected run dir")
	}
	if _, err := os.Stat(filepath.Join(r.RunDir, "iter-001", "outbox", "inspector.json")); err != nil {
		t.Fatalf("expected inspector artifact: %v", err)
	}
}

func TestRunPRHandoffIncludesInspectorEvidencePointer(t *testing.T) {
	workspace := t.TempDir()
	dataDir := t.TempDir()

	runGit(t, workspace, "init")
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, workspace, "add", "README.md")
	runGit(t, workspace, "commit", "-m", "init")

	inspectorCmd := `cat > "$OCX_LAB_INSPECTOR_JSON_FILE" <<'JSON'
{"verdict":"QUALIFIED","reasons":["inspection complete"],"patch_hints":[],"confidence":0.97}
JSON`
	branch := fmt.Sprintf("evolve-test-%d", time.Now().UnixNano())
	r, err := Run(Config{
		Workspace:          workspace,
		DataDir:            dataDir,
		Goal:               "create PR handoff with inspector evidence",
		MaxIterations:      1,
		VerifyCommand:      "exit 0",
		InspectorCommand:   inspectorCmd,
		ImplementerCommand: "echo 'updated by evolve test' >> README.md",
		AutoCommit:         false,
		Branch:             branch,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !r.Qualified {
		t.Fatalf("expected qualified")
	}
	if r.PRBodyFile == "" {
		t.Fatalf("expected PR body artifact")
	}

	prBody, err := os.ReadFile(r.PRBodyFile)
	if err != nil {
		t.Fatalf("read pr body: %v", err)
	}
	bodyText := string(prBody)
	if !strings.Contains(bodyText, "Lab report:") || !strings.Contains(bodyText, r.LabReportFile) {
		t.Fatalf("expected PR body to include lab report pointer, got %q", bodyText)
	}

	reportJSONPath := filepath.Join(r.RunDir, "report.json")
	b, err := os.ReadFile(reportJSONPath)
	if err != nil {
		t.Fatalf("read lab report json: %v", err)
	}
	var labReport struct {
		Iterations []struct {
			InspectorPath string `json:"inspector_path"`
		} `json:"iterations"`
	}
	if err := json.Unmarshal(b, &labReport); err != nil {
		t.Fatalf("parse lab report json: %v", err)
	}
	if len(labReport.Iterations) != 1 {
		t.Fatalf("expected 1 iteration in lab report json, got %d", len(labReport.Iterations))
	}
	inspectorPath := labReport.Iterations[0].InspectorPath
	if inspectorPath == "" {
		t.Fatalf("expected inspector evidence path in lab report json")
	}
	inspectorRaw, err := os.ReadFile(inspectorPath)
	if err != nil {
		t.Fatalf("read inspector evidence: %v", err)
	}
	if !strings.Contains(string(inspectorRaw), "\"verdict\":\"QUALIFIED\"") {
		t.Fatalf("expected inspector evidence content to include QUALIFIED verdict, got %q", string(inspectorRaw))
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
