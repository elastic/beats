package cmd

import (
	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/export"
)

func genExportCmd(name, beatVersion string, beatCreator beat.Creator) *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export current config or index template",
	}

	exportCmd.AddCommand(export.GenExportConfigCmd(name, beatVersion, beatCreator))
	exportCmd.AddCommand(export.GenTemplateConfigCmd(name, beatVersion, beatCreator))

	return exportCmd
}
