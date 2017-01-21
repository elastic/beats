package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/mock"
)

func main() {
	if err := beat.Run(mock.Name, mock.Version, mock.New); err != nil {
		os.Exit(1)
	}
}
