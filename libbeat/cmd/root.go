package cmd

import (
	"flag"

	"github.com/spf13/cobra"
)

// GenRootCmd returns the root command to use for your beat. It takes
// beat name as paramter, and also run command, which will be called if no args are
// given (for backwards compatibility)
func GenRootCmd(name string, runCmd *cobra.Command) *cobra.Command {
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

	// Run subcommand flags, only available to *beat run
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("N"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("e"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("v"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("httpprof"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("cpuprofile"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("memprofile"))

	// TODO deprecate in favor of subcommands (7.0):
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("configtest"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("setup"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("version"))

	// Inherit root flags from run command
	// TODO deprecate when root command no longer executes run (7.0)
	rootCmd.Flags().AddFlagSet(runCmd.Flags())

	// Register subcommands common to all beats
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(genVersionCmd(name))

	return rootCmd
}
