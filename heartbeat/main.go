package main

import (
	"os"

	"github.com/elastic/beats/heartbeat/cmd"

	_ "github.com/elastic/beats/heartbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
