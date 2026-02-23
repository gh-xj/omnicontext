package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunQualifies(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:          ws,
		Goal:               "create marker and pass verify",
		MaxIterations:      2,
		ImplementerCommand: "touch marker.txt && echo done > \"$OCX_LAB_IMPL_FILE\"",
		VerifyCommand:      "test -f marker.txt",
		InspectorCommand:   "cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'\n{\"verdict\":\"QUALIFIED\",\"reasons\":[\"all checks passed\"],\"patch_hints\":[],\"confidence\":0.92}\nJSON",
		Shell:              "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !r.Qualified {
		t.Fatalf("expected qualified")
	}
	if len(r.Iterations) < 1 {
		t.Fatalf("expected iterations")
	}
	if _, err := os.Stat(filepath.Join(r.RunDir, "report.json")); err != nil {
		t.Fatalf("report.json missing: %v", err)
	}
}

func TestRunNotQualified(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:        ws,
		Goal:             "never pass",
		MaxIterations:    2,
		VerifyCommand:    "exit 1",
		InspectorCommand: "cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'\n{\"verdict\":\"NOT_QUALIFIED\",\"reasons\":[\"verify failed\"],\"patch_hints\":[],\"confidence\":0.60}\nJSON",
		Shell:            "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if r.Qualified {
		t.Fatalf("expected not qualified")
	}
	if len(r.Iterations) != 2 {
		t.Fatalf("expected 2 iterations, got %d", len(r.Iterations))
	}
}

func TestRunQualifiedRequiresVerifierAndInspectorQualified(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:        ws,
		Goal:             "gate by inspector",
		MaxIterations:    1,
		VerifyCommand:    "exit 0",
		InspectorCommand: "cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'\n{\"verdict\":\"QUALIFIED\",\"reasons\":[\"ok\"],\"patch_hints\":[],\"confidence\":0.95}\nJSON",
		Shell:            "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !r.Qualified {
		t.Fatalf("expected qualified when verifier passes and inspector verdict is QUALIFIED")
	}
}

func TestRunFailsWhenInspectorNotQualifiedEvenIfVerifierPasses(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:        ws,
		Goal:             "inspector rejection should fail",
		MaxIterations:    1,
		VerifyCommand:    "exit 0",
		InspectorCommand: "cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'\n{\"verdict\":\"NOT_QUALIFIED\",\"reasons\":[\"missing tests\"],\"patch_hints\":[\"add unit tests\"],\"confidence\":0.80}\nJSON",
		Shell:            "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if r.Qualified {
		t.Fatalf("expected not qualified when inspector verdict is NOT_QUALIFIED")
	}
	if len(r.Iterations) != 1 {
		t.Fatalf("expected 1 iteration, got %d", len(r.Iterations))
	}
	if !strings.Contains(strings.ToLower(r.Iterations[0].Reason), "inspector") {
		t.Fatalf("expected reason to mention inspector, got %q", r.Iterations[0].Reason)
	}
}

func TestRunWritesNextStepHintsForNextIterationOnFail(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:        ws,
		Goal:             "carry hints forward",
		MaxIterations:    2,
		VerifyCommand:    "exit 0",
		InspectorCommand: "cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'\n{\"verdict\":\"NOT_QUALIFIED\",\"reasons\":[\"missing checks\"],\"patch_hints\":[\"add parser validation\",\"update merge condition\"],\"confidence\":0.71}\nJSON",
		Shell:            "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(r.Iterations) != 2 {
		t.Fatalf("expected 2 iterations, got %d", len(r.Iterations))
	}
	hintsPath := filepath.Join(r.RunDir, "iter-002", "inbox", "next-step-hints.md")
	hints, err := os.ReadFile(hintsPath)
	if err != nil {
		t.Fatalf("expected next-step-hints.md in second iteration inbox: %v", err)
	}
	text := string(hints)
	if !strings.Contains(text, "add parser validation") || !strings.Contains(text, "update merge condition") {
		t.Fatalf("expected patch hints to be present, got %q", text)
	}
}

