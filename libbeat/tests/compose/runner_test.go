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

package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunnerScenarios(t *testing.T) {
	cases := []struct {
		Title    string
		Options  RunnerOptions
		Expected []map[string]string
	}{
		{
			Title:    "Nil options",
			Options:  nil,
			Expected: []map[string]string{nil},
		},
		{
			Title:    "Empty options",
			Options:  RunnerOptions{},
			Expected: []map[string]string{nil},
		},
		{
			Title: "One option, two values",
			Options: RunnerOptions{
				"FOO": {"bar", "baz"},
			},
			Expected: []map[string]string{
				{"FOO": "bar"},
				{"FOO": "baz"},
			},
		},
		{
			Title: "Multiple options",
			Options: RunnerOptions{
				"FOO": {"bar", "baz"},
				"BAZ": {"stuff"},
			},
			Expected: []map[string]string{
				{"FOO": "bar", "BAZ": "stuff"},
				{"FOO": "baz", "BAZ": "stuff"},
			},
		},
		{
			Title: "Multiple options, single values",
			Options: RunnerOptions{
				"FOO": {"bar"},
				"BAZ": {"stuff"},
			},
			Expected: []map[string]string{
				{"FOO": "bar", "BAZ": "stuff"},
			},
		},
	}

	for _, c := range cases {
		r := TestRunner{Options: c.Options}
		found := r.scenarios()
		assert.Equal(t, c.Expected, found, c.Title)
	}
}
