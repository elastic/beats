package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/mock"
)

func main() {
	err := beat.Run(mock.Name, mock.Version, &mock.Mockbeat{})
	if err != nil {
		os.Exit(1)
	}
}
