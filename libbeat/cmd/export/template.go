package export

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/template"
)

func GenTemplateConfigCmd(name, beatVersion string, beatCreator beat.Creator) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "template",
		Short: "Export index template to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")
			index, _ := cmd.Flags().GetString("index")

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

			cfg := template.DefaultConfig
			if b.Config.Template.Enabled() {
				err = b.Config.Template.Unpack(&cfg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting template settings: %+v", err)
					os.Exit(1)
				}
			}

			tmpl, err := template.New(b.Info.Version, version, index, cfg.Settings)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating template: %+v", err)
				os.Exit(1)
			}

			templateString, err := tmpl.Load(cfg.Fields)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating template: %+v", err)
				os.Exit(1)
			}

			_, err = os.Stdout.WriteString(templateString.StringToPrint() + "\n")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing template: %+v", err)
				os.Exit(1)
			}
		},
	}

	genTemplateConfigCmd.Flags().String("es.version", beatVersion, "Elasticsearch version")
	genTemplateConfigCmd.Flags().String("index", name, "Base index name")

	return genTemplateConfigCmd
}
