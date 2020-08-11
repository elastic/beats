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

// NewCommandWithArgs returns a new version command.
func NewCommandWithArgs(streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display the version of the elastic-agent.",
		Run: func(cmd *cobra.Command, _ []string) {
			var daemon *release.VersionInfo
			var daemonError error

			binary := release.Info()
			binaryOnly, _ := cmd.Flags().GetBool("binary-only")
			if !binaryOnly {
				c := client.New()
				daemonError = c.Connect(context.Background())
				if daemonError == nil {
					defer c.Disconnect()

					var version client.Version
					version, daemonError = c.Version(context.Background())
					if daemonError == nil {
						daemon = &release.VersionInfo{
							Version:   version.Version,
							Commit:    version.Commit,
							BuildTime: version.BuildTime,
							Snapshot:  version.Snapshot,
						}
					}
				}
			}
			if daemonError != nil {
				fmt.Fprintf(streams.Err, "Failed talking to running daemon: %s\n", daemonError)
			}

			outputYaml, _ := cmd.Flags().GetBool("yaml")
			if outputYaml {
				p := Output{
					Binary: &binary,
					Daemon: daemon,
				}
				out, err := yaml.Marshal(p)
				if err != nil {
					fmt.Fprintf(streams.Err, "Failed to render YAML: %s\n", err)
				}
				fmt.Fprintf(streams.Out, "%s", out)
				return
			}

			if !binaryOnly {
				mismatch := false
				str := "<failed to communicate>"
				if daemon != nil {
					str = daemon.String()
					mismatch = isMismatch(&binary, daemon)
				}
				if mismatch {
					fmt.Fprintf(streams.Err, "WARN: Then running daemon of Elastic Agent does not match this version.\n")
				}
				fmt.Fprintf(streams.Out, "Daemon: %s\n", str)
			}
			fmt.Fprintf(streams.Out, "Binary: %s\n", binary.String())
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
