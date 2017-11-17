package main

import (
	"os"

	"github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/mock"
)

var RootCmd = cmd.GenRootCmd(mock.Name, mock.Version, mock.New)

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
