// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/mock"
	xpackcmd "github.com/elastic/beats/v7/x-pack/libbeat/cmd"
)

// RootCmd to test libbeat
var RootCmd = cmd.GenRootCmdWithSettings(mock.New, mock.Settings)

func main() {
	xpackcmd.AddXPack(RootCmd, mock.Name)
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
