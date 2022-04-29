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

package reader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestToEvent(t *testing.T) {
	testCases := map[string]struct {
		msg      Message
		expected beat.Event
	}{
		"empty message; emtpy event": {
			Message{},
			beat.Event{},
		},
		"empty content, one field": {
			Message{Fields: mapstr.M{"my_field": "my_value"}},
			beat.Event{Fields: mapstr.M{"my_field": "my_value"}},
		},
		"content, no field": {
			Message{Content: []byte("my message")},
			beat.Event{Fields: mapstr.M{"message": "my message"}},
		},
		"content, one field": {
			Message{Content: []byte("my message"), Fields: mapstr.M{"my_field": "my_value"}},
			beat.Event{Fields: mapstr.M{"message": "my message", "my_field": "my_value"}},
		},
		"content, message field": {
			Message{Content: []byte("my message"), Fields: mapstr.M{"message": "my_message_value"}},
			beat.Event{Fields: mapstr.M{"message": "my message"}},
		},
		"content, meta, message field": {
			Message{Content: []byte("my message"), Fields: mapstr.M{"my_field": "my_value"}, Meta: mapstr.M{"meta": "id"}},
			beat.Event{Fields: mapstr.M{"message": "my message", "my_field": "my_value"}, Meta: mapstr.M{"meta": "id"}},
		},
		"content, meta, message and private fields": {
			Message{
				Ts:      time.Date(2022, 1, 9, 10, 42, 0, 0, time.UTC),
				Content: []byte("my message"),
				Meta:    mapstr.M{"foo": "bar"},
				Private: 42,
			},
			beat.Event{
				Timestamp: time.Date(2022, 1, 9, 10, 42, 0, 0, time.UTC),
				Fields:    mapstr.M{"message": "my message"},
				Meta:      mapstr.M{"foo": "bar"},
				Private:   42,
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expected, test.msg.ToEvent())
		})
	}

}
