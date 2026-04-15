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

package actions

import (
	stdjson "encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// benchDJFLine is a representative 10-field JSON event similar to what
// decode_json_fields would see from a structured log source.
var benchDJFLine = `{"message":"GET /api/users 200","level":"info","timestamp":"2024-01-15T10:30:00Z","duration":142,"method":"GET","path":"/api/users","status":200,"bytes_sent":1024,"user_agent":"Mozilla/5.0","remote_addr":"10.0.0.1"}`

// stdlibDecodeJSONOld mirrors the old decodeJSON implementation:
// json.NewDecoder + UseNumber + Decode + TransformNumbers. A new Decoder
// is allocated per call, which is the pre-iterator behaviour.
func stdlibDecodeJSONOld(text string, to *interface{}) error {
	dec := stdjson.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	if err := dec.Decode(to); err != nil {
		return err
	}
	if dec.More() {
		return errProcessingSkipped
	}
	if _, err := dec.Token(); err != nil && err != io.EOF {
		return err
	}
	if m, ok := (*to).(map[string]interface{}); ok {
		jsontransform.TransformNumbers(m)
	}
	return nil
}

// BenchmarkDecodeJSONFields compares:
//
//   - decode_only/stdlib_baseline: the old per-call decoder + TransformNumbers
//   - decode_only/jsoniter_reuse:  the new reusable iterator path (decode only)
//   - full_run/jsoniter_reuse:     the complete processor Run() for absolute-cost context
func BenchmarkDecodeJSONFields(b *testing.B) {
	log := logptest.NewTestingLogger(b, "")
	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"fields":         []string{"msg"},
		"overwrite_keys": true,
	})
	proc, err := NewDecodeJSONFields(cfg, log)
	if err != nil {
		b.Fatal(err)
	}
	f := proc.(*decodeJSONFields)

	b.Run("decode_only", func(b *testing.B) {
		b.Run("stdlib_baseline", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out interface{}
				_ = stdlibDecodeJSONOld(benchDJFLine, &out)
			}
		})

		b.Run("jsoniter_reuse", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out interface{}
				_ = f.unmarshal(f.maxDepth, benchDJFLine, &out, f.processArray)
			}
		})
	})

	b.Run("full_run/jsoniter_reuse", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			event := &beat.Event{Fields: mapstr.M{"msg": benchDJFLine}}
			_, _ = proc.Run(event)
		}
	})
}
