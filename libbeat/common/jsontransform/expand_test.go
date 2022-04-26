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

package jsontransform

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpand(t *testing.T) {
	type data struct {
		Event    mapstr.M
		Expected mapstr.M
		Err      string
	}
	tests := []data{
		{
			Event: mapstr.M{
				"hello.world": 15,
			},
			Expected: mapstr.M{
				"hello": mapstr.M{
					"world": 15,
				},
			},
		},
		{
			Event: mapstr.M{
				"test": 15,
			},
			Expected: mapstr.M{
				"test": 15,
			},
		},
		{
			Event: mapstr.M{
				"test":           15,
				"hello.there":    1,
				"hello.world.ok": "test",
				"elastic.for":    "search",
			},
			Expected: mapstr.M{
				"test": 15,
				"hello": mapstr.M{
					"there": 1,
					"world": mapstr.M{
						"ok": "test",
					},
				},
				"elastic": mapstr.M{
					"for": "search",
				},
			},
		},
		{
			Event: mapstr.M{
				"root": mapstr.M{
					"ok": 1,
				},
				"root.shared":        "yes",
				"root.one.two.three": 4,
			},
			Expected: mapstr.M{
				"root": mapstr.M{
					"ok":     1,
					"shared": "yes",
					"one":    mapstr.M{"two": mapstr.M{"three": 4}},
				},
			},
		},
		{
			Event: mapstr.M{
				"root": mapstr.M{
					"seven": 1,
				},
				"root.seven.eight": 2,
			},
			Err: `cannot expand .*`,
		},
		{
			Event: mapstr.M{
				"a.b": 1,
				"a": mapstr.M{
					"b": 2,
				},
			},
			Err: `cannot expand .*`,
		},
		{
			Event: mapstr.M{
				"a.b": mapstr.M{
					"c": mapstr.M{
						"d": 1,
					},
				},
				"a.b.c": mapstr.M{
					"d": 2,
				},
			},
			Err: `cannot expand .*`,
		},
	}

	for _, test := range tests {
		err := expandFields(test.Event)
		if test.Err != "" {
			require.Error(t, err)
			assert.Regexp(t, test.Err, err.Error())
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.Expected, test.Event)
	}
}
