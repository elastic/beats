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

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestIncludeFields(t *testing.T) {

	var tests = []struct {
		Fields []string
		Input  common.MapStr
		Output common.MapStr
	}{
		{
			Fields: []string{"test"},
			Input: common.MapStr{
				"hello": "world",
				"test":  17,
			},
			Output: common.MapStr{
				"test": 17,
			},
		},
		{
			Fields: []string{"test", "a.b"},
			Input: common.MapStr{
				"a.b":  "b",
				"a.c":  "c",
				"test": 17,
			},
			Output: common.MapStr{
				"test": 17,
				"a": common.MapStr{
					"b": "b",
				},
			},
		},
	}

	for _, test := range tests {
		p := includeFields{
			Fields: test.Fields,
		}

		event := &beat.Event{
			Fields: test.Input,
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)

		assert.Equal(t, test.Output, newEvent.Fields)
	}
}
