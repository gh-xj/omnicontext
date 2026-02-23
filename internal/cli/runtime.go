package cli

import (
	"strings"

	agentcli "github.com/gh-xj/agentcli-go"
)

// ResolveExitCode maps common usage errors to ExitUsage and delegates other
// typed errors to agentcli-go.
func ResolveExitCode(err error) int {
	if err == nil {
		return agentcli.ExitSuccess
	}
	if usageError(err) {
		return agentcli.ExitUsage
	}
	code := agentcli.ResolveExitCode(err)
	if code == agentcli.ExitSuccess {
		return 1
	}
	return code
}

func usageError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	indicators := []string{
		"unknown command",
		"unknown flag",
		"accepts",
		"requires",
		"usage:",
	}
	for _, marker := range indicators {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
