package main

import (
	topbeat "github.com/elastic/beats/topbeat/beat"

	"github.com/elastic/beats/libbeat/beat"
	"os"
)

var Name = "topbeat"

func main() {
	if err := beat.Run(Name, "", topbeat.New()); err != nil {
		os.Exit(1)
	}
}
