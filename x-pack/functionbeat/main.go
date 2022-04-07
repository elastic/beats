// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"

	_ "github.com/elastic/beats/v8/x-pack/functionbeat/include" // imports features
	"github.com/elastic/beats/v8/x-pack/functionbeat/manager/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
