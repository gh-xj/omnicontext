package main

import (
	"fmt"
	"os"

	"github.com/gh-xj/omnicontext/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ResolveExitCode(err))
	}
}
