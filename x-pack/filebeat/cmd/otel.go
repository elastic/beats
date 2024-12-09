// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/elastic/elastic-agent-libs/logp"
)

func OtelCmd() *cobra.Command {
	command := &cobra.Command{
		Short:  "Run this to start filebeat with otel collector",
		Use:    "otel",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logp.NewLogger("filebeat-otel-mode")
			logger.Info("This mode is experimental and unsupported")

			// get filebeat configuration file
			filebeatCfg, _ := cmd.Flags().GetString("config")
			// adds scheme name as prefix
			filebeatCfg = "fb:" + filebeatCfg

			set := getCollectorSettings(filebeatCfg)
			col, err := otelcol.NewCollector(set)
			if err != nil {
				panic(fmt.Errorf("error initializing collector process: %w", err))
			}
			return col.Run(context.Background())
		},
	}

	command.Flags().String("config", "filebeat-otel.yml", "path to filebeat config file")
	return command
}
