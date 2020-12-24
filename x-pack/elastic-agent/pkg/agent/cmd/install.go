// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"

	"github.com/spf13/cobra"

	c "github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/warn"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newInstallCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Elastic Agent permanently on this system",
		Long: `This will install Elastic Agent permanently on this system and will become managed by the systems service manager.

Unless all the require command-line parameters are provided or -f is used this command will ask questions on how you
would like the Agent to operate.
`,
		Run: func(c *cobra.Command, args []string) {
			if err := installCmd(streams, c, flags, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringP("kibana-url", "k", "", "URL of Kibana to enroll Agent into Fleet")
	cmd.Flags().StringP("enrollment-token", "t", "", "Enrollment token to use to enroll Agent into Fleet")
	cmd.Flags().BoolP("force", "f", false, "Force overwrite the current and do not prompt for confirmation")
	addEnrollFlags(cmd)

	return cmd
}

func installCmd(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	var err error
	if !install.HasRoot() {
		return fmt.Errorf("unable to perform install command, not executed with %s permissions", install.PermissionUser)
	}
	status, reason := install.Status()
	if status == install.Installed {
		return fmt.Errorf("already installed at: %s", install.InstallPath)
	}

	// check the lock to ensure that elastic-agent is not already running in this directory
	locker := application.NewAppLocker(paths.Data(), agentLockFileName)
	if err := locker.TryLock(); err != nil {
		if err == application.ErrAppAlreadyRunning {
			return fmt.Errorf("cannot perform installation as Elastic Agent is already running from this directory")
		}
		return err
	}
	locker.Unlock()

	warn.PrintNotGA(streams.Out)
	force, _ := cmd.Flags().GetBool("force")
	if status == install.Broken {
		if !force {
			fmt.Fprintf(streams.Out, "Elastic Agent is installed but currently broken: %s\n", reason)
			confirm, err := c.Confirm(fmt.Sprintf("Continuing will re-install Elastic Agent over the current installation at %s. Do you want to continue?", install.InstallPath), true)
			if err != nil {
				return fmt.Errorf("problem reading prompt response")
			}
			if !confirm {
				return fmt.Errorf("installation was cancelled by the user")
			}
		}
	} else {
		if !force {
			confirm, err := c.Confirm(fmt.Sprintf("Elastic Agent will be installed at %s and will run as a service. Do you want to continue?", install.InstallPath), true)
			if err != nil {
				return fmt.Errorf("problem reading prompt response")
			}
			if !confirm {
				return fmt.Errorf("installation was cancelled by the user")
			}
		}
	}

	enroll := true
	askEnroll := true
	kibana, _ := cmd.Flags().GetString("kibana-url")
	token, _ := cmd.Flags().GetString("enrollment-token")
	if kibana != "" && token != "" {
		askEnroll = false
	}
	if force {
		askEnroll = false
	}
	if askEnroll {
		confirm, err := c.Confirm("Do you want to enroll this Agent into Fleet?", true)
		if err != nil {
			return fmt.Errorf("problem reading prompt response")
		}
		if !confirm {
			// not enrolling
			enroll = false
		}
	}
	if !askEnroll && (kibana == "" || token == "") {
		// force was performed without required enrollment arguments, all done (standalone mode)
		enroll = false
	}

	if enroll {
		if kibana == "" {
			kibana, err = c.ReadInput("Kibana URL you want to enroll this Agent into:")
			if err != nil {
				return fmt.Errorf("problem reading prompt response")
			}
			if kibana == "" {
				fmt.Fprintf(streams.Out, "Enrollment cancelled because no URL was provided.\n")
				return nil
			}
		}
		if token == "" {
			token, err = c.ReadInput("Fleet enrollment token:")
			if err != nil {
				return fmt.Errorf("problem reading prompt response")
			}
			if token == "" {
				fmt.Fprintf(streams.Out, "Enrollment cancelled because no enrollment token was provided.\n")
				return nil
			}
		}
	}

	err = install.Install()
	if err != nil {
		return err
	}

	if enroll {
		enrollArgs := []string{"enroll", kibana, token, "--from-install"}
		enrollArgs = append(enrollArgs, buildEnrollmentFlags(cmd)...)
		enrollCmd := exec.Command(install.ExecutablePath(), enrollArgs...)
		enrollCmd.Stdin = os.Stdin
		enrollCmd.Stdout = os.Stdout
		enrollCmd.Stderr = os.Stderr
		err = enrollCmd.Start()
		if err != nil {
			install.Uninstall()
			return fmt.Errorf("failed to execute enroll command: %s", err)
		}
		err = enrollCmd.Wait()
		if err != nil {
			install.Uninstall()
			exitErr, ok := err.(*exec.ExitError)
			if ok {
				return fmt.Errorf("enroll command failed with exit code: %d", exitErr.ExitCode())
			}
			return fmt.Errorf("enroll command failed for unknown reason: %s", err)
		}
	}

	err = install.StartService()
	if err != nil {
		fmt.Fprintf(streams.Out, "Installation of required system files was successful, but starting of the service failed.\n")
		return err
	}
	fmt.Fprintf(streams.Out, "Installation was successful and Elastic Agent is running.\n")
	return nil
}
