package main

import (
	"fmt"
	"os"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/omnicontext/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code := agentcli.ResolveExitCode(err)
		if code == 0 {
			code = 1
		}
		os.Exit(code)
	}
}
