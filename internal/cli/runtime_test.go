package cli

import (
	"errors"
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
)

func TestResolveExitCodeUsageError(t *testing.T) {
	err := errors.New("unknown command \"wat\" for \"ocx\"")
	if got := ResolveExitCode(err); got != agentcli.ExitUsage {
		t.Fatalf("ResolveExitCode() = %d, want usage code %d", got, agentcli.ExitUsage)
	}
}

func TestResolveExitCodeGenericError(t *testing.T) {
	err := errors.New("some runtime failure")
	if got := ResolveExitCode(err); got != 1 {
		t.Fatalf("ResolveExitCode() = %d, want 1", got)
	}
}
