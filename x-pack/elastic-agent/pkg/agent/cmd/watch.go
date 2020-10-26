// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/rollback"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

const (
	// period during which we monitor for failures resulting in a rollback
	gracePeriod = 10 * time.Minute
)

func newWatchCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch watches Elastic Agent for failures and initiates rollback.",
		Long:  `Watch watches Elastic Agent for failures and initiates rollback.`,
		Run: func(c *cobra.Command, args []string) {
			if err := watchCmd(streams, c, flags, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}

func watchCmd(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	marker, err := upgrade.LoadMarker()
	if err != nil {
		return err
	}
	if marker == nil {
		// no marker found we're not in upgrade process
		return nil
	}

	isWithinGrace, tilGrace := gracePeriod(marker)
	if !isWithinGrace {
		// if it is started outside of upgrade loop exit nicely
		return nil
	}

	locker := rollback.NewLocker(paths.Top())
	if err := locker.TryLock(); err != nil {
		if err == rollback.ErrAlreadyLocked {
			return nil
		}

		return err
	}
	defer locker.Unlock()

	ctx := context.Background()

	if err := watch(ctx, tilGrace); err != nil {
		return rollback.Rollback(ctx, marker.PrevHash, marker.Hash)
	}

	return rollback.Cleanup(marker.PrevHash)
}

func watch(ctx context.Context, tilGrace time.Duration) error {
	errChan := make(chan error)
	crashChan := make(chan error)

	ctx, cancel := context.WithCancel(ctx)

	//cleanup
	defer func() {
		cancel()
		close(errChan)
		close(crashChan)
	}()

	errorChecker := rollback.NewErrorChecker(errChan)
	crashChecker := rollback.NewCrashChecker(errChan)
	go errorChecker.Run(ctx)
	go crashChecker.Run(ctx)

WATCHLOOP:
	for {
		select {
		case <-ctx.Done():
			break WATCHLOOP
		// grace period passed, agent is considered stable
		case <-time.After(tilGrace):
			break WATCHLOOP
		// Agent in degraded state.
		case err := <-errChan:
			return err
		// Agent keeps crashing unexpectedly
		case err := <-crashChan:
			return err
		}
	}

	return nil
}

// gracePeriod returns true if it is within grace period and time until grace period ends.
// otherwise it returns false and 0
func gracePeriod(marker *upgrade.UpdateMarker) (bool, time.Duration) {
	// TODO: finish
	return true, gracePeriod
}
