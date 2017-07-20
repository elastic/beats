package export

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/dashboards"
	"github.com/elastic/beats/libbeat/paths"
)

func GenExportDashboardsCmd(name, beatVersion string, beatCreator beat.Creator) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "dashboards [dashboard ids]",
		Short: "Export dashboards",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := beat.New(name, beatVersion)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}
			err = b.Init()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			for _, dashboardID := range args {
				err := dashboards.ExportDashboards(name, beatVersion, paths.Resolve(paths.Home, ""),
					b.Config.Kibana, b.Config.Dashboards, dashboardID, os.Stdout)

				if err != nil {
					fmt.Fprintf(os.Stderr, "Error exporting dashboard %s: %s\n", dashboardID, err)
					os.Exit(1)
				}
			}
		},
	}

	return genTemplateConfigCmd
}
