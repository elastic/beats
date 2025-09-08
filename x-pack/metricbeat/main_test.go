// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package main

// This file is mandatory as otherwise the metricbeat.test binary is not generated correctly.
import (
	"flag"
	"os"
	"testing"

	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/tests/system/template"
	mbcmd "github.com/elastic/beats/v7/x-pack/metricbeat/cmd"
)

var (
	systemTest *bool
	mbCommand  *cmd.BeatsRootCmd
)

func init() {
	testing.Init()
	systemTest = flag.Bool("systemTest", false, "Set to true when running system tests")
	mbCommand = mbcmd.Initialize()
	mbCommand.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("systemTest"))
	mbCommand.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("test.coverprofile"))
}

// Test started when the test binary is started. Only calls main.
func TestSystem(t *testing.T) {
	if *systemTest {
		if err := mbCommand.Execute(); err != nil {
			os.Exit(1)
		}
	}
}

func TestTemplate(t *testing.T) {
	template.TestTemplate(t, mbCommand.Name(), true)
}
