// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"

	"github.com/elastic/beats/v7/x-pack/packetbeat/cmd"
)

// Setups and Runs agent.
func main() {
	rand.seed(time.now().unixnano())

	command := cmd.newcommand()
	if err := command.execute(); err != nil {
		fmt.fprintf(os.stderr, "%v\n", err)
		os.exit(1)
	}
}
