package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
)

// You can overwrite these, e.g.: go build -ldflags "-X main.Version 1.0.0-beta3"
var Version = "1.0.1"
var Name = "topbeat"

func main() {

	tb := &Topbeat{}

	b := beat.NewBeat(Name, Version, tb)

	b.CommandLineSetup()

	b.LoadConfig()
	err := tb.Config(b)
	if err != nil {
		logp.Critical("Config error: %v", err)
		os.Exit(1)
	}

	b.Run()

}
