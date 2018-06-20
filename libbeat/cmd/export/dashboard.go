package export

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/kibana"
)

// GenDashboardCmd is the command used to export a dashboard.
func GenDashboardCmd(name, idxPrefix, beatVersion string) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Export defined dashboard to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			dashboard, _ := cmd.Flags().GetString("id")

			b, err := instance.NewBeat(name, idxPrefix, beatVersion)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating beat: %s\n", err)
				os.Exit(1)
			}
			err = b.Init()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			// Use empty config to use default configs if not set
			if b.Config.Kibana == nil {
				b.Config.Kibana = common.NewConfig()
			}

			client, err := kibana.NewKibanaClient(b.Config.Kibana)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating Kibana client: %+v\n", err)
				os.Exit(1)
			}

			result, err := client.GetDashboard(dashboard)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting dashboard: %+v\n", err)
				os.Exit(1)
			}
			fmt.Println(result.StringToPrint())
		},
	}

	genTemplateConfigCmd.Flags().String("id", "", "Dashboard id")

	return genTemplateConfigCmd
}
