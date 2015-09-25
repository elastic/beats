package main

import (
	"github.com/elastic/libbeat/beat"
	. "github.com/elastic/libbeat/mock"
)

// Main file is only used for testing.
func main() {

	mock := &Mockbeat{}
	b := beat.NewBeat(Name, Version, mock)
	b.CommandLineSetup()
	b.LoadConfig()
	mock.Config(b)
	b.Run()
}
