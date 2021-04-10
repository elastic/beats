// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

type outputter func(io.Writer, *client.AgentStatus) error

var outputs = map[string]outputter{
	"human": humanOutput,
	"json":  jsonOutput,
	"yaml":  yamlOutput,
}

func newStatusCommand(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Status returns the current status of the running Elastic Agent daemon.",
		Long:  `Status returns the current status of the running Elastic Agent daemon.`,
		Run: func(c *cobra.Command, args []string) {
			if err := statusCmd(streams, c, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().String("output", "human", "Output the status information in either human, json, or yaml (default: human)")

	return cmd
}

func statusCmd(streams *cli.IOStreams, cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	outputFunc, ok := outputs[output]
	if !ok {
		return fmt.Errorf("unsupported output: %s", output)
	}

	ctx := handleSignal(context.Background())
	innerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	status, err := getDaemonStatus(innerCtx)
	if err == context.DeadlineExceeded {
		return errors.New("timed out after 30 seconds trying to connect to Elastic Agent daemon")
	} else if err == context.Canceled {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to communicate with Elastic Agent daemon: %s", err)
	}

	err = outputFunc(streams.Out, status)
	if err != nil {
		return err
	}
	// exit 0 only if the Elastic Agent daemon is healthy
	if status.Status == client.Healthy {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
	return nil
}

func humanOutput(w io.Writer, status *client.AgentStatus) error {
	fmt.Fprintf(w, "Status: %s\n", status.Status)
	if status.Message == "" {
		fmt.Fprint(w, "Message: (no message)\n")
	} else {
		fmt.Fprintf(w, "Message: %s\n", status.Message)
	}
	if len(status.Applications) == 0 {
		fmt.Fprint(w, "Applications: (none)\n")
	} else {
		fmt.Fprint(w, "Applications:\n")
		for _, app := range status.Applications {
			fmt.Fprintf(w, "  * %s\t(%s)\n", app.Name, app.Status)
			if app.Message == "" {
				fmt.Fprint(w, "    (no message)\n")
			} else {
				fmt.Fprintf(w, "    %s\n", app.Message)
			}
		}
	}
	return nil
}

func jsonOutput(w io.Writer, status *client.AgentStatus) error {
	bytes, err := json.MarshalIndent(status, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", bytes)
	return nil
}

func yamlOutput(w io.Writer, status *client.AgentStatus) error {
	bytes, err := yaml.Marshal(status)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", bytes)
	return nil
}
