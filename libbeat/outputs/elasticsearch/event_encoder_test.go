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

package elasticsearch

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testIndexSelector struct{}

func (testIndexSelector) Select(event *beat.Event) (string, error) {
	return "test", nil
}

func TestEncodeEntry(t *testing.T) {
	indexSelector := testIndexSelector{}

	encoder := newEventEncoder(true, indexSelector, nil)

	timestamp := time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	pubEvent := publisher.Event{
		Content: beat.Event{
			Timestamp: timestamp,
			Fields: mapstr.M{
				"test_field":   "test_value",
				"number_field": 5,
				"nested": mapstr.M{
					"nested_field": "nested_value",
				},
			},
			Meta: mapstr.M{
				events.FieldMetaOpType:   "create",
				events.FieldMetaPipeline: "TEST_PIPELINE",
				events.FieldMetaID:       "test_id",
			},
		},
	}

	encoded, encodedSize := encoder.EncodeEntry(pubEvent)
	encPubEvent, ok := encoded.(publisher.Event)

	// Check the resulting publisher.Event
	require.True(t, ok, "EncodeEntry must return a publisher.Event")
	require.NotNil(t, encPubEvent.EncodedEvent, "EncodeEntry must set EncodedEvent")
	assert.Nil(t, encPubEvent.Content.Fields, "EncodeEntry should clear event.Content")

	// Check the inner encodedEvent
	encBeatEvent, ok := encPubEvent.EncodedEvent.(*encodedEvent)
	require.True(t, ok, "EncodeEntry should set EncodedEvent to a *encodedEvent")
	require.Equal(t, len(encBeatEvent.encoding), encodedSize, "Reported size should match encoded buffer")

	// Check event metadata
	assert.Equal(t, "test_id", encBeatEvent.id, "Event id should match original metadata")
	assert.Equal(t, "test", encBeatEvent.index, "Event should have the index set by its selector")
	assert.Equal(t, "test_pipeline", encBeatEvent.pipeline, "Event pipeline should match original metadata")
	assert.Equal(t, timestamp, encBeatEvent.timestamp, "encodedEvent.timestamp should match the original event")
	assert.Equal(t, events.OpTypeCreate, encBeatEvent.opType, "encoded opType should match the original metadata")
	assert.False(t, encBeatEvent.deadLetter, "encoded event shouldn't have deadLetter flag set")

	// Check encoded fields
	var eventContent struct {
		Timestamp   time.Time `json:"@timestamp"`
		TestField   string    `json:"test_field"`
		NumberField int       `json:"number_field"`
		Nested      struct {
			NestedField string `json:"nested_field"`
		} `json:"nested"`
	}
	err := json.Unmarshal(encBeatEvent.encoding, &eventContent)
	require.NoError(t, err, "encoding should contain valid json")
	assert.Equal(t, timestamp, eventContent.Timestamp, "Encoded timestamp should match original")
	assert.Equal(t, "test_value", eventContent.TestField, "Encoded field should match original")
	assert.Equal(t, 5, eventContent.NumberField, "Encoded field should match original")
	assert.Equal(t, "nested_value", eventContent.Nested.NestedField, "Encoded field should match original")
}

// encodeBatch encodes a publisher.Batch so it can be provided to
// Client.Publish and other helpers.
// This modifies the batch in place, but also returns its input batch
// to allow for easy chaining while creating test batches.
func encodeBatch[B publisher.Batch](client *Client, batch B) B {
	encodeEvents(client, batch.Events())
	return batch
}

// A test helper to encode an event array for an Elasticsearch client.
// This isn't particularly efficient since it creates a new encoder object
// for every set of events, but it's much easier and the difference is
// negligible for any non-benchmark tests.
// This modifies the slice in place, but also returns its input slice
// to allow for easy chaining while creating test events.
func encodeEvents(client *Client, events []publisher.Event) []publisher.Event {
	encoder := newEventEncoder(
		client.conn.EscapeHTML,
		client.indexSelector,
		client.pipelineSelector,
	)
	for i := range events {
		// Skip encoding if there's already encoded data present
		if events[i].EncodedEvent == nil {
			encoded, _ := encoder.EncodeEntry(events[i])
			event := encoded.(publisher.Event)
			events[i] = event
		}
	}
	return events
}

func encodeEvent(client *Client, event publisher.Event) publisher.Event {
	encoder := newEventEncoder(
		client.conn.EscapeHTML,
		client.indexSelector,
		client.pipelineSelector,
	)
	encoded, _ := encoder.EncodeEntry(event)
	return encoded.(publisher.Event)
}
