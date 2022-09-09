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

//go:build !integration
// +build !integration

package file

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var cleanupTests = []struct {
	title        string
	state        State
	countBefore  int
	cleanupCount int
	countAfter   int
}{
	{
		"Finished and TTL set to 0",
		State{
			TTL:      0,
			Finished: true,
		},
		1, 1, 0,
	},
	{
		"Unfinished but TTL set to 0",
		State{
			TTL:      0,
			Finished: false,
		},
		1, 0, 1,
	},
	{
		"TTL = -1 means not expiring",
		State{
			TTL:      -1,
			Finished: true,
		},
		1, 0, 1,
	},
	{
		"Expired and finished",
		State{
			TTL:       1 * time.Second,
			Timestamp: time.Now().Add(-2 * time.Second),
			Finished:  true,
		},
		1, 1, 0,
	},
	{
		"Expired but unfinished",
		State{
			TTL:       1 * time.Second,
			Timestamp: time.Now().Add(-2 * time.Second),
			Finished:  false,
		},
		1, 0, 1,
	},
}

func TestCleanup(t *testing.T) {
	for _, test := range cleanupTests {
		test := test
		t.Run(test.title, func(t *testing.T) {
			states := NewStates()
			states.SetStates([]State{test.state})

			assert.Equal(t, test.countBefore, states.Count())
			cleanupCount, _ := states.Cleanup()
			assert.Equal(t, test.cleanupCount, cleanupCount)
			assert.Equal(t, test.countAfter, states.Count())
		})
	}
}
