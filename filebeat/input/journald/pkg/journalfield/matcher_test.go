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

//go:build linux && cgo

package journalfield

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		name  string
		im    IncludeMatches
		error bool
	}{
		{
			name: "OR condition exists",
			im: IncludeMatches{
				OR: []IncludeMatches{
					{},
				},
			},
			error: true,
		},
		{
			name: "AND condition exists",
			im: IncludeMatches{
				AND: []IncludeMatches{
					{},
				},
			},
			error: true,
		},
		{
			name: "empty include matches succeeds validation",
			im:   IncludeMatches{},
		},
		{
			name: "matches are allowed",
			im: IncludeMatches{
				Matches: []Matcher{
					{"foo"},
					{"bar"},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.im.Validate()
			if tc.error && err == nil {
				t.Fatal("expecting Validate to fail")
			}

			if !tc.error && err != nil {
				t.Fatalf("expecting Validate to succeed, but got error: %s", err)
			}
		})
	}
}
