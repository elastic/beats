package main

import (
	topbeat "github.com/elastic/beats/topbeat/beat"

	"github.com/elastic/beats/libbeat/beat"
)

// You can overwrite these, e.g.: go build -ldflags "-X main.Version 1.0.0-beta3"
var Version = "1.0.0"
var Name = "topbeat"

func main() {
	beat.Run(Name, Version, topbeat.New())
}
