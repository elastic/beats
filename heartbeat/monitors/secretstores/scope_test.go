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

package secretstores

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitName(t *testing.T) {
	tests := []struct {
		content, name, rest string
		hasName             bool
	}{
		{content: "app/creds#token", rest: "app/creds#token"},
		{content: "prod@app/creds#token", name: "prod", rest: "app/creds#token", hasName: true},
		{content: "app/creds#user@x", rest: "app/creds#user@x"},
		{content: "app@creds#token", name: "app", rest: "creds#token", hasName: true},
		{content: "prod@a@b#c", name: "prod", rest: "a@b#c", hasName: true},
		{content: "@a#b", name: "", rest: "a#b", hasName: true},
	}
	for _, tc := range tests {
		name, rest, hasName := SplitName(tc.content)
		assert.Equal(t, tc.name, name, "name of %q", tc.content)
		assert.Equal(t, tc.rest, rest, "rest of %q", tc.content)
		assert.Equal(t, tc.hasName, hasName, "hasName of %q", tc.content)
	}
}

func TestScopes(t *testing.T) {
	entries := []entry{
		{storeType: "a", name: "x"},
		{storeType: "a"},
		{storeType: "b", name: "only"},
		{storeType: "c", name: "p"},
		{storeType: "c", name: "q"},
	}
	assert.Equal(t, []Scope{
		{Name: "x", Default: false, HasDefault: true, Names: []string{"x"}},
		{Name: "", Default: true, HasDefault: true, Names: []string{"x"}},
		{Name: "only", Default: true, HasDefault: true, Names: []string{"only"}},
		{Name: "p", Default: false, HasDefault: false, Names: []string{"p", "q"}},
		{Name: "q", Default: false, HasDefault: false, Names: []string{"p", "q"}},
	}, scopes(entries), "unexpected scopes")
}
