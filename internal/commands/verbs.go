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
	Long: `Add foundries, ingots, ores, and other resources.

Available subcommands:
  foundry    Register a foundry index
  ingot      Download and register an ingot
  ore        Download and register an ore`,
}

var addFoundrySubCmd = &cobra.Command{
	Use:   "foundry <url>",
	Short: "Register a foundry index",
	Args:  cobra.ExactArgs(1),
	RunE:  runFoundryAdd,
}

var addIngotSubCmd = &cobra.Command{
	Use:   "ingot <reference>",
	Short: "Download and register an ingot",
	Args:  cobra.ExactArgs(1),
	RunE:  runIngotAdd,
}

var addOreSubCmd = &cobra.Command{
	Use:   "ore <reference>",
	Short: "Download and register an ore",
	Args:  cobra.ExactArgs(1),
	RunE:  runOreAdd,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Download resources without installing",
	Long: `Download molds, ingots, ores, and other resources to the local cache.

Available subcommands:
  mold       Download a mold without installing
  ingot      Download an ingot without installing
  ore        Download an ore without installing`,
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

var getOreSubCmd = &cobra.Command{
	Use:   "ore <reference>",
	Short: "Download an ore without installing",
	Args:  cobra.ExactArgs(1),
	RunE:  runOreGet,
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create new resources",
	Long: `Create new molds, foundries, ores, and other resources.

Available subcommands:
  mold       Scaffold a new mold directory
  foundry    Scaffold a new foundry index
  ore        Scaffold a new ore directory`,
}

var newMoldSubCmd = &cobra.Command{
	Use:     "mold <name>",
	Aliases: []string{"create"},
	Short:   "Scaffold a new mold directory",
	Args:    cobra.ExactArgs(1),
	RunE:    runNewMold,
}

var newFoundrySubCmd = &cobra.Command{
	Use:     "foundry <name>",
	Aliases: []string{"create"},
	Short:   "Scaffold a new foundry index",
	Args:    cobra.ExactArgs(1),
	RunE:    runNewFoundry,
}

var newOreSubCmd = &cobra.Command{
	Use:   "ore <name>",
	Short: "Scaffold a new ore directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runOreNew,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
	Long: `List registered resources.

Available subcommands:
  foundry    List registered foundry indexes`,
}

var listFoundrySubCmd = &cobra.Command{
	Use:   "foundry",
	Short: "List registered foundry indexes",
	RunE:  runFoundryList,
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove resources",
	Long: `Remove registered resources.

Available subcommands:
  foundry    Remove a registered foundry index
  ingot      Remove an installed ingot
  ore        Remove an installed ore`,
}

var removeFoundrySubCmd = &cobra.Command{
	Use:   "foundry <name|url>",
	Short: "Remove a registered foundry index",
	Args:  cobra.ExactArgs(1),
	RunE:  runFoundryRemove,
}

var removeIngotSubCmd = &cobra.Command{
	Use:   "ingot <name>",
	Short: "Remove an installed ingot",
	Args:  cobra.ExactArgs(1),
	RunE:  runIngotRemove,
}

var removeOreSubCmd = &cobra.Command{
	Use:   "ore <name>",
	Short: "Remove an installed ore",
	Args:  cobra.ExactArgs(1),
	RunE:  runOreRemove,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update resources",
	Long: `Update registered resources.

Available subcommands:
  foundry    Refresh cached foundry indexes`,
}

var updateFoundrySubCmd = &cobra.Command{
	Use:   "foundry",
	Short: "Refresh cached foundry indexes",
	RunE:  runFoundryUpdate,
}

func init() {
	// Flags for bidirectional "new mold" must mirror "mold new" flags.
	newMoldSubCmd.Flags().StringVarP(&newMoldOutput, "output", "o", ".", "parent directory to create the mold in")
	newMoldSubCmd.Flags().BoolVar(&newMoldNoAgents, "no-agents", false, "skip generating AGENTS.md")

	// Flags for bidirectional "new foundry" must mirror "foundry new" flags.
	newFoundrySubCmd.Flags().StringVarP(&newFoundryOutput, "output", "o", ".", "parent directory to create the foundry in")

	// Flags for bidirectional "add ore" must mirror "ore add" flags.
	addOreSubCmd.Flags().StringVar(&oreAddAlias, "as", "", "namespace alias (install at ore.<alias>.* instead of ore.<name>.*)")
	addOreSubCmd.Flags().BoolVar(&oreAddGlobal, "global", false, "install under ~/.ailloy/ores/ instead of ./.ailloy/ores/")

	// Flags for bidirectional "remove ore" must mirror "ore remove" flags.
	removeOreSubCmd.Flags().BoolVar(&oreRemoveForce, "force", false, "remove even if other molds depend on this ore")
	removeOreSubCmd.Flags().BoolVar(&oreRemoveGlobal, "global", false, "remove from ~/.ailloy/ores/ instead of ./.ailloy/ores/")

	// Flags for bidirectional "remove ingot" must mirror "ingot remove" flags.
	removeIngotSubCmd.Flags().BoolVar(&ingotRemoveForce, "force", false, "remove even if other molds depend on this ingot")
	removeIngotSubCmd.Flags().BoolVar(&ingotRemoveGlobal, "global", false, "remove from ~/.ailloy/ingots/ instead of ./.ailloy/ingots/")

	// search <noun>
	rootCmd.AddCommand(searchCmd)
	searchCmd.AddCommand(searchFoundrySubCmd)

	// add <noun>
	rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(addFoundrySubCmd)
	addCmd.AddCommand(addIngotSubCmd)
	addCmd.AddCommand(addOreSubCmd)

	// get <noun>
	rootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getMoldSubCmd)
	getCmd.AddCommand(getIngotSubCmd)
	getCmd.AddCommand(getOreSubCmd)

	// new <noun>
	rootCmd.AddCommand(newCmd)
	newCmd.AddCommand(newMoldSubCmd)
	newCmd.AddCommand(newFoundrySubCmd)
	newCmd.AddCommand(newOreSubCmd)

	// list <noun>
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listFoundrySubCmd)

	// remove <noun>
	rootCmd.AddCommand(removeCmd)
	removeCmd.AddCommand(removeFoundrySubCmd)
	removeCmd.AddCommand(removeIngotSubCmd)
	removeCmd.AddCommand(removeOreSubCmd)

	// update <noun>
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateFoundrySubCmd)
}
