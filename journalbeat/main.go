package main

import (
	"os"

	"github.com/elastic/xbeats/journalbeat/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
