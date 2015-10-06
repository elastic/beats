package main

import (
	"github.com/elastic/libbeat/beat"
)

// Setups and Runs Packetbeat
func main() {

	// Create Beater object
	pb := &Packetbeat{}

	// Initi beat objectefile
	b := beat.NewBeat(Name, Version, pb)

	// Additional command line args are used to overwrite config options
	pb.CmdLineArgs = fetchAdditionalCmdLineArgs()

	// Base CLI flags
	b.CommandLineSetup()

	// Beat CLI flags
	pb.CliFlags(b)

	// Loads base config
	b.LoadConfig()

	// Configures beat
	pb.Config(b)

	// Run beat. This calls first beater.Setup,
	// then beater.Run and beater.Cleanup in the end
	b.Run()
}
