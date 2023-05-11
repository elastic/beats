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

package diskqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

// A test to make sure serialization works correctly on multi-byte characters.
func TestSerialize(t *testing.T) {
	tests := map[string]struct {
		value  string
		format SerializationFormat
	}{
		"Ascii only, CBOR": {
			value:  "{\"name\": \"Momotaro\"}",
			format: SerializationCBOR,
		},
		"Multi-byte, CBOR": {
			value:  "{\"name\": \"桃太郎\"}",
			format: SerializationCBOR,
		},
		"Ascii only, Protobuf": {
			value:  "{\"name\": \"Momotaro\"}",
			format: SerializationProtobuf,
		},
		"Multi-byte, Protobuf": {
			value:  "{\"name\": \"桃太郎\"}",
			format: SerializationProtobuf,
		},
	}

	for name, tc := range tests {
		encoder := newEventEncoder(tc.format)
		var event interface{}
		switch tc.format {
		case SerializationCBOR:
			event = publisher.Event{
				Content: beat.Event{
					Fields: mapstr.M{
						"test_field": tc.value,
					},
				},
			}
		case SerializationProtobuf:
			event = &messages.Event{
				Fields: &messages.Struct{
					Data: map[string]*messages.Value{
						"test_field": {
							Kind: &messages.Value_StringValue{
								StringValue: tc.value,
							},
						},
					},
				},
			}
		}
		serialized, err := encoder.encode(event)
		assert.NoErrorf(t, err, "%s: Couldn't encode event, error: %v", name, err)

		// Use decoder to decode the serialized bytes.
		decoder := newEventDecoder()
		decoder.serializationFormat = tc.format
		buf := decoder.Buffer(len(serialized))
		copy(buf, serialized)
		decoded, err := decoder.Decode()
		require.NoErrorf(t, err, "%s: Couldn't decode event", name)

		switch tc.format {
		case SerializationCBOR:
			event, ok := decoded.(publisher.Event)
			require.True(t, ok)
			decodedValue, err := event.Content.Fields.GetValue("test_field")
			assert.NoErrorf(t, err, "%s: Couldn't get 'test_field'", name)
			assert.Equal(t, tc.value, decodedValue)
		case SerializationProtobuf:
			event, ok := decoded.(*messages.Event)
			require.True(t, ok)
			d := event.GetFields().GetData()
			test_field, prs := d["test_field"]
			assert.Truef(t, prs, "'test_field' was not present in decoded event data")
			decodedValue := test_field.GetStringValue()
			assert.Equal(t, tc.value, decodedValue)
		}
	}
}
