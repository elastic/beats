package main

import (
	"os"

	"github.com/elastic/beats/packetbeat/cmd"
)

var Name = "packetbeat"

// Setups and Runs Packetbeat
func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
