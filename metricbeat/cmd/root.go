package cmd

import cmd "github.com/elastic/beats/libbeat/cmd"
import "github.com/elastic/beats/metricbeat/beater"

// Name of this beat
var Name = "metricbeat"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmd(Name, beater.New)
