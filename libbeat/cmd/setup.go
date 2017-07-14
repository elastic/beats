package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
)

func genSetupCmd(name, version string, beatCreator beat.Creator) *cobra.Command {
	setup := cobra.Command{
		Use:   "setup",
		Short: "Setup index template and dashboards",
		Long: `This command does initial setup of the environment:

 * Index mapping template in Elasticsearch to ensure fields are mapped.
 * Kibana dashboards (where available).
`,
		Run: func(cmd *cobra.Command, args []string) {
			beat, err := beat.New(name, version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			template, _ := cmd.Flags().GetBool("template")
			dashboards, _ := cmd.Flags().GetBool("dashboards")
			machineLearning, _ := cmd.Flags().GetBool("machine-learning")

			// No flags: setup all
			if !template && !dashboards && !machineLearning {
				template = true
				dashboards = true
				machineLearning = true
			}

			if err = beat.Setup(beatCreator, template, dashboards, machineLearning); err != nil {
				os.Exit(1)
			}
		},
	}

	setup.Flags().Bool("template", false, "Setup index template only")
	setup.Flags().Bool("dashboards", false, "Setup dashboards only")
	setup.Flags().Bool("machine-learning", false, "Setup machine learning job configurations only")

	return &setup
}
