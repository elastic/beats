package cmd

import (
	"flag"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	// backwards compatibility workaround, convert -flags to --flags:
	for i, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 2 {
			logp.Deprecate("6.0", "Argument %s should be -%s", arg, arg)
			os.Args[1+i] = "-" + arg
		}
	}
}

// GenRootCmd returns the root command to use for your beat. It takes
// beat name as paramter, and also run command, which will be called if no args are
// given (for backwards compatibility)
func GenRootCmd(name, version string, beatCreator beat.Creator) *cobra.Command {
	return GenRootCmdWithRunFlags(name, version, beatCreator, nil)
}

// GenRootCmdWithRunFlags returns the root command to use for your beat. It takes
// beat name as paramter, and also run command, which will be called if no args are
// given (for backwards compatibility). runFlags parameter must the flagset used by
// run command
func GenRootCmdWithRunFlags(name, version string, beatCreator beat.Creator, runFlags *pflag.FlagSet) *cobra.Command {
	runCmd := genRunCmd(name, version, beatCreator, runFlags)

	rootCmd := &cobra.Command{
		Use: name,
		Run: runCmd.Run,
	}

	// Persistent flags, common accross all subcommands
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("E"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("c"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("d"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("v"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("e"))
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
	rootCmd.AddCommand(genVersionCmd(name, version))
	rootCmd.AddCommand(genSetupCmd(name))

	return rootCmd
}
