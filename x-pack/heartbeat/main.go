// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"

	_ "github.com/menderesk/beats/v7/heartbeat/include"
	"github.com/menderesk/beats/v7/x-pack/heartbeat/cmd"
	_ "github.com/menderesk/beats/v7/x-pack/heartbeat/monitors/browser"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
