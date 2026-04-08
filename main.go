package main

import (
	"fmt"
	"os"

	"github.com/oriyn-ai/cli/cmd"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if err := cmd.Execute(version, commit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
