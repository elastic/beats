// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filelock"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newInstallCommandWithArgs(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Elastic Agent permanently on this system",
		Long: `This will install Elastic Agent permanently on this system and will become managed by the systems service manager.

Unless all the require command-line parameters are provided or -f is used this command will ask questions on how you
would like the Agent to operate.
`,
		Run: func(c *cobra.Command, args []string) {
			if err := installCmd(streams, c, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n%s\n", err, troubleshootMessage())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Force overwrite the current and do not prompt for confirmation")
	addEnrollFlags(cmd)

	return cmd
}

func installCmd(streams *cli.IOStreams, cmd *cobra.Command, args []string) error {
	err := validateEnrollFlags(cmd)
	if err != nil {
		return err
	}

	isAdmin, err := install.HasRoot()
	if err != nil {
		return fmt.Errorf("unable to perform install command while checking for administrator rights, %v", err)
	}
	if !isAdmin {
		return fmt.Errorf("unable to perform install command, not executed with %s permissions", install.PermissionUser)
	}
	status, reason := install.Status()
	force, _ := cmd.Flags().GetBool("force")
	if status == install.Installed && !force {
		return fmt.Errorf("already installed at: %s", paths.InstallPath)
	}

	// check the lock to ensure that elastic-agent is not already running in this directory
	locker := filelock.NewAppLocker(paths.Data(), paths.AgentLockFileName)
	if err := locker.TryLock(); err != nil {
		if err == filelock.ErrAppAlreadyRunning {
			return fmt.Errorf("cannot perform installation as Elastic Agent is already running from this directory")
		}
		return err
	}
	locker.Unlock()

	if status == install.Broken {
		if !force {
			fmt.Fprintf(streams.Out, "Elastic Agent is installed but currently broken: %s\n", reason)
			confirm, err := cli.Confirm(fmt.Sprintf("Continuing will re-install Elastic Agent over the current installation at %s. Do you want to continue?", paths.InstallPath), true)
			if err != nil {
				return fmt.Errorf("problem reading prompt response")
			}
			if !confirm {
				return fmt.Errorf("installation was cancelled by the user")
			}
		}
	} else {
		if !force {
			confirm, err := cli.Confirm(fmt.Sprintf("Elastic Agent will be installed at %s and will run as a service. Do you want to continue?", paths.InstallPath), true)
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
	url, _ := cmd.Flags().GetString("url")
	token, _ := cmd.Flags().GetString("enrollment-token")
	delayEnroll, _ := cmd.Flags().GetBool("delay-enroll")
	if url != "" && token != "" {
		askEnroll = false
	}
	fleetServer, _ := cmd.Flags().GetString("fleet-server-es")
	if fleetServer != "" || force || delayEnroll {
		askEnroll = false
	}
	if askEnroll {
		confirm, err := cli.Confirm("Do you want to enroll this Agent into Fleet?", true)
		if err != nil {
			return fmt.Errorf("problem reading prompt response")
		}
		if !confirm {
			// not enrolling
			enroll = false
		}
	}
	if !askEnroll && (url == "" || token == "") && fleetServer == "" {
		// force was performed without required enrollment arguments, all done (standalone mode)
		enroll = false
	}

	if enroll && fleetServer == "" {
		if url == "" {
			url, err = cli.ReadInput("URL you want to enroll this Agent into:")
			if err != nil {
				return fmt.Errorf("problem reading prompt response")
			}
			if url == "" {
				fmt.Fprintf(streams.Out, "Enrollment cancelled because no URL was provided.\n")
				return nil
			}
		}
		if token == "" {
			token, err = cli.ReadInput("Fleet enrollment token:")
			if err != nil {
				return fmt.Errorf("problem reading prompt response")
			}
			if token == "" {
				fmt.Fprintf(streams.Out, "Enrollment cancelled because no enrollment token was provided.\n")
				return nil
			}
		}
	}
	cfgFile := paths.ConfigFile()
	err = install.Install(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			install.Uninstall(cfgFile)
		}
	}()

	if !delayEnroll {
		err = install.StartService()
		if err != nil {
			fmt.Fprintf(streams.Out, "Installation failed to start Elastic Agent service.\n")
			return err
		}

		defer func() {
			if err != nil {
				install.StopService()
			}
		}()
	}

	if enroll {
		enrollArgs := []string{"enroll", "--from-install"}
		enrollArgs = append(enrollArgs, buildEnrollmentFlags(cmd, url, token)...)
		enrollCmd := exec.Command(install.ExecutablePath(), enrollArgs...)
		enrollCmd.Stdin = os.Stdin
		enrollCmd.Stdout = os.Stdout
		enrollCmd.Stderr = os.Stderr
		err = enrollCmd.Start()
		if err != nil {
			return fmt.Errorf("failed to execute enroll command: %s", err)
		}
		err = enrollCmd.Wait()
		if err != nil {
			install.Uninstall(cfgFile)
			exitErr, ok := err.(*exec.ExitError)
			if ok {
				return fmt.Errorf("enroll command failed with exit code: %d", exitErr.ExitCode())
			}
			return fmt.Errorf("enroll command failed for unknown reason: %s", err)
		}
	}

	fmt.Fprint(streams.Out, "Elastic Agent has been successfully installed.\n")
	return nil
}
