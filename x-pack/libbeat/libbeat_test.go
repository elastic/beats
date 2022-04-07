// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"flag"
	"testing"

	"github.com/elastic/beats/v8/libbeat/tests/system/template"
)

var systemTest *bool

func init() {
	testing.Init()
	systemTest = flag.Bool("systemTest", false, "Set to true when running system tests")

	RootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("systemTest"))
	RootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("test.coverprofile"))
}

// Test started when the test binary is started
func TestSystem(t *testing.T) {
	if *systemTest {
		main()
	}
}

func TestTemplate(t *testing.T) {
	template.TestTemplate(t, "mockbeat", true)
}
