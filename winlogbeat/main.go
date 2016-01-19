package main

import (
	"github.com/elastic/beats/libbeat/beat"
	winlogbeat "github.com/elastic/beats/winlogbeat/beat"
	
	"os"
)

// Name of this beat.
var Name = "winlogbeat"

func main() {
	if err := beat.Run(Name, "", winlogbeat.New()); err != nil {
		os.Exit(1)
	}
}
