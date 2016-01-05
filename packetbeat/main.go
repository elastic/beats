package main

import (
	packetbeat "github.com/elastic/beats/packetbeat/beat"

	"github.com/elastic/beats/libbeat/beat"
)

var Name = "packetbeat"

// Setups and Runs Packetbeat
func main() {
	beat.Run(Name, "", packetbeat.New())
}
