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

package readjson

import (
	"bytes"
	stdjson "encoding/json"
	"testing"
	"unsafe"

	sonicDecoder "github.com/bytedance/sonic/decoder"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	benchMediumLine   = []byte(`{"message":"GET /api/users 200","level":"info","timestamp":"2024-01-15T10:30:00Z","duration":142,"method":"GET","path":"/api/users","status":200,"bytes_sent":1024,"user_agent":"Mozilla/5.0","remote_addr":"10.0.0.1"}`)
	benchJournaldLine = []byte(`{"message":"pam_unix(sudo:session): session closed for user root","event":{"kind":"event"},"host":{"hostname":"x-wing","id":"a6a19d57efcf4bf38705c63217a63ba3"},"journald":{"audit":{"login_uid":1000,"session":"1"},"custom":{"syslog_timestamp":"Nov 22 18:10:04 "},"gid":0,"host":{"boot_id":"537d392f028b4dd4b9b1995a4c78cfb6"},"pid":2084586,"process":{"capabilities":"1ffffffffff","command_line":"sudo journalctl --user --rotate","executable":"/usr/bin/sudo","name":"sudo"},"uid":1000},"log":{"syslog":{"appname":"sudo","facility":{"code":10},"priority":6}},"process":{"args":["sudo","journalctl","--user","--rotate"],"args_count":4,"command_line":"sudo journalctl --user --rotate","pid":2084586,"thread":{"capabilities":{"effective":["CAP_CHOWN","CAP_DAC_OVERRIDE","CAP_DAC_READ_SEARCH","CAP_FOWNER","CAP_FSETID","CAP_KILL","CAP_SETGID","CAP_SETUID"]}}}}`)
)

// stdlibUnmarshal is the pre-sonic baseline.
func stdlibUnmarshal(text []byte, fields *map[string]interface{}) error {
	dec := stdjson.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	if err := dec.Decode(fields); err != nil {
		return err
	}
	jsontransform.TransformNumbers(*fields)
	return nil
}

// BenchmarkJSONPipelineE2E benchmarks the full per-event processing pipeline:
// JSON decode → WriteJSONKeys (field merge) → two add_fields processors.
// This mirrors what filestream does for each log line with parsers.ndjson enabled.
//
// sub-benchmarks:
//
//	0_stdlib_baseline  – original encoding/json path (pre-optimization)
//	1_sonic_reader     – production path: reused decoder, UseInt64, unsafe.String aliasing,
//	                     no TransformNumbers walk
//	2_sonic_unsafe_num – UseNumber + TransformNumbers (the old approach); shows what
//	                     eliminating the TransformNumbers walk saves
//	3_sonic_useint64   – same as 1_sonic_reader but using a standalone decoder (not the
//	                     JSONReader wrapper); confirms both paths are equivalent
func BenchmarkJSONPipelineE2E(b *testing.B) {
	labelsProc := addfields.NewAddFields(
		mapstr.M{"labels": mapstr.M{"env": "production", "datacenter": "us-east-1"}},
		false, true,
	)
	serviceProc := addfields.NewAddFields(
		mapstr.M{"service": mapstr.M{"name": "log-collector", "version": "1.0.0"}},
		false, true,
	)

	for _, tc := range []struct {
		name string
		line []byte
	}{
		{"medium_10fields", benchMediumLine},
		{"journald_realistic", benchJournaldLine},
	} {
		b.Run(tc.name, func(b *testing.B) {
			line := tc.line

			b.Run("0_stdlib_baseline", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var jsonFields map[string]interface{}
					_ = stdlibUnmarshal(line, &jsonFields)
					event := &beat.Event{Fields: mapstr.M{}}
					jsontransform.WriteJSONKeys(event, jsonFields, false, true, false)
					event, _ = labelsProc.Run(event)
					event, _ = serviceProc.Run(event)
					_ = event
				}
			})

			// 1_sonic_reader exercises the actual JSONReader.decode() production path:
			// reused decoder + Reset(unsafe.String) + Decode + TransformNumbers.
			b.Run("1_sonic_reader", func(b *testing.B) {
				b.ReportAllocs()
				r := &JSONReader{cfg: &Config{OverwriteKeys: true}, logger: logp.NewLogger("bench")}
				for b.Loop() {
					_, jsonFields := r.decode(line)
					event := &beat.Event{Fields: mapstr.M{}}
					jsontransform.WriteJSONKeys(event, mapstr.M(jsonFields), false, true, false)
					event, _ = labelsProc.Run(event)
					event, _ = serviceProc.Run(event)
					_ = event
				}
			})

			// 2_sonic_unsafe_num: UseNumber + unsafe.String, fresh decoder (no Reset reuse).
			// Isolates the cost of decoder-reuse vs fresh allocation.
			b.Run("2_sonic_unsafe_num", func(b *testing.B) {
				b.ReportAllocs()
				lineStr := unsafe.String(unsafe.SliceData(line), len(line)) //nolint:gosec
				for b.Loop() {
					dc := sonicDecoder.NewDecoder(lineStr)
					dc.UseNumber()
					var jsonFields map[string]interface{}
					_ = dc.Decode(&jsonFields)
					jsontransform.TransformNumbers(jsonFields)
					event := &beat.Event{Fields: mapstr.M{}}
					jsontransform.WriteJSONKeys(event, mapstr.M(jsonFields), false, true, false)
					event, _ = labelsProc.Run(event)
					event, _ = serviceProc.Run(event)
					_ = event
				}
			})

			// 3_sonic_useint64: UseInt64 decodes integers as int64 and floats as
			// float64 in a single pass — no json.Number boxing, no TransformNumbers
			// walk. This is the candidate next optimisation step.
			b.Run("3_sonic_useint64", func(b *testing.B) {
				b.ReportAllocs()
				lineStr := unsafe.String(unsafe.SliceData(line), len(line)) //nolint:gosec
				dc := sonicDecoder.NewDecoder(lineStr)
				dc.UseInt64()
				for b.Loop() {
					var jsonFields map[string]interface{}
					dc.Reset(lineStr)
					_ = dc.Decode(&jsonFields)
					// No TransformNumbers needed: sonic emits int64/float64 directly.
					event := &beat.Event{Fields: mapstr.M{}}
					jsontransform.WriteJSONKeys(event, mapstr.M(jsonFields), false, true, false)
					event, _ = labelsProc.Run(event)
					event, _ = serviceProc.Run(event)
					_ = event
				}
			})
		})
	}
}
