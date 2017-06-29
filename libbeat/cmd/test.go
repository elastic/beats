package cmd

import (
	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
)

func genTestCmd(name, beatVersion string, beatCreator beat.Creator) *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "test",
		Short: "Test config",
	}

	return exportCmd
}