func TestRunMissingInspectorSchemaIsClearReason(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:        ws,
		Goal:             "missing inspector payload",
		MaxIterations:    1,
		VerifyCommand:    "exit 0",
		InspectorCommand: "true",
		Shell:            "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if r.Qualified {
		t.Fatalf("expected not qualified when inspector schema is missing")
	}
	if !strings.Contains(strings.ToLower(r.Iterations[0].Reason), "inspector schema invalid") {
		t.Fatalf("expected clear inspector schema reason, got %q", r.Iterations[0].Reason)
	}
}

func TestRunInspectorArtifactsAndSummary(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:          ws,
		Goal:               "inspector emits artifacts",
		MaxIterations:      1,
		ImplementerCommand: "touch marker.txt",
		VerifyCommand:      "test -f marker.txt",
		InspectorCommand:   "test -f \"$OCX_LAB_VERIFY_LOG_FILE\" && test -f \"$OCX_LAB_SESSION_REF_FILE\" && echo inspector-ok && cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'\n{\"verdict\":\"QUALIFIED\",\"reasons\":[\"protocol files present\"],\"patch_hints\":[],\"confidence\":0.99}\nJSON",
		Shell:              "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !r.Qualified {
		t.Fatalf("expected qualified")
	}

	iterDir := filepath.Join(r.RunDir, "iter-001")
	for _, artifact := range []string{
		filepath.Join(iterDir, "outbox", "inspector.log"),
		filepath.Join(iterDir, "outbox", "inspector.json"),
	} {
		if _, err := os.Stat(artifact); err != nil {
			t.Fatalf("expected artifact %s: %v", artifact, err)
		}
	}

	summaryPath := filepath.Join(iterDir, "summary.md")
	summary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	text := string(summary)
	if !strings.Contains(text, "- Inspector exit: 0") {
		t.Fatalf("expected inspector exit code in summary, got %q", text)
	}
	if !strings.Contains(text, "- Inspector verdict: QUALIFIED") {
		t.Fatalf("expected inspector verdict in summary, got %q", text)
	}
}

func TestRunInspectorCommandWritesJSONAtInspectorPath(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:     ws,
		Goal:          "inspector writes structured JSON",
		MaxIterations: 1,
		VerifyCommand: "exit 0",
		InspectorCommand: `cat > "$OCX_LAB_INSPECTOR_JSON_FILE" <<'JSON'
{"verdict":"QUALIFIED","reasons":["json emitted"],"patch_hints":[],"confidence":0.93}
JSON`,
		Shell: "sh",
	}

	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !r.Qualified {
		t.Fatalf("expected qualified")
	}
	if len(r.Iterations) != 1 {
		t.Fatalf("expected 1 iteration, got %d", len(r.Iterations))
	}

	iter := r.Iterations[0]
	if iter.InspectorPath == "" {
		t.Fatalf("expected inspector path to be recorded")
	}
	b, err := os.ReadFile(iter.InspectorPath)
	if err != nil {
		t.Fatalf("read inspector json: %v", err)
	}
	var payload struct {
		Verdict    string   `json:"verdict"`
		Reasons    []string `json:"reasons"`
		PatchHints []string `json:"patch_hints"`
		Confidence float64  `json:"confidence"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("parse inspector json: %v", err)
	}
	if payload.Verdict != "QUALIFIED" {
		t.Fatalf("unexpected verdict: %q", payload.Verdict)
	}
	if len(payload.Reasons) != 1 || payload.Reasons[0] != "json emitted" {
		t.Fatalf("unexpected reasons: %+v", payload.Reasons)
	}
	if iter.Inspector != "QUALIFIED" || !iter.InspectorOK {
		t.Fatalf("expected parsed inspector verdict in iteration: verdict=%q ok=%v", iter.Inspector, iter.InspectorOK)
	}

	reportPath := filepath.Join(r.RunDir, "report.json")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report.json: %v", err)
	}
	var report struct {
		Iterations []struct {
			InspectorPath string `json:"inspector_path"`
		} `json:"iterations"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse report.json: %v", err)
	}
	if len(report.Iterations) != 1 {
		t.Fatalf("expected 1 iteration in report.json, got %d", len(report.Iterations))
	}
	if report.Iterations[0].InspectorPath != iter.InspectorPath {
		t.Fatalf("inspector path mismatch: report=%q iteration=%q", report.Iterations[0].InspectorPath, iter.InspectorPath)
	}
}

