package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RunWith wrap cli function with an error handler instead of having the code exit early.
func RunWith(
	fn func(cmd *cobra.Command, args []string) error,
) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := fn(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}
