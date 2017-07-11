package test

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/testing"
)

func GenTestOutputCmd(name, beatVersion string) *cobra.Command {
	return &cobra.Command{
		Use:   "output",
		Short: "Test output works with current settings",
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

			output, err := outputs.Load(b.Info, b.Config.Output.Name(), b.Config.Output.Config())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing output: %s\n", err)
				os.Exit(1)
			}

			for _, client := range output.Clients {
				tClient, ok := client.(testing.Testable)
				if !ok {
					fmt.Printf("%s output doesn't support testing\n", b.Config.Output.Name())
					os.Exit(1)
				}

				// Perform test:
				tClient.Test(testing.NewConsoleDriver(os.Stdout))
			}
		},
	}
}
