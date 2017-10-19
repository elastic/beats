package cmd

import (
	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/export"
)

func genExportCmd(name, idxPrefix, beatVersion string) *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export current config or index template",
	}

	exportCmd.AddCommand(export.GenExportConfigCmd(name, idxPrefix, beatVersion))
	exportCmd.AddCommand(export.GenTemplateConfigCmd(name, idxPrefix, beatVersion))

	return exportCmd
}
