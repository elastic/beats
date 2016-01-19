package main

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/mock"
	"os"
)

func main() {
	if err := beat.Run(mock.Name, mock.Version, &mock.Mockbeat{}); err != nil {
		os.Exit(1)
	}
}
