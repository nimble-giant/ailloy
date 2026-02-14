package main

import (
	"github.com/nimble-giant/ailloy/internal/commands"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	commands.SetVersionInfo(version, commit, date)
	commands.Execute()
}
