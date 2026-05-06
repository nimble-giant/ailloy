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
  foundry    Register a foundry index
  ingot      Download and register an ingot`,
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
	Long: `Create new molds, foundries, and other resources.

Available subcommands:
  mold       Scaffold a new mold directory
  foundry    Scaffold a new foundry index`,
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

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
	Long: `List registered resources.

Available subcommands:
  foundry      List registered foundry indexes
  extensions   List available and installed CLI extensions`,
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
  foundry      Remove a registered foundry index
  extension    Remove an installed extension`,
}

var removeFoundrySubCmd = &cobra.Command{
	Use:   "foundry <name|url>",
	Short: "Remove a registered foundry index",
	Args:  cobra.ExactArgs(1),
	RunE:  runFoundryRemove,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update resources",
	Long: `Update registered resources.

Available subcommands:
  foundry      Refresh cached foundry indexes
  extension    Update an installed extension`,
}

var updateFoundrySubCmd = &cobra.Command{
	Use:   "foundry",
	Short: "Refresh cached foundry indexes",
	RunE:  runFoundryUpdate,
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install resources",
	Long: `Install resources.

Available subcommands:
  extension    Install an ailloy extension`,
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear resources",
	Long: `Clear resources.

Available subcommands:
  cache      Clear ailloy's on-disk cache`,
}

var clearCacheSubCmd = &cobra.Command{
	Use:   "cache",
	Short: "Clear ailloy's on-disk cache",
	Args:  cobra.NoArgs,
	RunE:  runCacheClear,
}

func init() {
	// Flags for bidirectional "new mold" must mirror "mold new" flags.
	newMoldSubCmd.Flags().StringVarP(&newMoldOutput, "output", "o", ".", "parent directory to create the mold in")
	newMoldSubCmd.Flags().BoolVar(&newMoldNoAgents, "no-agents", false, "skip generating AGENTS.md")

	// Flags for bidirectional "new foundry" must mirror "foundry new" flags.
	newFoundrySubCmd.Flags().StringVarP(&newFoundryOutput, "output", "o", ".", "parent directory to create the foundry in")

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
	newCmd.AddCommand(newFoundrySubCmd)

	// list <noun>
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listFoundrySubCmd)
	listCmd.AddCommand(listExtensionsSubCmd)

	// remove <noun>
	rootCmd.AddCommand(removeCmd)
	removeCmd.AddCommand(removeFoundrySubCmd)
	removeCmd.AddCommand(removeExtensionSubCmd)

	// update <noun>
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateFoundrySubCmd)
	updateCmd.AddCommand(updateExtensionSubCmd)

	// install <noun>  (added with extensions)
	rootCmd.AddCommand(installCmd)
	installCmd.AddCommand(installExtensionSubCmd)

	// clear <noun>  (mirrors "cache clear")
	rootCmd.AddCommand(clearCmd)
	clearCmd.AddCommand(clearCacheSubCmd)

	// Mirror Long help and flags so "clear cache" matches "cache clear".
	// Set in init() — package-level var initializer order across files is unspecified.
	clearCacheSubCmd.Long = cacheClearCmd.Long
	registerCacheClearFlags(clearCacheSubCmd)

	// show <noun>  (showCmd is declared in mold.go; we just hang another sub on it)
	showCmd.AddCommand(showExtensionSubCmd)
}
