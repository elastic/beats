// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newReExecWindowsCommand(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Hidden: true,
		Use:    "reexec_windows <service_name> <pid>",
		Short:  "ReExec the windows service",
		Long:   "This waits for the windows service to stop then restarts it to allow self-upgrading.",
		Args:   cobra.ExactArgs(2),
		Run: func(c *cobra.Command, args []string) {
			serviceName := args[0]
			servicePid, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
			err = reExec(serviceName, servicePid)
			if err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n%s\n", err, troubleshootMessage())
				os.Exit(1)
			}
		},
	}

	return cmd
}

func reExec(serviceName string, servicePid int) error {
	manager, err := mgr.Connect()
	if err != nil {
		return errors.New(err, "failed to connect to service manager")
	}
	service, err := manager.OpenService(serviceName)
	if err != nil {
		return errors.New(err, "failed to open service")
	}
	for {
		status, err := service.Query()
		if err != nil {
			return errors.New(err, "failed to query service")
		}
		if status.State == svc.Stopped {
			err = service.Start()
			if err != nil {
				return errors.New(err, "failed to start service")
			}
			// triggered restart; done
			return nil
		}
		if int(status.ProcessId) != servicePid {
			// already restarted; has different PID, done!
			return nil
		}
		<-time.After(300 * time.Millisecond)
	}
}
