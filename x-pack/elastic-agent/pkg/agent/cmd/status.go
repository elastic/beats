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
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

type outputter func(io.Writer, interface{}) error

var statusOutputs = map[string]outputter{
	"human": humanStatusOutput,
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
				fmt.Fprintf(streams.Err, "Error: %v\n%s\n", err, troubleshootMessage())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().String("output", "human", "Output the status information in either human, json, or yaml")

	return cmd
}

func statusCmd(streams *cli.IOStreams, cmd *cobra.Command, args []string) error {
	err := tryContainerLoadPaths()
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	outputFunc, ok := statusOutputs[output]
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

func humanStatusOutput(w io.Writer, obj interface{}) error {
	status, ok := obj.(*client.AgentStatus)
	if !ok {
		return fmt.Errorf("unable to cast %T as *client.AgentStatus", obj)
	}
	return outputStatus(w, status)
}

func outputStatus(w io.Writer, status *client.AgentStatus) error {
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
		tw := tabwriter.NewWriter(w, 4, 1, 2, ' ', 0)
		for _, app := range status.Applications {
			fmt.Fprintf(tw, "  * %s\t(%s)\n", app.Name, app.Status)
			if app.Message == "" {
				fmt.Fprint(tw, "\t(no message)\n")
			} else {
				fmt.Fprintf(tw, "\t%s\n", app.Message)
			}
		}
		tw.Flush()
	}
	return nil
}

func jsonOutput(w io.Writer, out interface{}) error {
	bytes, err := json.MarshalIndent(out, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", bytes)
	return nil
}

func yamlOutput(w io.Writer, out interface{}) error {
	bytes, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", bytes)
	return nil
}
