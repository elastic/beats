// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	// import logp flags
	_ "github.com/elastic/beats/v7/libbeat/logp/configure"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/basecmd"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

func troubleshootMessage() string {
	v := strings.Split(release.Version(), ".")
	version := strings.Join(v[:2], ".")
	return fmt.Sprintf("For help, please see our troubleshooting guide at https://www.elastic.co/guide/en/fleet/%s/fleet-troubleshooting.html", version)
}

// NewCommand returns the default command for the agent.
func NewCommand() *cobra.Command {
	return NewCommandWithArgs(os.Args, cli.NewIOStreams())
}

// NewCommandWithArgs returns a new agent with the flags and the subcommand.
func NewCommandWithArgs(args []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "elastic-agent [subcommand]",
	}

	// path flags
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.home"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.home.unversioned"))
	cmd.PersistentFlags().MarkHidden("path.home.unversioned") // hidden used internally by container subcommand
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.config"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("c"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.logs"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.downloads"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.install"))

	// logging flags
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("v"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("e"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("d"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("environment"))

	// sub-commands
	run := newRunCommandWithArgs(args, streams)
	cmd.AddCommand(basecmd.NewDefaultCommandsWithArgs(args, streams)...)
	cmd.AddCommand(run)
	cmd.AddCommand(newInstallCommandWithArgs(args, streams))
	cmd.AddCommand(newUninstallCommandWithArgs(args, streams))
	cmd.AddCommand(newUpgradeCommandWithArgs(args, streams))
	cmd.AddCommand(newEnrollCommandWithArgs(args, streams))
	cmd.AddCommand(newInspectCommandWithArgs(args, streams))
	cmd.AddCommand(newWatchCommandWithArgs(args, streams))
	cmd.AddCommand(newContainerCommand(args, streams))
	cmd.AddCommand(newStatusCommand(args, streams))
	cmd.AddCommand(newDiagnosticsCommand(args, streams))

	// windows special hidden sub-command (only added on windows)
	reexec := newReExecWindowsCommand(args, streams)
	if reexec != nil {
		cmd.AddCommand(reexec)
	}
	cmd.Run = run.Run

	return cmd
}
