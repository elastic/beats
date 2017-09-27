package export

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/template"
)

func GenTemplateConfigCmd(name, idxPrefix, beatVersion string) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "template",
		Short: "Export index template to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")
			index, _ := cmd.Flags().GetString("index")

			b, err := instance.NewBeat(name, idxPrefix, beatVersion)
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

			tmpl, err := template.New(b.Info.Version, index, version, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating template: %+v", err)
				os.Exit(1)
			}

			fieldsPath := paths.Resolve(paths.Config, cfg.Fields)
			templateString, err := tmpl.Load(fieldsPath)
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
	genTemplateConfigCmd.Flags().String("index", idxPrefix, "Base index name")

	return genTemplateConfigCmd
}
