// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package version

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// Output returns the output when `--yaml` is used.
type Output struct {
	Binary *release.VersionInfo `yaml:"binary"`
	Daemon *release.VersionInfo `yaml:"daemon,omitempty"`
}

// queryDaemon gather version information from a running agent
func queryDaemon() (*release.VersionInfo, error) {
	c := client.New()
	err := c.Connect(context.Background())
	if err != nil {
		return nil, err
	}
	defer c.Disconnect()

	version, err := c.Version(context.Background())
	if err != nil {
		return nil, err
	}
	return &release.VersionInfo{
		Version:   version.Version,
		Commit:    version.Commit,
		BuildTime: version.BuildTime,
		Snapshot:  version.Snapshot,
	}, nil
}

// NewCommandWithArgs returns a new version command.
func NewCommandWithArgs(streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display the version of the elastic-agent.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// the error returned from this function
			var returnErr error
			// prevent RunE from dumping usage on error
			defer func() {
				if returnErr != nil {
					cmd.SilenceErrors = true
					cmd.SilenceUsage = true
				}
			}()
			var daemon *release.VersionInfo

			binary := release.Info()
			binaryOnly, _ := cmd.Flags().GetBool("binary-only")
			if !binaryOnly {
				if d, err := queryDaemon(); err != nil {
					returnErr = fmt.Errorf("could not get version. failed to communicate with running daemon: %w\nUse --binary-only flag to skip trying to retrieve version from running daemon", err)
				} else {
					daemon = d
					if isMismatch(&binary, daemon) {
						fmt.Fprintf(streams.Err, "WARN: the running daemon of Elastic Agent does not match this version.\n")
					}
				}
			}

			outputYaml, _ := cmd.Flags().GetBool("yaml")
			if outputYaml {
				out, err := yaml.Marshal(Output{
					Binary: &binary,
					Daemon: daemon,
				})
				if err != nil {
					return fmt.Errorf("failed to render YAML: %w", err)
				}
				fmt.Fprintf(streams.Out, "%s", out)
				return returnErr
			}

			fmt.Fprintf(streams.Out, "Binary: %s\n", binary.String())
			if binaryOnly {
				return returnErr
			}

			str := "<failed to communicate>"
			if daemon != nil {
				str = daemon.String()
			}
			fmt.Fprintf(streams.Out, "Daemon: %s\n", str)
			return returnErr
		},
	}

	cmd.Flags().Bool("binary-only", false, "Version of current binary only")
	cmd.Flags().Bool("yaml", false, "Output information in YAML format")

	return cmd
}

func isMismatch(a *release.VersionInfo, b *release.VersionInfo) bool {
	if a.Commit != "unknown" && b.Commit != "unknown" {
		return a.Commit != b.Commit
	}
	return a.Version != b.Version || a.BuildTime != b.BuildTime || a.Snapshot != b.Snapshot
}
