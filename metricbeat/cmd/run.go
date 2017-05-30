package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/metricbeat/beater"
)

// RunCmd runs the beat
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run " + Name,
	Run: func(cmd *cobra.Command, args []string) {
		if err := beat.Run(Name, "", beater.New); err != nil {
			os.Exit(1)
		}
	},
}
