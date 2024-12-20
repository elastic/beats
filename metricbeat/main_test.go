// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

// This file is mandatory as otherwise the packetbeat.test binary is not generated correctly.

import (
	"flag"
	"os"
	"testing"

	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/tests/system/template"
	mbcmd "github.com/elastic/beats/v7/metricbeat/cmd"
)

var (
	systemTest *bool
	mbCommand  *cmd.BeatsRootCmd
)

func init() {
	testing.Init()
	systemTest = flag.Bool("systemTest", false, "Set to true when running system tests")
	mbCommand = mbcmd.Initialize(mbcmd.MetricbeatSettings(""))
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
	template.TestTemplate(t, mbCommand.Name(), false)
}
