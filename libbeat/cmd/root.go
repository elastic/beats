package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
)

func init() {
	// backwards compatibility workaround, convert -flags to --flags:
	for i, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 2 {
			os.Args[1+i] = "-" + arg
		}
	}
}

// BeatsRootCmd handles all application command line interface, parses user
// flags and runs subcommands
type BeatsRootCmd struct {
	cobra.Command
	RunCmd        *cobra.Command
	SetupCmd      *cobra.Command
	VersionCmd    *cobra.Command
	CompletionCmd *cobra.Command
	ExportCmd     *cobra.Command
	TestCmd       *cobra.Command
	KeystoreCmd   *cobra.Command
}

// GenRootCmd returns the root command to use for your beat. It takes
// beat name as parameter, and also run command, which will be called if no args are
// given (for backwards compatibility)
func GenRootCmd(name, version string, beatCreator beat.Creator) *BeatsRootCmd {
	return GenRootCmdWithRunFlags(name, version, beatCreator, nil)
}

// GenRootCmdWithRunFlags returns the root command to use for your beat. It takes
// beat name as parameter, and also run command, which will be called if no args are
// given (for backwards compatibility). runFlags parameter must the flagset used by
// run command
func GenRootCmdWithRunFlags(name, version string, beatCreator beat.Creator, runFlags *pflag.FlagSet) *BeatsRootCmd {
	return GenRootCmdWithIndexPrefixWithRunFlags(name, name, version, beatCreator, runFlags)
}

func GenRootCmdWithIndexPrefixWithRunFlags(name, indexPrefix, version string, beatCreator beat.Creator, runFlags *pflag.FlagSet) *BeatsRootCmd {
	rootCmd := &BeatsRootCmd{}
	rootCmd.Use = name

	// Due to a dependence upon the beat name, the default config file path
	err := cfgfile.ChangeDefaultCfgfileFlag(name)
	if err != nil {
		panic(fmt.Errorf("failed to set default config file path: %v", err))
	}

	// must be updated prior to CLI flag handling.

	rootCmd.RunCmd = genRunCmd(name, indexPrefix, version, beatCreator, runFlags)
	rootCmd.SetupCmd = genSetupCmd(name, indexPrefix, version, beatCreator)
	rootCmd.VersionCmd = genVersionCmd(name, version)
	rootCmd.CompletionCmd = genCompletionCmd(name, version, rootCmd)
	rootCmd.ExportCmd = genExportCmd(name, indexPrefix, version)
	rootCmd.TestCmd = genTestCmd(name, version, beatCreator)
	rootCmd.KeystoreCmd = genKeystoreCmd(name, indexPrefix, version, beatCreator, runFlags)

	// Root command is an alias for run
	rootCmd.Run = rootCmd.RunCmd.Run

	// Persistent flags, common across all subcommands
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
	if f := flag.CommandLine.Lookup("plugin"); f != nil {
		rootCmd.PersistentFlags().AddGoFlag(f)
	}

	// Inherit root flags from run command
	// TODO deprecate when root command no longer executes run (7.0)
	rootCmd.Flags().AddFlagSet(rootCmd.RunCmd.Flags())

	// Register subcommands common to all beats
	rootCmd.AddCommand(rootCmd.RunCmd)
	rootCmd.AddCommand(rootCmd.SetupCmd)
	rootCmd.AddCommand(rootCmd.VersionCmd)
	rootCmd.AddCommand(rootCmd.CompletionCmd)
	rootCmd.AddCommand(rootCmd.ExportCmd)
	rootCmd.AddCommand(rootCmd.TestCmd)
	rootCmd.AddCommand(rootCmd.KeystoreCmd)

	return rootCmd
}
