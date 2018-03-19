package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func genCompletionCmd(name, version string, rootCmd *BeatsRootCmd) *cobra.Command {
	completionCmd := cobra.Command{
		Use:   "completion SHELL",
		Short: "Output shell completion code for the specified shell (bash and zsh only by the moment)",
		// We don't want to expose this one in help:
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				fmt.Println("Expected one argument with the desired shell")
				os.Exit(1)
			}

			switch args[0] {
			case "bash":
				rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				rootCmd.GenZshCompletion(os.Stdout)
			default:
				fmt.Printf("Unknown shell %s, only bash and zsh are available\n", args[0])
				os.Exit(1)
			}
		},
	}

	return &completionCmd
}
