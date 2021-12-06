// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filelock"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	// period during which we monitor for failures resulting in a rollback
	gracePeriodDuration = 10 * time.Minute

	watcherName     = "elastic-agent-watcher"
	watcherLockFile = "watcher.lock"
)

func newWatchCommandWithArgs(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch watches Elastic Agent for failures and initiates rollback.",
		Long:  `Watch watches Elastic Agent for failures and initiates rollback.`,
		Run: func(c *cobra.Command, args []string) {
			if err := watchCmd(streams, c, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n%s\n", err, troubleshootMessage())
				os.Exit(1)
			}
		},
	}

	return cmd
}

func watchCmd(streams *cli.IOStreams, cmd *cobra.Command, args []string) error {
	log, err := configuredLogger()
	if err != nil {
		return err
	}

	marker, err := upgrade.LoadMarker()
	if err != nil {
		log.Error("failed to load marker", err)
		return err
	}
	if marker == nil {
		// no marker found we're not in upgrade process
		log.Debugf("update marker not present at '%s'", paths.Data())
		return nil
	}

	locker := filelock.NewAppLocker(paths.Top(), watcherLockFile)
	if err := locker.TryLock(); err != nil {
		if err == filelock.ErrAppAlreadyRunning {
			log.Debugf("exiting, lock already exists")
			return nil
		}

		log.Error("failed to acquire lock", err)
		return err
	}
	defer locker.Unlock()

	isWithinGrace, tilGrace := gracePeriod(marker)
	if !isWithinGrace {
		log.Debugf("not within grace [updatedOn %v] %v", marker.UpdatedOn.String(), time.Since(marker.UpdatedOn).String())
		// if it is started outside of upgrade loop
		// if we're not within grace and marker is still there it might mean
		// that cleanup was not performed ok, cleanup everything except current version
		// hash is the same as hash of agent which initiated watcher.
		if err := upgrade.Cleanup(release.ShortCommit(), true); err != nil {
			log.Error("rollback failed", err)
		}
		// exit nicely
		return nil
	}

	ctx := context.Background()
	if err := watch(ctx, tilGrace, log); err != nil {
		log.Debugf("Error detected proceeding to rollback: %v", err)
		err = upgrade.Rollback(ctx, marker.PrevHash, marker.Hash)
		if err != nil {
			log.Error("rollback failed", err)
		}
		return err
	}

	// cleanup older versions,
	// in windows it might leave self untouched, this will get cleaned up
	// later at the start, because for windows we leave marker untouched.
	removeMarker := runtime.GOOS != "windows"
	err = upgrade.Cleanup(marker.Hash, removeMarker)
	if err != nil {
		log.Error("rollback failed", err)
	}
	return err
}

func watch(ctx context.Context, tilGrace time.Duration, log *logger.Logger) error {
	errChan := make(chan error)
	crashChan := make(chan error)

	ctx, cancel := context.WithCancel(ctx)

	//cleanup
	defer func() {
		cancel()
		close(errChan)
		close(crashChan)
	}()

	errorChecker, err := upgrade.NewErrorChecker(errChan, log)
	if err != nil {
		return err
	}

	crashChecker, err := upgrade.NewCrashChecker(ctx, errChan, log)
	if err != nil {
		return err
	}

	go errorChecker.Run(ctx)
	go crashChecker.Run(ctx)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	t := time.NewTimer(tilGrace)
	defer t.Stop()

WATCHLOOP:
	for {
		select {
		case <-signals:
			// ignore
			continue
		case <-ctx.Done():
			break WATCHLOOP
		// grace period passed, agent is considered stable
		case <-t.C:
			log.Info("Grace period passed, not watching")
			break WATCHLOOP
		// Agent in degraded state.
		case err := <-errChan:
			log.Error("Agent Error detected", err)
			return err
		// Agent keeps crashing unexpectedly
		case err := <-crashChan:
			log.Error("Agent crash detected", err)
			return err
		}
	}

	return nil
}

// gracePeriod returns true if it is within grace period and time until grace period ends.
// otherwise it returns false and 0
func gracePeriod(marker *upgrade.UpdateMarker) (bool, time.Duration) {
	sinceUpdate := time.Since(marker.UpdatedOn)

	if 0 < sinceUpdate && sinceUpdate < gracePeriodDuration {
		return true, gracePeriodDuration - sinceUpdate
	}

	return false, gracePeriodDuration
}

func configuredLogger() (*logger.Logger, error) {
	pathConfigFile := paths.ConfigFile()
	rawConfig, err := config.LoadFile(pathConfigFile)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("could not read configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("could not parse configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	cfg.Settings.LoggingConfig.Beat = watcherName

	logger, err := logger.NewFromConfig("", cfg.Settings.LoggingConfig, false)
	if err != nil {
		return nil, err
	}

	return logger, nil
}
