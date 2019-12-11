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

// +build !windows,!integration

package log

import (
	"testing"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common/match"

	"github.com/stretchr/testify/assert"
)

var matchTests = []struct {
	file         string
	paths        []string
	excludeFiles []match.Matcher
	result       bool
}{
	{
		"test/test.log",
		[]string{"test/*"},
		nil,
		true,
	},
	{
		"notest/test.log",
		[]string{"test/*"},
		nil,
		false,
	},
	{
		"test/test.log",
		[]string{"test/*.log"},
		nil,
		true,
	},
	{
		"test/test.log",
		[]string{"test/*.nolog"},
		nil,
		false,
	},
	{
		"test/test.log",
		[]string{"test/*"},
		[]match.Matcher{match.MustCompile("test.log")},
		false,
	},
	{
		"test/test.log",
		[]string{"test/*"},
		[]match.Matcher{match.MustCompile("test2.log")},
		true,
	},
}

func TestMatchFile(t *testing.T) {
	for _, test := range matchTests {

		p := Input{
			config: config{
				Paths:        test.paths,
				ExcludeFiles: test.excludeFiles,
			},
		}

		assert.Equal(t, test.result, p.matchesFile(test.file))
	}
}

var initStateTests = []struct {
	states []file.State // list of states
	paths  []string     // input glob
	count  int          // expected states in input
}{
	{
		[]file.State{
			{Source: "test"},
		},
		[]string{"test"},
		1,
	},
	{
		[]file.State{
			{Source: "notest"},
		},
		[]string{"test"},
		0,
	},
	{
		[]file.State{
			{Source: "test1.log", Id: "1"},
			{Source: "test2.log", Id: "2"},
		},
		[]string{"*.log"},
		2,
	},
	{
		[]file.State{
			{Source: "test1.log", Id: "1"},
			{Source: "test2.log", Id: "2"},
		},
		[]string{"test1.log"},
		1,
	},
	{
		[]file.State{
			{Source: "test1.log", Id: "1"},
			{Source: "test2.log", Id: "2"},
		},
		[]string{"test.log"},
		0,
	},
	{
		[]file.State{
			{Source: "test1.log", Id: "1"},
			{Source: "test2.log", Id: "1"},
		},
		[]string{"*.log"},
		1, // Expecting only 1 state because of same inode (this is only a theoretical case)
	},
}

// TestInit checks that the correct states are in an input after the init phase
// This means only the ones that match the glob and not exclude files
func TestInit(t *testing.T) {
	for _, test := range initStateTests {
		p := Input{
			config: config{
				Paths: test.paths,
			},
			states: file.NewStates(),
			outlet: TestOutlet{},
		}

		// Set states to finished
		for i, state := range test.states {
			state.Finished = true
			test.states[i] = state
		}

		err := p.loadStates(test.states)
		assert.NoError(t, err)
		assert.Equal(t, test.count, p.states.Count())
	}
}
