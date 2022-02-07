package main

import (
	"os"

	"github.com/elastic/beats/v7/cloudbeat/cmd"

	_ "github.com/elastic/beats/v7/cloudbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
