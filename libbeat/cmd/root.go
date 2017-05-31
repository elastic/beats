package cmd

import (
	"flag"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/spf13/cobra"
)

// GenRootCmd returns the root command to use for your beat. It takes
// beat name as paramter, and also run command, which will be called if no args are
// given (for backwards compatibility)
func GenRootCmd(name string, beatCreator beat.Creator) *cobra.Command {
	runCmd := genRunCmd(name, beatCreator)

	rootCmd := &cobra.Command{
		Use: name,
		Run: runCmd.Run,
	}

	// Persistent flags, common accross all subcommands
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("E"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("c"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("d"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.config"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.data"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.logs"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.home"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("strict.perms"))

	// Inherit root flags from run command
	// TODO deprecate when root command no longer executes run (7.0)
	rootCmd.Flags().AddFlagSet(runCmd.Flags())

	// Register subcommands common to all beats
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(genVersionCmd(name))

	return rootCmd
}
