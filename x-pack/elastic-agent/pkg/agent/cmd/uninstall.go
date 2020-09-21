// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	c "github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newUninstallCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall permanent Elastic Agent from this system",
		Long: `This will uninstall permanent Elastic Agent from this system and will no longer be managed by this system.

Unless -f is used this command will ask confirmation before performing removal.
`,
		Run: func(c *cobra.Command, args []string) {
			if err := uninstallCmd(streams, c, flags, args); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Force overwrite the current and do not prompt for confirmation")

	return cmd
}

func uninstallCmd(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	if !install.HasRoot() {
		return fmt.Errorf("Error: unable to perform uninstall command, not executed with %s permissions.", install.PermissionUser)
	}
	status, reason := install.Status()
	if status == install.NotInstalled {
		return fmt.Errorf("Elastic Agent is not installed")
	}
	if status == install.Installed && !install.RunningInstalled() {
		return fmt.Errorf("Elastic Agent can only be uninstall by executing the installed Elastic Agent at: %s", install.ExecutablePath())
	}

	force, _ := cmd.Flags().GetBool("force")
	if status == install.Broken {
		if !force {
			fmt.Fprintf(streams.Out, "Elastic Agent is installed but currently broken: %s\n", reason)
			confirm, err := c.Confirm("Continuing will uninstall the broken Elastic Agent. Do you want to continue?", true)
			if err != nil {
				return fmt.Errorf("Error: problem reading prompt response")
			}
			if !confirm {
				return fmt.Errorf("Uninstall was cancelled by the user")
			}
		}
	} else {
		if !force {
			confirm, err := c.Confirm("Elastic Agent will be uninstalled from your system. Do you want to continue?", true)
			if err != nil {
				return fmt.Errorf("Error: problem reading prompt response")
			}
			if !confirm {
				return fmt.Errorf("Uninstall was cancelled by the user")
			}
		}
	}

	err := install.Uninstall()
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}
	fmt.Fprintf(streams.Out, "Elastic Agent has been uninstalled.\n")

	if runtime.GOOS == "windows" {
		// The installation path will still exists because we are executing from that
		// directory. So cmd.exe is spawned to remove the directory after we exit.
		cmd := exec.Command(filepath.Join(os.Getenv("windir"), "system32", "cmd.exe"), "/C", "rmdir", "/s", "/q", install.InstallPath)
		_ = cmd.Start()
	}

	return nil
}
