package main

import (
	"os"

	"{beat_path}/cmd"

	// Make sure all your modules and metricsets are linked in this file
	_ "{beat_path}/include"
)


func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
