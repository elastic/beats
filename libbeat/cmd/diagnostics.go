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

var interval, duration, protocol, host string

var (
	logName = "diagnostics"
)

//TODO Better descriptions
func genDiagCmd(settings instance.Settings, beatCreator beat.Creator) *cobra.Command {
	diagCmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Run Diagnostics",
	}

	diagCmd.AddCommand(genDiagInfoCmd(settings, beatCreator))
	diagCmd.AddCommand(genDiagMonitorCmd(settings))
	diagCmd.AddCommand(genDiagProfileCmd(settings))

	return diagCmd
}

// TODO, all the cmd's does pretty much the same, maybe create it into a function instead?
func genDiagInfoCmd(settings instance.Settings, beatCreator beat.Creator) *cobra.Command {
	genDiagInfoCmd := &cobra.Command{
		Use:   "info",
		Short: "Export defined dashboard to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			var config map[string]interface{}
			err = b.RawConfig.Unpack(&config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error unpacking configuration: %s\n", err)
				os.Exit(1)
			}
			diag := diagnostics.NewDiag(b, config)
			diag.Type = "info"
			diag.Interval = interval
			diag.Duration = duration
			diag.HTTP.Host = host
			diag.HTTP.Protocol = protocol
			diag.GetInfo()
		},
	}
	genDiagInfoCmd.Flags().StringVar(&protocol, "protocol", "unix", "Which protocol to use, can be tcp, npipe or unix")
	genDiagInfoCmd.Flags().StringVar(&host, "host", "localhost", "Which host to connect to")
	return genDiagInfoCmd
}

func genDiagMonitorCmd(settings instance.Settings) *cobra.Command {
	genDiagMonitorCmd := &cobra.Command{
		Use:   "monitor",
		Short: "Export defined dashboard to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			var config map[string]interface{}
			err = b.RawConfig.Unpack(&config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error unpacking configuration: %s\n", err)
				os.Exit(1)
			}

			diag := diagnostics.NewDiag(b, config)
			diag.Type = "monitor"
			diag.Interval = interval
			diag.Duration = duration
			diag.HTTP.Host = host
			diag.HTTP.Protocol = protocol
			diag.GetMonitor()
		},
	}
	genDiagMonitorCmd.Flags().StringVar(&protocol, "protocol", "unix", "Which protocol to use, can be tcp, npipe or unix")
	genDiagMonitorCmd.Flags().StringVar(&host, "host", "localhost", "Which host to connect to")
	genDiagMonitorCmd.Flags().StringVar(&interval, "interval", "10s", "Metric collection interval")
	genDiagMonitorCmd.Flags().StringVar(&duration, "duration", "10m", "Metric collection duration")
	return genDiagMonitorCmd
}

func genDiagProfileCmd(settings instance.Settings) *cobra.Command {
	genDiagProfileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Export defined dashboard to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			var config map[string]interface{}
			err = b.RawConfig.Unpack(&config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error unpacking configuration: %s\n", err)
				os.Exit(1)
			}

			diag := diagnostics.NewDiag(b, config)
			diag.Type = "profile"
			diag.Interval = interval
			diag.Duration = duration
			diag.HTTP.Host = host
			diag.HTTP.Protocol = protocol
			diag.GetProfile()
		},
	}
	genDiagProfileCmd.Flags().StringVar(&protocol, "protocol", "unix", "Which protocol to use, can be tcp, npipe or unix")
	genDiagProfileCmd.Flags().StringVar(&host, "host", "localhost", "Which host to connect to")
	genDiagProfileCmd.Flags().StringVar(&interval, "interval", "10s", "Metric collection interval")
	genDiagProfileCmd.Flags().StringVar(&duration, "duration", "10m", "Metric collection duration")
	return genDiagProfileCmd
}
