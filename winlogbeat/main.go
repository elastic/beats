package main

import (
	"github.com/elastic/beats/libbeat/beat"
	winlogbeat "github.com/elastic/beats/winlogbeat/beat"
)

// Version of Winlogbeat.
var Version = "1.2.0"

// Name of this beat.
var Name = "winlogbeat"

func main() {
	beat.Run(Name, Version, winlogbeat.New())
}
