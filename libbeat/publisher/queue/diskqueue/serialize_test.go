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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

// A test to make sure serialization works correctly on multi-byte characters.
func TestSerializeMultiByte(t *testing.T) {
	asciiOnly := "{\"name\": \"Momotaro\"}"
	multiBytes := "{\"name\": \"桃太郎\"}"

	encoder := newEventEncoder()
	event := publisher.Event{
		Content: beat.Event{
			Fields: common.MapStr{
				"ascii_only":  asciiOnly,
				"multi_bytes": multiBytes,
			},
		},
	}
	serialized, err := encoder.encode(&event)
	if err != nil {
		t.Fatalf("Couldn't encode event: %v", err)
	}

	// Use decoder to decode the serialized bytes.
	decoder := newEventDecoder()
	buf := decoder.Buffer(len(serialized))
	copy(buf, serialized)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Couldn't decode serialized data: %v", err)
	}

	decodedAsciiOnly, _ := decoded.Content.Fields.GetValue("ascii_only")
	assert.Equal(t, asciiOnly, decodedAsciiOnly)

	decodedMultiBytes, _ := decoded.Content.Fields.GetValue("multi_bytes")
	assert.Equal(t, multiBytes, decodedMultiBytes)
}
