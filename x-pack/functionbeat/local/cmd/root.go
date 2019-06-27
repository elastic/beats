// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	libcmd "github.com/elastic/beats/libbeat/cmd"
	funcmd "github.com/elastic/beats/x-pack/functionbeat/function/cmd"
)

// Name of this beat
var Name = "functionbeat-local"

// RootCmd to handle functionbeat
var RootCmd *libcmd.BeatsRootCmd

func init() {
	RootCmd = funcmd.GenRootCmdWithBeatName(Name)
}
