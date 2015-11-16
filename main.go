package main

import (
	"os"

	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/logp"
	winlogbeat "github.com/elastic/winlogbeat/beat"
)

var Version = "0.0.1"
var Name = "winlogbeat"

var GlobalBeat *beat.Beat

func main() {
	// Create Beater object
	fb := &winlogbeat.Winlogbeat{}

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
