package cmd

import cmd "github.com/elastic/beats/libbeat/cmd"
import "github.com/elastic/beats/heartbeat/beater"

// Name of this beat
var Name = "heartbeat"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmd(Name, "", beater.New)
