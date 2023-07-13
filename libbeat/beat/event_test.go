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

func BenchmarkTestEventPutGetTimestamp(b *testing.B) {
	evt := newEmptyEvent()
	ts := time.Now()

	evt.PutValue("@timestamp", ts)

	v, err := evt.GetValue("@timestamp")
	if err != nil {
		b.Fatal(err)
	}

	assert.Equal(b, ts, v)
	assert.Equal(b, ts, evt.Timestamp)

	// The @timestamp is not written into Fields.
	assert.Nil(b, evt.Fields["@timestamp"])
}

func BenchmarkTestDeepUpdate(b *testing.B) {
	ts := time.Now()

	cases := []struct {
		name      string
		event     *Event
		update    mapstr.M
		overwrite bool
		expected  *Event
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
			overwrite: true,
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
			overwrite: false,
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
			overwrite: false,
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
			overwrite: true,
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
			overwrite: false,
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
			overwrite: true,
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
		b.Run(tc.name, func(b *testing.B) {
			tc.event.deepUpdate(tc.update, tc.overwrite)
			assert.Equal(b, tc.expected.Timestamp, tc.event.Timestamp)
			assert.Equal(b, tc.expected.Fields, tc.event.Fields)
			assert.Equal(b, tc.expected.Meta, tc.event.Meta)
		})
	}
}

func BenchmarkTestEventMetadata(b *testing.B) {
	const id = "123"
	newMeta := func() mapstr.M { return mapstr.M{"_id": id} }

	b.Run("put", func(b *testing.B) {
		evt := newEmptyEvent()
		meta := newMeta()

		evt.PutValue("@metadata", meta)

		assert.Equal(b, meta, evt.Meta)
		assert.Empty(b, evt.Fields)
	})

	b.Run("get", func(b *testing.B) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		meta, err := evt.GetValue("@metadata")

		assert.NoError(b, err)
		assert.Equal(b, evt.Meta, meta)
	})

	b.Run("put sub-key", func(b *testing.B) {
		evt := newEmptyEvent()

		evt.PutValue("@metadata._id", id)

		assert.Equal(b, newMeta(), evt.Meta)
		assert.Empty(b, evt.Fields)
	})

	b.Run("get sub-key", func(b *testing.B) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		v, err := evt.GetValue("@metadata._id")

		assert.NoError(b, err)
		assert.Equal(b, id, v)
	})

	b.Run("delete", func(b *testing.B) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		err := evt.Delete("@metadata")

		assert.NoError(b, err)
		assert.Nil(b, evt.Meta)
	})

	b.Run("delete sub-key", func(b *testing.B) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		err := evt.Delete("@metadata._id")

		assert.NoError(b, err)
		assert.Empty(b, evt.Meta)
	})

	b.Run("setID", func(b *testing.B) {
		evt := newEmptyEvent()

		evt.SetID(id)

		assert.Equal(b, newMeta(), evt.Meta)
	})

	b.Run("put non-metadata", func(b *testing.B) {
		evt := newEmptyEvent()

		evt.PutValue("@metadataSpecial", id)

		assert.Equal(b, mapstr.M{"@metadataSpecial": id}, evt.Fields)
	})

	b.Run("delete non-metadata", func(b *testing.B) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		err := evt.Delete("@metadataSpecial")

		assert.Error(b, err)
		assert.Equal(b, newMeta(), evt.Meta)
	})
}
