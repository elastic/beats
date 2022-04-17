// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package main

// This file is mandatory as otherwise the packetbeat.test binary is not generated correctly.
import (
	"flag"
	"testing"

	"github.com/menderesk/beats/v7/libbeat/tests/system/template"
	"github.com/menderesk/beats/v7/x-pack/packetbeat/cmd"
)

var systemTest *bool

func init() {
	testing.Init()
	systemTest = flag.Bool("systemTest", false, "Set to true when running system tests")
	cmd.RootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("systemTest"))
	cmd.RootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("test.coverprofile"))
}

// Test started when the test binary is started. Only calls main.
func TestSystem(t *testing.T) {
	if *systemTest {
		main()
	}
}

func TestTemplate(t *testing.T) {
	template.TestTemplate(t, cmd.Name, true)
}
