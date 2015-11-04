package main

import (
	"os"

	eventbeat "github.com/elastic/eventbeat/beat"
	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/logp"
)

var Version = "0.0.1"
var Name = "eventbeat"

var GlobalBeat *beat.Beat

func main() {
	// Create Beater object
	fb := &eventbeat.Eventbeat{}

	// Initialize beat objectefile
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
