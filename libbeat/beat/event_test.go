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

package beat

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	propSize = 1024 * 2014 * 10
)

var largeProp string

func init() {
	b := make([]byte, propSize)
	_, _ = rand.Read(b)
	largeProp = string(b)
}

func newEmptyEvent() *Event {
	return &Event{Fields: mapstr.M{}}
}

func newEvent(e mapstr.M) *Event {
	n := &mapstr.M{
		"Fields": mapstr.M{
			"large_prop": largeProp,
		},
	}
	n.DeepUpdate(e)
	var ts time.Time
	var meta mapstr.M
	var fields mapstr.M
	var private mapstr.M

	v, ex := (*n)["Timestamp"]
	if ex {
		ts = v.(time.Time)
	}
	v, ex = (*n)["Meta"]
	if ex {
		meta = v.(mapstr.M)
	}
	v, ex = (*n)["Fields"]
	if ex {
		fields = v.(mapstr.M)
	}
	v, ex = (*n)["Private"]
	if ex {
		private = v.(mapstr.M)
	}
	return &Event{
		Timestamp: ts,
		Meta:      meta,
		Fields:    fields,
		Private:   private,
	}
}

func TestEventPutGetTimestamp(t *testing.T) {
	evt := newEmptyEvent()
	ts := time.Now()

	prev, err := evt.PutValue("@timestamp", ts)
	require.NoError(t, err)
	require.Equal(t, time.Time{}, prev, "previous timestamp should be empty")

	v, err := evt.GetValue("@timestamp")
	require.NoError(t, err)
	require.Equal(t, ts, v)
	require.Equal(t, ts, evt.Timestamp)

	// The @timestamp is not written into Fields.
	require.Nil(t, evt.Fields["@timestamp"])
}

func TestDeepUpdate(t *testing.T) {
	ts := time.Now()

	cases := []struct {
		name     string
		event    *Event
		update   mapstr.M
		mode     updateMode
		expected *Event
	}{
		{
			name:     "does nothing if no update",
			event:    newEvent(mapstr.M{}),
			update:   mapstr.M{},
			expected: newEvent(mapstr.M{}),
		},
		{
			name:  "updates timestamp",
			event: newEvent(mapstr.M{}),
			update: mapstr.M{
				timestampFieldKey: ts,
			},
			mode: updateModeOverwrite,
			expected: &Event{
				Timestamp: ts,
				Fields: mapstr.M{
					"large_prop": largeProp,
				},
			},
		},
		{
			name: "does not overwrite timestamp",
			event: newEvent(mapstr.M{
				"Timestamp": ts,
			}),
			update: mapstr.M{
				timestampFieldKey: time.Now().Add(time.Hour),
			},
			mode: updateModeNoOverwrite,
			expected: &Event{
				Timestamp: ts,
				Fields: mapstr.M{
					"large_prop": largeProp,
				},
			},
		},
		{
			name:  "initializes metadata if nil",
			event: newEvent(mapstr.M{}),
			update: mapstr.M{
				metadataFieldKey: mapstr.M{
					"first":  "new",
					"second": 42,
				},
			},
			expected: &Event{
				Meta: mapstr.M{
					"first":  "new",
					"second": 42,
				},
				Fields: mapstr.M{
					"large_prop": largeProp,
				},
			},
		},
		{
			name: "updates metadata but does not overwrite",
			event: newEvent(mapstr.M{
				"Meta": mapstr.M{
					"first": "initial",
				},
			}),
			update: mapstr.M{
				metadataFieldKey: mapstr.M{
					"first":  "new",
					"second": 42,
				},
			},
			mode: updateModeNoOverwrite,
			expected: &Event{
				Meta: mapstr.M{
					"first":  "initial",
					"second": 42,
				},
				Fields: mapstr.M{
					"large_prop": largeProp,
				},
			},
		},
		{
			name: "updates metadata and overwrites",
			event: newEvent(mapstr.M{
				"Meta": mapstr.M{
					"first": "initial",
				},
			}),
			update: mapstr.M{
				metadataFieldKey: mapstr.M{
					"first":  "new",
					"second": 42,
				},
			},
			mode: updateModeOverwrite,
			expected: &Event{
				Meta: mapstr.M{
					"first":  "new",
					"second": 42,
				},
				Fields: mapstr.M{
					"large_prop": largeProp,
				},
			},
		},
		{
			name: "updates fields but does not overwrite",
			event: newEvent(mapstr.M{
				"Fields": mapstr.M{
					"first": "initial",
				},
			}),
			update: mapstr.M{
				"first":  "new",
				"second": 42,
			},
			mode: updateModeNoOverwrite,
			expected: &Event{
				Fields: mapstr.M{
					"first":      "initial",
					"second":     42,
					"large_prop": largeProp,
				},
			},
		},
		{
			name: "updates metadata and overwrites",
			event: newEvent(mapstr.M{
				"Fields": mapstr.M{
					"first": "initial",
				},
			}),
			update: mapstr.M{
				"first":  "new",
				"second": 42,
			},
			mode: updateModeOverwrite,
			expected: &Event{
				Fields: mapstr.M{
					"first":      "new",
					"second":     42,
					"large_prop": largeProp,
				},
			},
		},
		{
			name:  "initializes fields if nil",
			event: newEvent(mapstr.M{}),
			update: mapstr.M{
				"first":  "new",
				"second": 42,
			},
			expected: &Event{
				Fields: mapstr.M{
					"first":      "new",
					"second":     42,
					"large_prop": largeProp,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.event.deepUpdate(tc.update, tc.mode)
			assert.Equal(t, tc.expected.Timestamp, tc.event.Timestamp)
			assert.Equal(t, tc.expected.Fields, tc.event.Fields)
			assert.Equal(t, tc.expected.Meta, tc.event.Meta)
		})
	}
}

func TestEventFieldsAndMetadata(t *testing.T) {
	// TODO re-write all these tests using a case list
}
