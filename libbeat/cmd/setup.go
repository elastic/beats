package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
)

func genSetupCmd(name string) *cobra.Command {
	setup := cobra.Command{
		Use:   "setup",
		Short: "Setup index template and dashboards",
		Long: `This command does initial setup of the environment:

 * Index mapping template in Elasticsearch to ensure fields are mapped.
 * Kibana dashboards (where available).
`,
		Run: func(cmd *cobra.Command, args []string) {
			beat, err := beat.New(name, "")
			if err != nil {
				os.Exit(1)
			}

			template, _ := cmd.Flags().GetBool("template")
			dashboards, _ := cmd.Flags().GetBool("dashboards")

			// No flags: setup all
			if !template && !dashboards {
				template = true
				dashboards = true
			}

			beat.Setup(template, dashboards)
		},
	}

	setup.Flags().Bool("template", false, "Setup index template only")
	setup.Flags().Bool("dashboards", false, "Setup dashboards only")

	return &setup
}
