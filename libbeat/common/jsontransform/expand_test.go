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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestExpand(t *testing.T) {
	type data struct {
		Event    common.MapStr
		Expected common.MapStr
		Err      string
	}
	tests := []data{
		{
			Event: common.MapStr{
				"hello.world": 15,
			},
			Expected: common.MapStr{
				"hello": common.MapStr{
					"world": 15,
				},
			},
		},
		{
			Event: common.MapStr{
				"test": 15,
			},
			Expected: common.MapStr{
				"test": 15,
			},
		},
		{
			Event: common.MapStr{
				"test":           15,
				"hello.there":    1,
				"hello.world.ok": "test",
				"elastic.for":    "search",
			},
			Expected: common.MapStr{
				"test": 15,
				"hello": common.MapStr{
					"there": 1,
					"world": common.MapStr{
						"ok": "test",
					},
				},
				"elastic": common.MapStr{
					"for": "search",
				},
			},
		},
		{
			Event: common.MapStr{
				"root": common.MapStr{
					"ok": 1,
				},
				"root.shared":        "yes",
				"root.one.two.three": 4,
			},
			Expected: common.MapStr{
				"root": common.MapStr{
					"ok":     1,
					"shared": "yes",
					"one":    common.MapStr{"two": common.MapStr{"three": 4}},
				},
			},
		},
		{
			Event: common.MapStr{
				"root": common.MapStr{
					"seven": 1,
				},
				"root.seven.eight": 2,
			},
			Err: `cannot expand .*`,
		},
		{
			Event: common.MapStr{
				"a.b": 1,
				"a": common.MapStr{
					"b": 2,
				},
			},
			Err: `cannot expand .*`,
		},
		{
			Event: common.MapStr{
				"a.b": common.MapStr{
					"c": common.MapStr{
						"d": 1,
					},
				},
				"a.b.c": common.MapStr{
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
