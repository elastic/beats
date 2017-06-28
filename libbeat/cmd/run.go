package cmd

import (
	"flag"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/elastic/beats/libbeat/beat"
)

func genRunCmd(name, version string, beatCreator beat.Creator, runFlags *pflag.FlagSet) *cobra.Command {
	runCmd := cobra.Command{
		Use:   "run",
		Short: "Run " + name,
		Run: func(cmd *cobra.Command, args []string) {
			err := beat.Run(name, version, beatCreator)
			if err != nil {
				os.Exit(1)
			}
		},
	}

	// Run subcommand flags, only available to *beat run
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("N"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("httpprof"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("cpuprofile"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("memprofile"))

	// TODO deprecate in favor of subcommands (7.0):
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("configtest"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("setup"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("version"))

	runCmd.Flags().MarkDeprecated("version", "version flag has been deprectad, use version subcommand")
	runCmd.Flags().MarkDeprecated("configtest", "setup flag has been deprectad, use configtest subcommand")

	if runFlags != nil {
		runCmd.Flags().AddFlagSet(runFlags)
	}

	return &runCmd
}
