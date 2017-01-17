package main

import (
	"os"

	"github.com/elastic/beats/packetbeat/cmd"
)

// Setups and Runs Packetbeat
func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
