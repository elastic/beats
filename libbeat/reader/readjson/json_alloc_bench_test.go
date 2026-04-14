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

	jsoniter "github.com/json-iterator/go"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// stdlibUnmarshal is the pre-iterator implementation, kept here as the oracle
// baseline for TestIterParseMatchesUnmarshal and the stdlib comparison benchmarks.
func stdlibUnmarshal(text []byte, fields *map[string]interface{}) error {
	dec := stdjson.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	if err := dec.Decode(fields); err != nil {
		return err
	}
	jsontransform.TransformNumbers(*fields)
	return nil
}

var benchMediumLine = []byte(`{"message":"GET /api/users 200","level":"info","timestamp":"2024-01-15T10:30:00Z","duration":142,"method":"GET","path":"/api/users","status":200,"bytes_sent":1024,"user_agent":"Mozilla/5.0","remote_addr":"10.0.0.1"}`)

// benchJournaldLine is derived from filebeat/input/journald/testdata/input-multiline-parser-events.json.
// It has 6 root map fields, numbers, arrays, and 3-level nesting — representative of what
// journald input produces after JSON parsing.
var benchJournaldLine = []byte(`{"message":"pam_unix(sudo:session): session closed for user root","event":{"kind":"event"},"host":{"hostname":"x-wing","id":"a6a19d57efcf4bf38705c63217a63ba3"},"journald":{"audit":{"login_uid":1000,"session":"1"},"custom":{"syslog_timestamp":"Nov 22 18:10:04 "},"gid":0,"host":{"boot_id":"537d392f028b4dd4b9b1995a4c78cfb6"},"pid":2084586,"process":{"capabilities":"1ffffffffff","command_line":"sudo journalctl --user --rotate","executable":"/usr/bin/sudo","name":"sudo"},"uid":1000},"log":{"syslog":{"appname":"sudo","facility":{"code":10},"priority":6}},"process":{"args":["sudo","journalctl","--user","--rotate"],"args_count":4,"command_line":"sudo journalctl --user --rotate","pid":2084586,"thread":{"capabilities":{"effective":["CAP_CHOWN","CAP_DAC_OVERRIDE","CAP_DAC_READ_SEARCH","CAP_FOWNER","CAP_FSETID","CAP_KILL","CAP_SETGID","CAP_SETUID"]}}}}`)

// BenchmarkJSONPipelineE2E benchmarks the full per-event processing pipeline:
// JSON decode → WriteJSONKeys (field merge) → two add_fields processors.
// This mirrors what filestream does for each log line with parsers.ndjson enabled.
//
// A "0_processors_only" sub-benchmark shows the floor cost so you can read off
// what fraction of total time is the JSON decode step.
func BenchmarkJSONPipelineE2E(b *testing.B) {
	// Two realistic add_fields processors that would sit after JSON parsing.
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

			// 0. Floor: processors only, no JSON decode.
			// Shows how much of the total pipeline cost is NOT the decoder.
			b.Run("0_processors_only", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					event := &beat.Event{Fields: mapstr.M{"message": "test"}}
					event, _ = labelsProc.Run(event)
					event, _ = serviceProc.Run(event)
					_ = event
				}
			})

			// 1. Pre-iterator baseline: stdlib decoder + WriteJSONKeys + processors.
			b.Run("1_stdlib_pipeline", func(b *testing.B) {
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

			// 2. jsoniter iterator + WriteJSONKeys + processors.
			b.Run("2_iterator_pipeline", func(b *testing.B) {
				b.ReportAllocs()
				iter := jsoniter.NewIterator(jsoniterAPI)
				for b.Loop() {
					iter.ResetBytes(line)
					jsonFields := iterParseObject(iter)
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
