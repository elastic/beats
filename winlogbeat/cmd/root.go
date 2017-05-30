package cmd

import cmd "github.com/elastic/beats/libbeat/cmd"

// Name of this beat
var Name = "winlogbeat"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmd(Name, RunCmd)
