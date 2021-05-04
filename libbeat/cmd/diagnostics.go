// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/diagnostics"
)

// Different arguments available to the user
var interval, duration, protocol, host, port, socket, target string

// Argument for the "info" type diagnostic, in case we only want to collect files and not call any API's
// For example when the beat is unable to start or is unavailable.
var logOnly bool

func genDiagCmd(settings instance.Settings, beatCreator beat.Creator) *cobra.Command {
	genDiagCmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Collects diagnostic from beats instances",
		Long: `This command runs diagnostics on a local or remote beats instance depending on the subcommand:

		* The info subcommand collects basic logs and configurations.
		* The metric subcommand, in addition to what is collected from info, also collects useful metrics from a running beat.
		* The profile subcommand, in addition to what is collected from metric, also collects profiling data from a running beat.
	   `,
	}

	genDiagCmd.AddCommand(genDiagInfoCmd(settings, beatCreator))
	genDiagCmd.AddCommand(genDiagMonitorCmd(settings))
	genDiagCmd.AddCommand(genDiagProfileCmd(settings))

	return genDiagCmd
}

// TODO, add validators
func genDiagInfoCmd(settings instance.Settings, beatCreator beat.Creator) *cobra.Command {
	genDiagInfoCmd := &cobra.Command{
		Use:   "info",
		Short: "Collects diagnostics from beats instance",
		Run: func(cmd *cobra.Command, args []string) {
			b, c := initializeBeat(settings)

			diag := diagnostics.NewDiag(b, c)
			diag.Type = "info"
			diag.LogOnly = logOnly
			diag.TargetDir = target
			diag.API.Host = host
			diag.API.Port = port
			diag.API.Socket = socket
			diag.API.Protocol = protocol
			diag.Run()
		},
	}
	genDiagInfoCmd.Flags().StringVar(&protocol, "protocol", "unix", "Which protocol to use, can be http, https, npipe or unix")
	genDiagInfoCmd.Flags().StringVar(&host, "host", "localhost", "Which host address to connect to")
	genDiagInfoCmd.Flags().StringVar(&target, "target", "/tmp", "Target directory to store diagnostic output")
	genDiagInfoCmd.Flags().StringVar(&socket, "socket", "/var/run/filebeat.sock", "Full path to the unix socket used")
	genDiagInfoCmd.Flags().StringVar(&port, "port", "5066", "Which port to connect to")
	genDiagInfoCmd.Flags().BoolVar(&logOnly, "logonly", false, "Only collect logs and configuration files, without API calls, in case beat is not running")
	return genDiagInfoCmd
}

// TODO, add validators
func genDiagMonitorCmd(settings instance.Settings) *cobra.Command {
	genDiagMonitorCmd := &cobra.Command{
		Use:   "monitor",
		Short: "Collects diagnostics and metrics from beats instance",
		Run: func(cmd *cobra.Command, args []string) {
			b, c := initializeBeat(settings)

			diag := diagnostics.NewDiag(b, c)
			diag.Type = "monitor"
			diag.TargetDir = target
			diag.Interval = interval
			diag.Duration = duration
			diag.LogOnly = false
			diag.API.Host = host
			diag.API.Port = port
			diag.API.Socket = socket
			diag.API.Protocol = protocol
			diag.Run()
		},
	}
	genDiagMonitorCmd.Flags().StringVar(&protocol, "protocol", "unix", "Which protocol to use, can be tcp, npipe or unix")
	genDiagMonitorCmd.Flags().StringVar(&host, "host", "localhost", "Which host to connect to")
	genDiagMonitorCmd.Flags().StringVar(&port, "port", "5066", "Which port to connect to")
	genDiagMonitorCmd.Flags().StringVar(&socket, "socket", "/var/run/filebeat.sock", "Full path to the unix socket used")
	genDiagMonitorCmd.Flags().StringVar(&target, "target", "/tmp", "Target directory to store diagnostic output")
	genDiagMonitorCmd.Flags().StringVar(&interval, "interval", "10s", "Metric collection interval")
	genDiagMonitorCmd.Flags().StringVar(&duration, "duration", "10m", "Metric collection duration")
	return genDiagMonitorCmd
}

// TODO, add validators
func genDiagProfileCmd(settings instance.Settings) *cobra.Command {
	genDiagProfileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Collects diagnostics, metrics and profiling data from beats instance",
		Run: func(cmd *cobra.Command, args []string) {
			b, c := initializeBeat(settings)

			diag := diagnostics.NewDiag(b, c)
			diag.Type = "profile"
			diag.TargetDir = target
			diag.Interval = interval
			diag.Duration = duration
			diag.LogOnly = false
			diag.API.Host = host
			diag.API.Port = port
			diag.API.Socket = socket
			diag.API.Protocol = protocol
			diag.Run()
		},
	}
	genDiagProfileCmd.Flags().StringVar(&protocol, "protocol", "unix", "Which protocol to use, can be tcp, npipe or unix")
	genDiagProfileCmd.Flags().StringVar(&host, "host", "localhost", "Which host address or socket path used to connect")
	genDiagProfileCmd.Flags().StringVar(&port, "port", "5066", "Which port to connect to")
	genDiagProfileCmd.Flags().StringVar(&socket, "socket", "/var/run/filebeat.sock", "Full path to the unix socket used")
	genDiagProfileCmd.Flags().StringVar(&target, "target", "/tmp", "Target directory to store diagnostic output")
	genDiagProfileCmd.Flags().StringVar(&interval, "interval", "10s", "Metric collection interval")
	genDiagProfileCmd.Flags().StringVar(&duration, "duration", "10m", "Metric collection duration")
	return genDiagProfileCmd
}

// Initializes a beat instance to get settings, metadata and a copy of the unpacked configuration.
func initializeBeat(settings instance.Settings) (beat *instance.Beat, config map[string]interface{}) {
	b, err := instance.NewInitializedBeat(settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing beat: %s\n", err)
		os.Exit(1)
	}

	err = b.RawConfig.Unpack(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error unpacking configuration: %s\n", err)
		os.Exit(1)
	}
	return b, config
}
