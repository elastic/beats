package test

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/instance"
)

func GenTestConfigCmd(name, version string, beatCreator beat.Creator) *cobra.Command {
	configTestCmd := cobra.Command{
		Use:   "config",
		Short: "Test configuration settings",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := instance.NewBeat(name, "", version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			if err = b.TestConfig(beatCreator); err != nil {
				os.Exit(1)
			}
		},
	}

	return &configTestCmd
}
