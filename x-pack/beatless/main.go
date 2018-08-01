package main

import (
	"os"

	"github.com/elastic/beats/x-pack/beatless/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
