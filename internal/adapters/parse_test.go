package adapters

import (
	"path/filepath"
	"testing"
)

func TestParseClaudeFile(t *testing.T) {
	p := filepath.Join("testdata", "claude_sample.jsonl")
	s, err := parseClaudeFile(p)
	if err != nil {
		t.Fatalf("parseClaudeFile: %v", err)
	}
	if s.SessionID != "claude-session-1" {
		t.Fatalf("session id = %q", s.SessionID)
	}
	if s.WorkspacePath != "/tmp/project-a" {
		t.Fatalf("workspace = %q", s.WorkspacePath)
	}
	if len(s.Turns) != 1 {
		t.Fatalf("turn count = %d", len(s.Turns))
	}
	if s.Turns[0].UserMessage != "Help me design this" || s.Turns[0].AssistantSummary != "Sure, let's do it." {
		t.Fatalf("unexpected turn: %+v", s.Turns[0])
	}
}

func TestParseCodexFile(t *testing.T) {
	p := filepath.Join("testdata", "codex_sample.jsonl")
	s, err := parseCodexFile(p)
	if err != nil {
		t.Fatalf("parseCodexFile: %v", err)
	}
	if s.SessionID != "codex-session-1" {
		t.Fatalf("session id = %q", s.SessionID)
	}
	if s.WorkspacePath != "/tmp/project-b" {
		t.Fatalf("workspace = %q", s.WorkspacePath)
	}
	if len(s.Turns) != 1 {
		t.Fatalf("turn count = %d", len(s.Turns))
	}
	if s.Turns[0].UserMessage != "Hi" || s.Turns[0].AssistantSummary != "Hello there" {
		t.Fatalf("unexpected turn: %+v", s.Turns[0])
	}
}

func TestParseDir(t *testing.T) {
	sessions, err := Parse("claude", "testdata")
	if err != nil {
		t.Fatalf("Parse dir: %v", err)
	}
	if len(sessions) < 1 {
		t.Fatalf("expected sessions")
	}
}

func TestParseCodexVariantFile(t *testing.T) {
	p := filepath.Join("testdata", "codex_variant_sample.jsonl")
	s, err := parseCodexFile(p)
	if err != nil {
		t.Fatalf("parseCodexFile variant: %v", err)
	}
	if s.SessionID != "codex-variant-1" {
		t.Fatalf("session id = %q", s.SessionID)
	}
	if s.WorkspacePath != "/tmp/variant" {
		t.Fatalf("workspace = %q", s.WorkspacePath)
	}
	if len(s.Turns) < 2 {
		t.Fatalf("expected >=2 turns, got %d", len(s.Turns))
	}
	found := false
	for _, tr := range s.Turns {
		if tr.UserMessage == "user via input_text" && tr.AssistantSummary == "assistant via output_text" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("did not find expected input_text/output_text turn: %+v", s.Turns)
	}
}
