package main

import (
	"os"

	"github.com/elastic/beats/heartbeat/cmd"

	// register default heartbeat monitors
	_ "github.com/elastic/beats/heartbeat/monitors/defaults"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
