// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"

	"github.com/elastic/beats/v8/x-pack/auditbeat/cmd"

	// Register modules.
	_ "github.com/elastic/beats/v8/auditbeat/module/auditd"
	_ "github.com/elastic/beats/v8/auditbeat/module/file_integrity"

	// Register includes.
	_ "github.com/elastic/beats/v8/auditbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
