package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"
	winlogbeat "github.com/elastic/beats/winlogbeat/beat"
)

// Name of this beat.
var Name = "winlogbeat"

func main() {
	err := beat.Run(Name, "", winlogbeat.New())
	if err != nil {
		os.Exit(1)
	}
}
