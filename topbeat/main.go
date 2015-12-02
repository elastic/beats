package main

import (
	topbeat "github.com/elastic/topbeat/beat"

	"github.com/elastic/libbeat/beat"
)

// You can overwrite these, e.g.: go build -ldflags "-X main.Version 1.0.0-beta3"
var Version = "1.0.0"
var Name = "topbeat"

func main() {
	beat.Run(Name, Version, topbeat.New())
}
