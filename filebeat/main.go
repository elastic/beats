package main

import (
	"os"

	filebeat "github.com/elastic/beats/filebeat/beat"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
)

var Version = "1.0.1"
var Name = "filebeat"

var GlobalBeat *beat.Beat

// The basic model of execution:
// - prospector: finds files in paths/globs to harvest, starts harvesters
// - harvester: reads a file, sends events to the spooler
// - spooler: buffers events until ready to flush to the publisher
// - publisher: writes to the network, notifies registrar
// - registrar: records positions of files read
// Finally, prospector uses the registrar information, on restart, to
// determine where in each file to restart a harvester.

func main() {

	// Create Beater object
	fb := &filebeat.Filebeat{}

	// Initi beat objectefile
	b := beat.NewBeat(Name, Version, fb)

	// Additional command line args are used to overwrite config options
	b.CommandLineSetup()

	// Loads base config
	b.LoadConfig()

	// Configures beat
	err := fb.Config(b)
	if err != nil {
		logp.Critical("Config error: %v", err)
		os.Exit(1)
	}

	// Run beat. This calls first beater.Setup,
	// then beater.Run and beater.Cleanup in the end
	b.Run()

}
