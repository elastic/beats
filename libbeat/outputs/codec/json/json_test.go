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

package json

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestJsonCodec(t *testing.T) {
	expectedValue := `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"doc","version":"1.2.3"},"msg":"message"}`

	codec := New(false, "1.2.3")
	output, err := codec.Encode("test", &beat.Event{Fields: common.MapStr{"msg": "message"}})

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}

func TestJsonWriterPrettyPrint(t *testing.T) {
	expectedValue := `{
  "@timestamp": "0001-01-01T00:00:00.000Z",
  "@metadata": {
    "beat": "test",
    "type": "doc",
    "version": "1.2.3"
  },
  "msg": "message"
}`

	codec := New(true, "1.2.3")
	output, err := codec.Encode("test", &beat.Event{Fields: common.MapStr{"msg": "message"}})

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}
