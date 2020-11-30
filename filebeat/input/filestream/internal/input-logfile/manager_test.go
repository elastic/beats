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

package input_logfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testPluginName = "my_test_plugin"

type testSource struct {
	name string
}

func (s *testSource) Name() string {
	return s.name
}

func TestSourceIdentifier_ID(t *testing.T) {
	testCases := map[string]struct {
		userID            string
		sources           []*testSource
		expectedSourceIDs []string
	}{
		"plugin with no user configured ID": {
			sources: []*testSource{
				&testSource{"unique_name"},
				&testSource{"another_unique_name"},
			},
			expectedSourceIDs: []string{
				testPluginName + "::unique_name",
				testPluginName + "::another_unique_name",
			},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			srcIdentifier := newSourceIdentifier(testPluginName, test.userID)

			for i, src := range test.sources {
				srcID := srcIdentifier.ID(src)
				assert.Equal(t, test.expectedSourceIDs[i], srcID)
			}
		})
	}
}

func TestSourceIdentifier_MachesInput(t *testing.T) {
	testCases := map[string]struct {
		userID      string
		matchingIDs []string
	}{
		"plugin with no user configured ID": {
			matchingIDs: []string{
				testPluginName + "::my_id",
				testPluginName + "::path::my_id",
				testPluginName + "::" + testPluginName + "::my_id",
			},
		},
		"plugin with user configured ID": {
			userID: "my-id",
			matchingIDs: []string{
				testPluginName + "::my-id::my_id",
				testPluginName + "::my-id::path::my_id",
				testPluginName + "::my-id::" + testPluginName + "::my_id",
			},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			srcIdentifier := newSourceIdentifier(testPluginName, test.userID)

			for _, id := range test.matchingIDs {
				assert.True(t, srcIdentifier.MatchesInput(id))
			}
		})
	}
}

func TestSourceIdentifier_NotMachesInput(t *testing.T) {

	testCases := map[string]struct {
		userID         string
		notMatchingIDs []string
	}{
		"plugin with no user configured ID": {
			notMatchingIDs: []string{
				"::my_id",
				"::path::my_id::" + testPluginName,
			},
		},
		"plugin with user configured ID": {
			userID: "my-id",
			notMatchingIDs: []string{
				testPluginName + "-other-id::my_id",
				"my-id::path::my_id",
			},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			srcIdentifier := newSourceIdentifier(testPluginName, test.userID)

			for _, id := range test.notMatchingIDs {
				assert.False(t, srcIdentifier.MatchesInput(id))
			}
		})
	}
}
