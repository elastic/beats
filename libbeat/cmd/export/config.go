package export

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/cmd/instance"
)

func GenExportConfigCmd(name, idxPrefix, beatVersion string) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Export current config to stdout",
		Run: func(cmd *cobra.Command, args []string) {
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
