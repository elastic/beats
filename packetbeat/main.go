package main

import (
	packetbeat "github.com/elastic/beats/packetbeat/beat"

	"github.com/elastic/beats/libbeat/beat"
)

// You can overwrite these, e.g.: go build -ldflags "-X main.Version 1.0.0-beta3"
var Version = "1.1.2"
var Name = "packetbeat"

// Setups and Runs Packetbeat
func main() {
	beat.Run(Name, Version, packetbeat.New())
}
