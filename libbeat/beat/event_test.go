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
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func newEmptyEvent() *Event {
	return &Event{Fields: mapstr.M{}}
}

func TestEventPutGetTimestamp(t *testing.T) {
	evt := newEmptyEvent()
	ts := time.Now()

	evt.PutValue("@timestamp", ts)

	v, err := evt.GetValue("@timestamp")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ts, v)
	assert.Equal(t, ts, evt.Timestamp)

	// The @timestamp is not written into Fields.
	assert.Nil(t, evt.Fields["@timestamp"])
}

func TestDeepUpdate(t *testing.T) {
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
			event:    &Event{},
			update:   mapstr.M{},
			expected: &Event{},
		},
		{
			name:  "updates timestamp",
			event: &Event{},
			update: mapstr.M{
				timestampFieldKey: ts,
			},
			overwrite: true,
			expected: &Event{
				Timestamp: ts,
			},
		},
		{
			name: "does not overwrite timestamp",
			event: &Event{
				Timestamp: ts,
			},
			update: mapstr.M{
				timestampFieldKey: time.Now().Add(time.Hour),
			},
			overwrite: false,
			expected: &Event{
				Timestamp: ts,
			},
		},
		{
			name:  "initializes metadata if nil",
			event: &Event{},
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
			},
		},
		{
			name: "updates metadata but does not overwrite",
			event: &Event{
				Meta: mapstr.M{
					"first": "initial",
				},
			},
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
			},
		},
		{
			name: "updates metadata and overwrites",
			event: &Event{
				Meta: mapstr.M{
					"first": "initial",
				},
			},
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
			},
		},
		{
			name: "updates fields but does not overwrite",
			event: &Event{
				Fields: mapstr.M{
					"first": "initial",
				},
			},
			update: mapstr.M{
				"first":  "new",
				"second": 42,
			},
			overwrite: false,
			expected: &Event{
				Fields: mapstr.M{
					"first":  "initial",
					"second": 42,
				},
			},
		},
		{
			name: "updates metadata and overwrites",
			event: &Event{
				Fields: mapstr.M{
					"first": "initial",
				},
			},
			update: mapstr.M{
				"first":  "new",
				"second": 42,
			},
			overwrite: true,
			expected: &Event{
				Fields: mapstr.M{
					"first":  "new",
					"second": 42,
				},
			},
		},
		{
			name:  "initializes fields if nil",
			event: &Event{},
			update: mapstr.M{
				"first":  "new",
				"second": 42,
			},
			expected: &Event{
				Fields: mapstr.M{
					"first":  "new",
					"second": 42,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.event.deepUpdate(tc.update, tc.overwrite)
			assert.Equal(t, tc.expected.Timestamp, tc.event.Timestamp)
			assert.Equal(t, tc.expected.Fields, tc.event.Fields)
			assert.Equal(t, tc.expected.Meta, tc.event.Meta)
		})
	}
}

func TestEventMetadata(t *testing.T) {
	const id = "123"
	newMeta := func() mapstr.M { return mapstr.M{"_id": id} }

	t.Run("put", func(t *testing.T) {
		evt := newEmptyEvent()
		meta := newMeta()

		evt.PutValue("@metadata", meta)

		assert.Equal(t, meta, evt.Meta)
		assert.Empty(t, evt.Fields)
	})

	t.Run("get", func(t *testing.T) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		meta, err := evt.GetValue("@metadata")

		assert.NoError(t, err)
		assert.Equal(t, evt.Meta, meta)
	})

	t.Run("put sub-key", func(t *testing.T) {
		evt := newEmptyEvent()

		evt.PutValue("@metadata._id", id)

		assert.Equal(t, newMeta(), evt.Meta)
		assert.Empty(t, evt.Fields)
	})

	t.Run("get sub-key", func(t *testing.T) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		v, err := evt.GetValue("@metadata._id")

		assert.NoError(t, err)
		assert.Equal(t, id, v)
	})

	t.Run("delete", func(t *testing.T) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		err := evt.Delete("@metadata")

		assert.NoError(t, err)
		assert.Nil(t, evt.Meta)
	})

	t.Run("delete sub-key", func(t *testing.T) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		err := evt.Delete("@metadata._id")

		assert.NoError(t, err)
		assert.Empty(t, evt.Meta)
	})

	t.Run("setID", func(t *testing.T) {
		evt := newEmptyEvent()

		evt.SetID(id)

		assert.Equal(t, newMeta(), evt.Meta)
	})

	t.Run("put non-metadata", func(t *testing.T) {
		evt := newEmptyEvent()

		evt.PutValue("@metadataSpecial", id)

		assert.Equal(t, mapstr.M{"@metadataSpecial": id}, evt.Fields)
	})

	t.Run("delete non-metadata", func(t *testing.T) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		err := evt.Delete("@metadataSpecial")

		assert.Error(t, err)
		assert.Equal(t, newMeta(), evt.Meta)
	})
}
