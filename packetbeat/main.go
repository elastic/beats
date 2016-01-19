package main

import (
	packetbeat "github.com/elastic/beats/packetbeat/beat"

	"github.com/elastic/beats/libbeat/beat"
	"os"
)

var Name = "packetbeat"

// Setups and Runs Packetbeat
func main() {
	if err := beat.Run(Name, "", packetbeat.New()); err != nil {
		os.Exit(1)
	}
}