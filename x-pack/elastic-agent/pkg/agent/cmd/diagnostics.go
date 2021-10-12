// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

var diagOutputs = map[string]outputter{
	"human": humanDiagnosticsOutput,
	"json":  jsonOutput,
	"yaml":  yamlOutput,
}

func newDiagnosticsCommand(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Diagnostics gather diagnostics information from the elastic-agent and running processes.",
		Long:  "Diagnostics gather diagnostics information from the elastic-agent and running processes.",
		Run: func(c *cobra.Command, args []string) {
			if err := diagnosticCmd(streams, c, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n%s\n", err, troubleshootMessage())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().String("output", "human", "Output the diagnostics information in either human, json, or yaml (default: human)")

	return cmd
}

// DiagnosticsInfo a struct to track all inforation related to diagnostics for the agent.
type DiagnosticsInfo struct {
	ProcMeta     []client.ProcMeta
	AgentVersion client.Version
}

func diagnosticCmd(streams *cli.IOStreams, cmd *cobra.Command, args []string) error {
	err := tryContainerLoadPaths()
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	outputFunc, ok := diagOutputs[output]
	if !ok {
		return fmt.Errorf("unsupported output: %s", output)
	}

	ctx := handleSignal(context.Background())
	innerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	diag, err := getDiagnostics(innerCtx)
	if err == context.DeadlineExceeded {
		return errors.New("timed out after 30 seconds trying to connect to Elastic Agent daemon")
	} else if err == context.Canceled {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to communicate with Elastic Agent daemon: %s", err)
	}

	return outputFunc(streams.Out, diag)
}

func getDiagnostics(ctx context.Context) (DiagnosticsInfo, error) {
	daemon := client.New()
	diag := DiagnosticsInfo{}
	err := daemon.Connect(ctx)
	if err != nil {
		return DiagnosticsInfo{}, err
	}
	defer daemon.Disconnect()

	bv, err := daemon.ProcMeta(ctx)
	if err != nil {
		return DiagnosticsInfo{}, err
	}
	diag.ProcMeta = bv

	version, err := daemon.Version(ctx)
	if err != nil {
		return DiagnosticsInfo{}, err
	}
	diag.AgentVersion = version

	return diag, nil
}

func humanDiagnosticsOutput(w io.Writer, obj interface{}) error {
	diag, ok := obj.(DiagnosticsInfo)
	if !ok {
		return fmt.Errorf("unable to cast %T as DiagnosticsInfo")
	}
	return outputDiagnostics(w, diag)
}

func outputDiagnostics(w io.Writer, d DiagnosticsInfo) error {
	fmt.Fprintf(w, "%#v\n", d)
	return nil
}
