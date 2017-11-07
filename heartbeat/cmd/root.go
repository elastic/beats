package cmd

import (
	// register default heartbeat monitors
	_ "github.com/elastic/beats/heartbeat/monitors/defaults"

	"github.com/elastic/beats/heartbeat/beater"
	cmd "github.com/elastic/beats/libbeat/cmd"
)

// Name of this beat
var Name = "heartbeat"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmd(Name, "", beater.New)
