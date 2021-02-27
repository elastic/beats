// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package main

// This file is mandatory as otherwise the agent.test binary is not generated correctly.
import (
	"flag"
	"testing"

	"github.com/spf13/cobra"
)

var systemTest *bool

func init() {
	testing.Init()

	cmd := &cobra.Command{
		Use: "elastic-agent [subcommand]",
	}

	systemTest = flag.Bool("systemTest", false, "Set to true when running system tests")
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("systemTest"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("test.coverprofile"))
}

// Test started when the test binary is started. Only calls main.
func TestSystem(t *testing.T) {
	if *systemTest {
		main()
	}
}
