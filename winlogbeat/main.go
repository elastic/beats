package main

import (
	"github.com/elastic/libbeat/beat"
	winlogbeat "github.com/elastic/winlogbeat/beat"
)

// Version of Winlogbeat.
var Version = "0.0.1"

// Name of this beat.
var Name = "winlogbeat"

func main() {
	beat.Run(Name, Version, winlogbeat.New())
}
