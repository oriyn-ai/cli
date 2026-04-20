package main

import (
	"os"

	"github.com/oriyn-ai/cli/cmd"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	os.Exit(cmd.Execute(version, commit, SkillFiles))
}
