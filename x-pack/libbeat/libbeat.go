// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"

	"github.com/menderesk/beats/v7/libbeat/cmd"
	"github.com/menderesk/beats/v7/libbeat/mock"
	_ "github.com/menderesk/beats/v7/x-pack/libbeat/include"
)

// RootCmd to test libbeat
var RootCmd = cmd.GenRootCmdWithSettings(mock.New, mock.Settings)

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
