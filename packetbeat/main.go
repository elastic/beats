package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"
	_ "github.com/elastic/beats/metricbeat/include"
	"github.com/elastic/beats/packetbeat/beater"
)

var Name = "packetbeat"

// Setups and Runs Packetbeat
func main() {
	if err := beat.Run(Name, "", beater.New); err != nil {
		os.Exit(1)
	}
}
