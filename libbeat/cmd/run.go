package cmd

import (
	"flag"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/instance"
)

func genRunCmd(name, idxPrefix, version string, beatCreator beat.Creator, runFlags *pflag.FlagSet) *cobra.Command {
	runCmd := cobra.Command{
		Use:   "run",
		Short: "Run " + name,
		Run: func(cmd *cobra.Command, args []string) {
			err := instance.Run(name, idxPrefix, version, beatCreator)
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
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("setup"))

	// TODO deprecate in favor of subcommands (7.0):
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("configtest"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("version"))

	runCmd.Flags().MarkDeprecated("version", "version flag has been deprecated, use version subcommand")
	runCmd.Flags().MarkDeprecated("configtest", "configtest flag has been deprecated, use test config subcommand")

	if runFlags != nil {
		runCmd.Flags().AddFlagSet(runFlags)
	}

	return &runCmd
}
