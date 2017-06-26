package export

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/beat"
)

func GenExportConfigCmd(name, beatVersion string, beatCreator beat.Creator) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Export current config to stdout",
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

			var config map[string]interface{}
			err = b.RawConfig.Unpack(&config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error unpacking config")
				os.Exit(1)
			}
			res, err := yaml.Marshal(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error converting config to YAML format")
				os.Exit(1)
			}

			os.Stdout.Write(res)
		},
	}
}
