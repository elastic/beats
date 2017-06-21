package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
)

func genConfigTestCmd(name, version string, beatCreator beat.Creator) *cobra.Command {
	configTestCmd := cobra.Command{
		Use:   "configtest",
		Short: "Test configuration settings",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := beat.New(name, version)
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
