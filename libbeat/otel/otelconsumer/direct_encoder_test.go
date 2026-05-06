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

package otelconsumer

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/plog"
)

func TestFillLogRecordFromEventMatchesFromRawOracle(t *testing.T) {
	logger := logp.NewNopLogger()
	beatInfo := beat.Info{}

	for _, tc := range append(benchmarkEventCases(), directEncoderOracleCases()...) {
		t.Run(tc.name, func(t *testing.T) {
			pubEvent := publisher.Event{Content: tc.event}

			oracle := plog.NewLogRecord()
			current := plog.NewLogRecord()

			if err := fillLogRecordFromEventFromRaw(oracle, pubEvent, beatInfo, logger, false); err != nil {
				t.Fatal(err)
			}
			if err := fillLogRecordFromEvent(current, pubEvent, beatInfo, logger, false); err != nil {
				t.Fatal(err)
			}

			assertLogRecordsEquivalent(t, oracle, current)
		})
	}
}

func TestFillLogRecordFromEventWithMetadataMatchesFromRawOracle(t *testing.T) {
	logger := logp.NewNopLogger()
	beatInfo := beat.Info{
		IncludeMetadata: true,
		Beat:            "filebeat",
		Version:         "9.0.0",
	}

	pubEvent := publisher.Event{Content: beat.Event{
		Timestamp: time.Date(2026, 4, 23, 12, 34, 56, 789000000, time.UTC),
		Meta: mapstr.M{
			"id":        "1",
			"routing":   "hot",
			"arbitrary": []string{"a", "b"},
		},
		Fields: mapstr.M{
			"message": "hello world",
			"event": mapstr.M{
				"dataset": "service.access",
			},
		},
	}}

	oracle := plog.NewLogRecord()
	current := plog.NewLogRecord()

	if err := fillLogRecordFromEventFromRaw(oracle, pubEvent, beatInfo, logger, false); err != nil {
		t.Fatal(err)
	}
	if err := fillLogRecordFromEvent(current, pubEvent, beatInfo, logger, false); err != nil {
		t.Fatal(err)
	}

	assertLogRecordsEquivalent(t, oracle, current)
}

func TestRealWorldCasesAreDirectEncodable(t *testing.T) {
	logger := logp.NewNopLogger()
	beatInfo := beat.Info{}

	for _, tc := range directEncoderOracleCases() {
		if tc.name != "packetbeat_flow_like" && tc.name != "metricbeat_sql_like" {
			continue
		}

		t.Run(tc.name, func(t *testing.T) {
			pubEvent := publisher.Event{Content: tc.event}
			logRecord := plog.NewLogRecord()
			beatEvent := prepareLogRecordFromEvent(logRecord, pubEvent, logger, false)

			if err := tryEncodeLogRecordBodyDirect(logRecord, beatEvent, tc.event.Timestamp, logBodyMetadata(pubEvent, beatInfo)); err != nil {
				t.Fatalf("expected direct encoding, got %v", err)
			}
		})
	}
}

func TestBenchmarkCasesAreDirectEncodable(t *testing.T) {
	logger := logp.NewNopLogger()
	beatInfo := beat.Info{}

	for _, tc := range benchmarkEventCases() {
		t.Run(tc.name, func(t *testing.T) {
			pubEvent := publisher.Event{Content: tc.event}
			logRecord := plog.NewLogRecord()
			beatEvent := prepareLogRecordFromEvent(logRecord, pubEvent, logger, false)

			if err := tryEncodeLogRecordBodyDirect(logRecord, beatEvent, tc.event.Timestamp, logBodyMetadata(pubEvent, beatInfo)); err != nil {
				t.Fatalf("expected direct encoding, got %v", err)
			}
		})
	}
}

func TestMetadataCaseIsDirectEncodable(t *testing.T) {
	logger := logp.NewNopLogger()
	beatInfo := beat.Info{
		IncludeMetadata: true,
		Beat:            "filebeat",
		Version:         "9.0.0",
	}

	pubEvent := publisher.Event{Content: beat.Event{
		Timestamp: time.Date(2026, 4, 23, 12, 34, 56, 789000000, time.UTC),
		Meta: mapstr.M{
			"id":        "1",
			"routing":   "hot",
			"arbitrary": []string{"a", "b"},
		},
		Fields: mapstr.M{
			"message": "hello world",
			"event": mapstr.M{
				"dataset": "service.access",
			},
		},
	}}

	logRecord := plog.NewLogRecord()
	beatEvent := prepareLogRecordFromEvent(logRecord, pubEvent, logger, false)
	if err := tryEncodeLogRecordBodyDirect(logRecord, beatEvent, pubEvent.Content.Timestamp, logBodyMetadata(pubEvent, beatInfo)); err != nil {
		t.Fatalf("expected direct encoding, got %v", err)
	}
}

type oracleTextMarshaler struct {
	value string
}

func (o oracleTextMarshaler) MarshalText() ([]byte, error) {
	return []byte("marshal:" + o.value), nil
}

type oracleStruct struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

func directEncoderOracleCases() []benchmarkEventCase {
	timestamp := time.Date(2026, 4, 23, 12, 34, 56, 789000000, time.UTC)

	return []benchmarkEventCase{
		{
			name: "special_types",
			event: beat.Event{
				Timestamp: timestamp,
				Fields: mapstr.M{
					"bytes":          []byte{1, 2, 3},
					"marshaler":      oracleTextMarshaler{value: "ok"},
					"struct":         oracleStruct{Value: "x", Count: 2},
					"dotted.field":   "literal",
					"max_uint64":     ^uint64(0),
					"uint64_values":  []uint64{0, 1, ^uint64(0)},
					"array":          [3]any{1, "two", true},
					"nested_structs": []map[string]any{{"service": "api"}, {"count": 2}},
				},
			},
		},
		{
			name: "nested_unsupported_types",
			event: beat.Event{
				Timestamp: timestamp,
				Fields: mapstr.M{
					"service": mapstr.M{
						"name": "api",
						"meta": map[string]any{
							"owner":  "team-a",
							"struct": oracleStruct{Value: "nested", Count: 3},
						},
					},
				},
			},
		},
		{
			name: "packetbeat_flow_like",
			event: beat.Event{
				Timestamp: timestamp,
				Fields: mapstr.M{
					"event": mapstr.M{
						"start":    common.Time(timestamp.Add(-2 * time.Second)),
						"end":      common.Time(timestamp),
						"duration": 2 * time.Second,
						"category": []string{"network"},
						"type":     []string{"connection", "end"},
					},
					"flow": mapstr.M{
						"id":    common.NetString("flow-id"),
						"final": true,
						"vlan":  []uint64{100, 200},
					},
					"network": mapstr.M{
						"bytes":        uint64(1234),
						"packets":      uint64(12),
						"community_id": "1:abc",
					},
				},
			},
		},
		{
			name: "metricbeat_sql_like",
			event: beat.Event{
				Timestamp: timestamp,
				Fields: mapstr.M{
					"sql": mapstr.M{
						"row": mapstr.M{
							"string":         "000400",
							"unsigned_int":   uint64(100),
							"array":          []any{0, 1, 2},
							"byte_array":     "byte_array",
							"formatted_time": timestamp.Format(time.RFC3339Nano),
						},
					},
				},
			},
		},
	}
}
