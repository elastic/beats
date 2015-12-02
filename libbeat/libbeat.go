package main

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/mock"
)

func main() {
	beat.Run(mock.Name, mock.Version, &mock.Mockbeat{})
}
