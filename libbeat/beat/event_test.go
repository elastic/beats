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

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func newEmptyEvent() *Event {
	return &Event{Fields: common.MapStr{}}
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

func TestEventMetadata(t *testing.T) {
	const id = "123"
	newMeta := func() common.MapStr { return common.MapStr{"id": id} }

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

		evt.PutValue("@metadata.id", id)

		assert.Equal(t, newMeta(), evt.Meta)
		assert.Empty(t, evt.Fields)
	})

	t.Run("get sub-key", func(t *testing.T) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		v, err := evt.GetValue("@metadata.id")

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

		err := evt.Delete("@metadata.id")

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

		assert.Equal(t, common.MapStr{"@metadataSpecial": id}, evt.Fields)
	})

	t.Run("delete non-metadata", func(t *testing.T) {
		evt := newEmptyEvent()
		evt.Meta = newMeta()

		err := evt.Delete("@metadataSpecial")

		assert.Error(t, err)
		assert.Equal(t, newMeta(), evt.Meta)
	})
}

func TestEvent_AddECSErrorMessage(t *testing.T) {

	t.Run("single error", func(t *testing.T) {
		evt := newEmptyEvent()
		msg := "something went wrong"
		evt.AddECSErrorMessage(msg)
		iface, err := evt.GetValue("error.message")
		assert.NoError(t, err)
		assert.Equal(t, msg, iface)
	})
	t.Run("multiple errors", func(t *testing.T) {
		evt := newEmptyEvent()
		msgs := []string{
			"something went wrong",
			"also this",
			"and that",
		}
		for _, msg := range msgs {
			evt.AddECSErrorMessage(msg)
		}
		iface, err := evt.GetValue("error.message")
		assert.NoError(t, err)
		assert.Equal(t, msgs, iface)
	})
	t.Run("multiple errors (alternative)", func(t *testing.T) {
		evt := newEmptyEvent()
		msgs := []string{
			"something went wrong",
			"also this",
			"and that",
		}
		evt.PutValue("error.message", msgs[0])
		for _, msg := range msgs[1:] {
			evt.AddECSErrorMessage(msg)
		}
		iface, err := evt.GetValue("error.message")
		assert.NoError(t, err)
		assert.Equal(t, msgs, iface)
	})
	t.Run("multiple errors (empty list)", func(t *testing.T) {
		evt := newEmptyEvent()
		msgs := []string{
			"something went wrong",
		}
		evt.PutValue("error.message", []string{})
		for _, msg := range msgs {
			evt.AddECSErrorMessage(msg)
		}
		iface, err := evt.GetValue("error.message")
		assert.NoError(t, err)
		assert.Equal(t, msgs, iface)
	})
	t.Run("bad type", func(t *testing.T) {
		evt := newEmptyEvent()
		msgs := []string{
			"42",
			"Previous error message has unexpected type int",
			"something went wrong",
		}
		evt.PutValue("error.message", 42)
		evt.AddECSErrorMessage(msgs[2])
		iface, err := evt.GetValue("error.message")
		assert.NoError(t, err)
		assert.Equal(t, msgs, iface)
	})
	t.Run("bad type (list)", func(t *testing.T) {
		evt := newEmptyEvent()
		msgs := []string{
			"[42 33]",
			"Previous error message has unexpected type []int",
			"something went wrong",
		}
		evt.PutValue("error.message", []int{42, 33})
		evt.AddECSErrorMessage(msgs[2])
		iface, err := evt.GetValue("error.message")
		assert.NoError(t, err)
		assert.Equal(t, msgs, iface)
	})
	t.Run("existing nil error", func(t *testing.T) {
		evt := newEmptyEvent()
		evt.PutValue("error.message", nil)
		evt.AddECSErrorMessage("custom error")
		iface, err := evt.GetValue("error.message")
		assert.NoError(t, err)
		assert.Equal(t, "custom error", iface)
	})
}
