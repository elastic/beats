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

package beater

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type mockStatefulLogger struct {
	states []file.State
}

func (sf *mockStatefulLogger) Published(states []file.State) {
	sf.states = states
}

type mockStatelessLogger struct {
	count int
}

func (sl *mockStatelessLogger) Published(count int) bool {
	sl.count = count
	return true
}

func TestACKer(t *testing.T) {
	tests := []struct {
		name      string
		data      []interface{}
		stateless int
		stateful  []file.State
	}{
		{
			name:      "only stateless",
			data:      []interface{}{nil, nil},
			stateless: 2,
		},
		{
			name:      "only stateful",
			data:      []interface{}{file.State{Source: "-"}, file.State{Source: "-"}},
			stateful:  []file.State{{Source: "-"}, {Source: "-"}},
			stateless: 0,
		},
		{
			name:      "both",
			data:      []interface{}{file.State{Source: "-"}, nil, file.State{Source: "-"}},
			stateful:  []file.State{{Source: "-"}, {Source: "-"}},
			stateless: 1,
		},
		{
			name:      "any other Private type",
			data:      []interface{}{struct{}{}, nil},
			stateless: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sl := &mockStatelessLogger{}
			sf := &mockStatefulLogger{}

			h := eventACKer(sl, sf)

			for _, datum := range test.data {
				h.AddEvent(beat.Event{Private: datum}, true)
			}

			h.ACKEvents(len(test.data))
			assert.Equal(t, test.stateless, sl.count)
			assert.Equal(t, test.stateful, sf.states)
		})
	}
}
