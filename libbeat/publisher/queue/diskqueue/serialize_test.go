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

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/publisher"
)

// A test to make sure serialization works correctly on multi-byte characters.
func TestSerialize(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{name: "Ascii only", value: "{\"name\": \"Momotaro\"}"},
		{name: "Multi-byte", value: "{\"name\": \"桃太郎\"}"},
	}

	for _, test := range testCases {
		encoder := newEventEncoder()
		event := publisher.Event{
			Content: beat.Event{
				Fields: common.MapStr{
					"test_field": test.value,
				},
			},
		}
		serialized, err := encoder.encode(&event)
		if err != nil {
			t.Fatalf("[%v] Couldn't encode event: %v", test.name, err)
		}

		// Use decoder to decode the serialized bytes.
		decoder := newEventDecoder()
		buf := decoder.Buffer(len(serialized))
		copy(buf, serialized)
		decoded, err := decoder.Decode()
		if err != nil {
			t.Fatalf("[%v] Couldn't decode serialized data: %v", test.name, err)
		}

		decodedValue, err := decoded.Content.Fields.GetValue("test_field")
		if err != nil {
			t.Fatalf("[%v] Couldn't get field 'test_field': %v", test.name, err)
		}
		assert.Equal(t, test.value, decodedValue)
	}
}
