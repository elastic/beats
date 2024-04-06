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
	"testing"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

func TestEncodeEntry(t *testing.T) {
	/*cfg := c.MustNewConfigFrom(mapstr.M{})
	info := beat.Info{
		IndexPrefix: "test",
		Version:     version.GetDefaultVersion(),
	}

	index, pipeline, err := buildSelectors(im, info, cfg)
	require.NoError(t, err)

	encoder := newEventEncoder(true, testIndexSelector{})*/
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
