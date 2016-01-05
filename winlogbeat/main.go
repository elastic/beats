package main

import (
	"github.com/elastic/beats/libbeat/beat"
	winlogbeat "github.com/elastic/beats/winlogbeat/beat"
)

// Name of this beat.
var Name = "winlogbeat"

func main() {
	beat.Run(Name, "", winlogbeat.New())
}
