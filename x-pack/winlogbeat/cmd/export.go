// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/winlogbeat/module"
)

// GenTemplateConfigCmd is the command used to export the elasticsearch template.
func GenExportPipelineCmd(settings instance.Settings) *cobra.Command {
	genExportPipelineCmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Export pipelines",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")
			dir, _ := cmd.Flags().GetString("dir")

			if version == "" {
				fatalf("--es.version is required")
			}

			ver, err := common.NewVersion(version)
			if err != nil {
				fatalf("Unable to parse ES version from %s: %+v", version, err)
			}

			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fatalf("Failed to initialize 'pipeline' command: %+v", err)
			}

			err = module.ExportPipelines(b.Info, *ver, dir)
			if err != nil {
				fatalf("Failed to export pipelines: %+v", err)
			}

			fmt.Fprintf(os.Stdout, "Exported pipelines")
		},
	}

	genExportPipelineCmd.Flags().String("es.version", settings.Version, "Elasticsearch version (required)")
	genExportPipelineCmd.Flags().String("dir", "", "Specify directory for exporting pipelines. Default is current directory.")

	return genExportPipelineCmd
}

func fatalf(msg string, vs ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, vs...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}
