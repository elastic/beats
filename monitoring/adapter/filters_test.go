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

package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilters(t *testing.T) {
	tests := []struct {
		start    state
		filters  *metricFilters
		expected state
	}{
		{
			state{action: actIgnore, name: "test"},
			nil,
			state{action: actIgnore, name: "test"},
		},
		{
			state{action: actIgnore, name: "test"},
			makeFilters(),
			state{action: actIgnore, name: "test"},
		},
		{
			state{action: actIgnore, name: "test"},
			makeFilters(
				WhitelistIf(func(_ string) bool { return true }),
			),
			state{action: actAccept, name: "test"},
		},
		{
			state{action: actIgnore, name: "test"},
			makeFilters(
				WhitelistIf(func(_ string) bool { return false }),
			),
			state{action: actIgnore, name: "test"},
		},
		{
			state{action: actIgnore, name: "test"},
			makeFilters(Whitelist("other")),
			state{action: actIgnore, name: "test"},
		},
		{
			state{action: actIgnore, name: "test"},
			makeFilters(Whitelist("test")),
			state{action: actAccept, name: "test"},
		},
		{
			state{action: actIgnore, name: "test"},
			makeFilters(Rename("test", "new")),
			state{action: actAccept, name: "new"},
		},
		{
			state{action: actIgnore, name: "t-e-s-t"},
			makeFilters(NameReplace("-", ".")),
			state{action: actIgnore, name: "t.e.s.t"},
		},
		{
			state{action: actIgnore, name: "test"},
			makeFilters(ToUpperName),
			state{action: actIgnore, name: "TEST"},
		},
		{
			state{action: actIgnore, name: "TEST"},
			makeFilters(ToLowerName),
			state{action: actIgnore, name: "test"},
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v => %v", i, test.start, test.expected)

		actual := test.filters.apply(test.start)
		assert.Equal(t, test.expected, actual)
	}
}
