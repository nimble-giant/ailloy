package commands

import (
	"github.com/spf13/cobra"
)

// Verb-first top-level commands that enable bidirectional noun-verb / verb-noun
// ordering. For example, both "ailloy foundry search" and "ailloy search foundry"
// invoke the same handler.

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for resources",
	Long: `Search for molds, ingots, and other resources.

Available subcommands:
  foundry    Search foundries for molds and ingots`,
}

var searchFoundrySubCmd = &cobra.Command{
	Use:   "foundry <query>",
	Short: "Search for molds and ingots",
	Args:  cobra.ExactArgs(1),
	RunE:  runFoundrySearch,
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add resources",
	Long: `Add foundries, ingots, and other resources.

Available subcommands:
  foundry    Register a foundry index URL
  ingot      Download and register an ingot`,
}

var addFoundrySubCmd = &cobra.Command{
	Use:   "foundry <url>",
	Short: "Register a foundry index URL",
	Args:  cobra.ExactArgs(1),
	RunE:  runFoundryAdd,
}

var addIngotSubCmd = &cobra.Command{
	Use:   "ingot <reference>",
	Short: "Download and register an ingot",
	Args:  cobra.ExactArgs(1),
	RunE:  runIngotAdd,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Download resources without installing",
	Long: `Download molds, ingots, and other resources to the local cache.

Available subcommands:
  mold       Download a mold without installing
  ingot      Download an ingot without installing`,
}

var getMoldSubCmd = &cobra.Command{
	Use:   "mold <reference>",
	Short: "Download a mold without installing",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetMold,
}

var getIngotSubCmd = &cobra.Command{
	Use:   "ingot <reference>",
	Short: "Download an ingot without installing",
	Args:  cobra.ExactArgs(1),
	RunE:  runIngotGet,
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create new resources",
	Long: `Create new molds and other resources.

Available subcommands:
  mold       Scaffold a new mold directory`,
}

var newMoldSubCmd = &cobra.Command{
	Use:     "mold <name>",
	Aliases: []string{"create"},
	Short:   "Scaffold a new mold directory",
	Args:    cobra.ExactArgs(1),
	RunE:    runNewMold,
}

func init() {
	// Flags for bidirectional "new mold" must mirror "mold new" flags.
	newMoldSubCmd.Flags().StringVarP(&newMoldOutput, "output", "o", ".", "parent directory to create the mold in")
	newMoldSubCmd.Flags().BoolVar(&newMoldNoAgents, "no-agents", false, "skip generating AGENTS.md")

	// search <noun>
	rootCmd.AddCommand(searchCmd)
	searchCmd.AddCommand(searchFoundrySubCmd)

	// add <noun>
	rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(addFoundrySubCmd)
	addCmd.AddCommand(addIngotSubCmd)

	// get <noun>
	rootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getMoldSubCmd)
	getCmd.AddCommand(getIngotSubCmd)

	// new <noun>
	rootCmd.AddCommand(newCmd)
	newCmd.AddCommand(newMoldSubCmd)
}
