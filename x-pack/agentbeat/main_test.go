// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package main

// This file is mandatory as otherwise the agentbeat.test binary is not generated correctly.
import (
	"flag"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/agentbeat/cmd"
)

var (
	systemTest *bool
	abCommand  *cobra.Command
)

func init() {
	testing.Init()
	systemTest = flag.Bool("systemTest", false, "Set to true when running system tests")
	abCommand = cmd.AgentBeat()
	abCommand.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("systemTest"))
	abCommand.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("test.coverprofile"))
}

// Test started when the test binary is started. Only calls main.
func TestSystem(t *testing.T) {
	if *systemTest {
		if err := abCommand.Execute(); err != nil {
			os.Exit(1)
		}
	}
}
