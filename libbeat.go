package main

import (
	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/mock"
)

func main() {
	beat.Run(mock.Name, mock.Version, &mock.Mockbeat{})
}