func TestRunSessionRefArtifactPerIteration(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:        ws,
		Goal:             "write session ref each iteration",
		MaxIterations:    2,
		VerifyCommand:    "exit 1",
		InspectorCommand: "cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'\n{\"verdict\":\"NOT_QUALIFIED\",\"reasons\":[\"verify failed\"],\"patch_hints\":[],\"confidence\":0.61}\nJSON",
		Shell:            "sh",
	}
	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(r.Iterations) != 2 {
		t.Fatalf("expected 2 iterations, got %d", len(r.Iterations))
	}

	for i := 1; i <= 2; i++ {
		refPath := filepath.Join(r.RunDir, fmt.Sprintf("iter-%03d", i), "inbox", "session-ref.json")
		b, err := os.ReadFile(refPath)
		if err != nil {
			t.Fatalf("read %s: %v", refPath, err)
		}
		var raw map[string]any
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatalf("unmarshal raw %s: %v", refPath, err)
		}
		if _, ok := raw["session_path"]; !ok {
			t.Fatalf("expected session_path key in %s", refPath)
		}
		var payload struct {
			Workspace   string `json:"workspace"`
			RunID       string `json:"run_id"`
			Iteration   int    `json:"iteration"`
			SessionPath string `json:"session_path"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			t.Fatalf("unmarshal %s: %v", refPath, err)
		}
		if payload.Workspace != ws {
			t.Fatalf("workspace mismatch for iteration %d: got %q want %q", i, payload.Workspace, ws)
		}
		if payload.RunID != r.RunID {
			t.Fatalf("run_id mismatch for iteration %d: got %q want %q", i, payload.RunID, r.RunID)
		}
		if payload.Iteration != i {
			t.Fatalf("iteration mismatch: got %d want %d", payload.Iteration, i)
		}
		if payload.SessionPath != "" {
			t.Fatalf("expected empty session_path placeholder, got %q", payload.SessionPath)
		}
	}
}

func TestRunSessionRefIncludesSessionPathFromLauncherHint(t *testing.T) {
	base := t.TempDir()
	ws := t.TempDir()
	cfg := Config{
		Workspace:       ws,
		Goal:            "session ref includes launcher hint path",
		MaxIterations:   1,
		LauncherCommand: `echo "$OCX_LAB_ITER_DIR/outbox/agent-session.json" > "$OCX_LAB_SESSION_PATH_FILE"`,
		VerifyCommand:   "exit 0",
		InspectorCommand: `grep -q '"session_path"[[:space:]]*:[[:space:]]*"[^"]\+"' "$OCX_LAB_SESSION_REF_FILE" && cat > "$OCX_LAB_INSPECTOR_JSON_FILE" <<'JSON'
{"verdict":"QUALIFIED","reasons":["session path propagated"],"patch_hints":[],"confidence":0.91}
JSON`,
		Shell: "sh",
	}

	r, err := Run(base, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !r.Qualified {
		t.Fatalf("expected qualified")
	}

	refPath := filepath.Join(r.RunDir, "iter-001", "inbox", "session-ref.json")
	b, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("read session-ref: %v", err)
	}
	var payload struct {
		SessionPath string `json:"session_path"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("unmarshal session-ref: %v", err)
	}
	want := filepath.Join(r.RunDir, "iter-001", "outbox", "agent-session.json")
	if payload.SessionPath != want {
		t.Fatalf("session_path mismatch: got %q want %q", payload.SessionPath, want)
	}
}
