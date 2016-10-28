package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"

	"github.com/elastic/beats/heartbeat/beater"

	// register default heartbeat monitors
	_ "github.com/elastic/beats/heartbeat/monitors/defaults"
)

func main() {
	err := beat.Run("heartbeat", "", beater.New)
	if err != nil {
		os.Exit(1)
	}
}
