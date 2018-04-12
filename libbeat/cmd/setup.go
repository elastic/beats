package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/instance"
)

func genSetupCmd(name, idxPrefix, version string, beatCreator beat.Creator) *cobra.Command {
	setup := cobra.Command{
		Use:   "setup",
		Short: "Setup index template, dashboards and ML jobs",
		Long: `This command does initial setup of the environment:

 * Index mapping template in Elasticsearch to ensure fields are mapped.
 * Kibana dashboards (where available).
 * ML jobs (where available).
 * Ingest pipelines (where available).
`,
		Run: func(cmd *cobra.Command, args []string) {
			beat, err := instance.NewBeat(name, idxPrefix, version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			template, _ := cmd.Flags().GetBool("template")
			dashboards, _ := cmd.Flags().GetBool("dashboards")
			machineLearning, _ := cmd.Flags().GetBool("machine-learning")
			pipelines, _ := cmd.Flags().GetBool("pipelines")

			// No flags: setup all
			if !template && !dashboards && !machineLearning && !pipelines {
				template = true
				dashboards = true
				machineLearning = true
			}

			if err = beat.Setup(beatCreator, template, dashboards, machineLearning, pipelines); err != nil {
				os.Exit(1)
			}
		},
	}

	setup.Flags().Bool("template", false, "Setup index template only")
	setup.Flags().Bool("dashboards", false, "Setup dashboards only")
	setup.Flags().Bool("machine-learning", false, "Setup machine learning job configurations only")
	setup.Flags().Bool("pipelines", false, "Setup Ingest pipelines only")

	return &setup
}
