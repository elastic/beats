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
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BenchmarkJSONPipelineE2E benchmarks the full per-event processing pipeline:
// JSON decode → WriteJSONKeys (field merge) → two add_fields processors.
// This mirrors what filestream does for each log line with parsers.ndjson enabled.
//
// The "0_stdlib_baseline" sub-benchmark shows the pre-jsoniter cost so you can
// read off what fraction of total time the switch to jsoniter eliminates.
//
// For the full file-read pipeline benchmark (os.File → readfile → parser.Config.Create),
// see BenchmarkNDJSONFilePipeline in libbeat/reader/parser/parser_bench_test.go.
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

			b.Run("1_jsoniter_reuse", func(b *testing.B) {
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
