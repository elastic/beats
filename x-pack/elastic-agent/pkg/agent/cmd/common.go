// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	// import logp flags
	_ "github.com/elastic/beats/v7/libbeat/logp/configure"
	"github.com/elastic/beats/v7/libbeat/service"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/reexec"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/basecmd"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	defaultConfig = "elastic-agent.yml"
	hashLen       = 6
	commitFile    = ".build_hash.txt"
)

type globalFlags struct {
	PathConfigFile string
}

// Config returns path which identifies configuration file.
func (f *globalFlags) Config() string {
	if len(f.PathConfigFile) == 0 || f.PathConfigFile == defaultConfig {
		return filepath.Join(paths.Config(), defaultConfig)
	}
	return f.PathConfigFile
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

	flags := &globalFlags{}

	// path flags
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.home"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.config"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.data"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.logs"))
	cmd.PersistentFlags().StringVarP(&flags.PathConfigFile, "c", "c", defaultConfig, `Configuration file, relative to path.config`)

	// logging flags
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("v"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("e"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("d"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("environment"))

	// sub-commands
	run := newRunCommandWithArgs(flags, args, streams)
	cmd.AddCommand(basecmd.NewDefaultCommandsWithArgs(args, streams)...)
	cmd.AddCommand(run)
	cmd.AddCommand(newEnrollCommandWithArgs(flags, args, streams))
	cmd.AddCommand(newIntrospectCommandWithArgs(flags, args, streams))

	// windows special hidden sub-command (only added on windows)
	reexec := newReExecWindowsCommand(flags, args, streams)
	if reexec != nil {
		cmd.AddCommand(reexec)
	}
	cmd.PersistentPreRunE = preRunCheck(flags)
	cmd.Run = run.Run

	return cmd
}

func preRunCheck(flags *globalFlags) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		commitFilepath := filepath.Join(paths.Home(), commitFile)
		content, err := ioutil.ReadFile(commitFilepath)

		// file is not present we are at child
		if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return err
		}

		// Windows: Mark service as stopped.
		// After this is run, the service is considered by the OS to be stopped.
		// This must be the first deferred cleanup task (last to execute).
		defer service.NotifyTermination()

		service.BeforeRun()

		stop := make(chan struct{})
		stopFn := func() {
			close(stop)
			os.Exit(0)
		}
		_, cancel := context.WithCancel(context.Background())
		service.HandleSignals(stopFn, cancel)
		defer service.Cleanup()

		pathConfigFile := flags.Config()
		rawConfig, err := config.LoadYAML(pathConfigFile)
		if err != nil {
			return errors.New(err,
				fmt.Sprintf("could not read configuration file %s", pathConfigFile),
				errors.TypeFilesystem,
				errors.M(errors.MetaKeyPath, pathConfigFile))
		}

		cfg, err := configuration.NewFromConfig(rawConfig)
		if err != nil {
			return errors.New(err,
				fmt.Sprintf("could not parse configuration file %s", pathConfigFile),
				errors.TypeFilesystem,
				errors.M(errors.MetaKeyPath, pathConfigFile))
		}

		if err := startSubprocess(content, cfg.Settings.LoggingConfig, stop); err != nil {
			return err
		}

		// prevent running Run function
		os.Exit(0)

		return nil
	}
}

func hashedDirName(filecontent []byte) string {
	s := strings.TrimSpace(string(filecontent))
	if len(s) == 0 {
		return "elastic-agent"
	}

	if len(s) > hashLen {
		s = s[:hashLen]
	}

	return fmt.Sprintf("elastic-agent-%s", s)
}

func startSubprocess(hasFileContent []byte, loggingConfig *logger.Config, stopChan <-chan struct{}) error {
	reexecPath := filepath.Join(paths.Data(), hashedDirName(hasFileContent), filepath.Base(os.Args[0]))
	argsOverrides := []string{
		"--path.data", paths.Data(),
		"--path.home", filepath.Dir(reexecPath),
		"--path.config", paths.Config(),
	}

	if runtime.GOOS == "windows" {
		args := append([]string{reexecPath}, os.Args[1:]...)
		args = append(args, argsOverrides...)
		// no support for exec just spin a new child
		cmd := exec.Cmd{
			Path:   reexecPath,
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		go func() {
			<-stopChan
			cmd.Process.Kill()
		}()

		if err := cmd.Start(); err != nil {
			return err
		}

		// Wait so agent wont exit and service wont start another instance
		return cmd.Wait()
	}

	logger, err := logger.NewFromConfig("", loggingConfig)
	if err != nil {
		return err
	}

	rexLogger := logger.Named("reexec")
	rm := reexec.Manager(rexLogger, reexecPath)
	rm.ReExec(argsOverrides...)

	// trigger reexec
	rm.ShutdownComplete()

	return nil
}
