package evolve

import (
	"strings"
	"testing"
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
