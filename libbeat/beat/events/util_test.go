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

package events

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestGetMetaStringValue(t *testing.T) {
	tests := map[string]struct {
		event         beat.Event
		metaFieldPath string
		expectedValue string
		expectedErr   error
	}{
		"nonexistent_field": {
			beat.Event{
				Meta: mapstr.M{
					"foo": "bar",
				},
			},
			"nonexistent",
			"",
			mapstr.ErrKeyNotFound,
		},
		"root": {
			beat.Event{
				Meta: mapstr.M{
					"foo": "bar",
					"baz": "hello",
				},
			},
			"baz",
			"hello",
			nil,
		},
		"nested": {
			beat.Event{
				Meta: mapstr.M{
					"foo": "bar",
					"baz": mapstr.M{
						"qux": "hello",
					},
				},
			},
			"baz.qux",
			"hello",
			nil,
		},
		"non_string": {
			beat.Event{
				Meta: mapstr.M{
					"foo": "bar",
					"baz": 17,
				},
			},
			"baz",
			"",
			nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			value, err := GetMetaStringValue(test.event, test.metaFieldPath)
			require.Equal(t, test.expectedValue, value)
			require.Equal(t, test.expectedErr, err)
		})
	}
}
