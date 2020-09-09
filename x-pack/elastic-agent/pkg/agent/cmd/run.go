// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/service"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/reexec"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func newRunCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Start the elastic-agent.",
		Run: func(_ *cobra.Command, _ []string) {
			if err := run(flags, streams); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}
}

func run(flags *globalFlags, streams *cli.IOStreams) error { // Windows: Mark service as stopped.
	// After this is run, the service is considered by the OS to be stopped.
	// This must be the first deferred cleanup task (last to execute).
	defer service.NotifyTermination()

	locker := application.NewAppLocker(paths.Data())
	if err := locker.TryLock(); err != nil {
		return err
	}
	defer locker.Unlock()

	service.BeforeRun()
	defer service.Cleanup()

	// register as a service
	stop := make(chan bool)
	_, cancel := context.WithCancel(context.Background())
	var stopBeat = func() {
		close(stop)
	}
	service.HandleSignals(stopBeat, cancel)

	pathConfigFile := flags.Config()
	rawConfig, err := application.LoadConfigFromFile(pathConfigFile)
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

	logger, err := logger.NewFromConfig("", cfg.Settings.LoggingConfig)
	if err != nil {
		return err
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	rexLogger := logger.Named("reexec")
	rex := reexec.NewManager(rexLogger, execPath)

	// start the control listener
	control := server.New(logger.Named("control"), rex)
	if err := control.Start(); err != nil {
		return err
	}
	defer control.Stop()

	app, err := application.New(logger, pathConfigFile)
	if err != nil {
		return err
	}

	if err := app.Start(); err != nil {
		return err
	}

	// listen for signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	reexecing := false
	for {
		breakout := false
		select {
		case <-stop:
			breakout = true
		case <-rex.ShutdownChan():
			reexecing = true
			breakout = true
		case sig := <-signals:
			if sig == syscall.SIGHUP {
				rexLogger.Infof("SIGHUP triggered re-exec")
				rex.ReExec()
			} else {
				breakout = true
			}
		}
		if breakout {
			if !reexecing {
				logger.Info("Shutting down Elastic Agent and sending last events...")
			}
			break
		}
	}

	err = app.Stop()
	if !reexecing {
		logger.Info("Shutting down completed.")
		return err
	}
	rex.ShutdownComplete()
	return err
}
