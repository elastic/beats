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

package otelmap

import (
	"encoding/json"
	"math"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BenchCase is a named fixture used by both benchmarks and cross-package tests.
type BenchCase struct {
	Name string
	Src  mapstr.M
}


type fixtureStruct struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type (
	fixtureSeverity string
	fixturePort     uint16
)

// BenchmarkCases returns the fixture set shared by benchmarks and cross-package
// comparison tests.
func BenchmarkCases() []BenchCase {
	timestamp := time.Date(2026, 4, 23, 12, 34, 56, 789000000, time.UTC)

	return []BenchCase{
		{
			Name: "minimal",
			Src: mapstr.M{
				"message": "hello world",
			},
		},
		{
			Name: "primitives",
			Src: mapstr.M{
				"string":  "value",
				"int":     int(42),
				"int8":    int8(8),
				"int16":   int16(16),
				"int32":   int32(32),
				"int64":   int64(64),
				"uint":    uint(42),
				"uint8":   uint8(8),
				"uint16":  uint16(16),
				"uint32":  uint32(32),
				"uint64":  uint64(64),
				"float32": float32(3.14),
				"float64": 3.14,
				"bool":    true,
				"null":    nil,
			},
		},
		{
			Name: "typed_slices",
			Src: mapstr.M{
				"strings":   []string{"a", "b", "c", "d", "e"},
				"bools":     []bool{true, false, true, false},
				"ints":      []int{1, 2, 3, 4, 5},
				"int64s":    []int64{10, 20, 30, 40, 50},
				"uints":     []uint{1, 2, 3, 4, 5},
				"uint64s":   []uint64{100, 200, 300},
				"floats":    []float64{1.1, 2.2, 3.3, 4.4},
				"float32s":  []float32{1.1, 2.2, 3.3},
				"times":     []time.Time{timestamp, timestamp.Add(time.Hour)},
				"ctimes":    []common.Time{common.Time(timestamp)},
				"durations": []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
				"any":       []any{1, "two", true, 3.14, nil},
			},
		},
		{
			Name: "times",
			Src: mapstr.M{
				"timestamp":   timestamp,
				"common_time": common.Time(timestamp),
				"duration":    1500 * time.Millisecond,
			},
		},
		{
			Name: "nested_maps",
			Src: mapstr.M{
				"level1": mapstr.M{
					"level2": mapstr.M{
						"level3": mapstr.M{
							"value":   "deep",
							"numbers": []int{1, 2, 3},
						},
					},
					"siblings": []mapstr.M{
						{"id": 1, "tag": "a"},
						{"id": 2, "tag": "b"},
					},
				},
				"plain_map": map[string]any{
					"foo": "bar",
					"qux": 42,
				},
			},
		},
		// edge cases from https://github.com/elastic/elastic-agent/issues/14610
		{
			Name: "edge_cases",
			Src: mapstr.M{
				"neg_float64":        float64(-1.5),
				"neg_float32":        float32(-1.5),
				"zero_float64":       float64(0.0),
				"zero_float32":       float32(0.0),
				"integer_valued_f64": float64(1.0),
				"integer_valued_f32": float32(2.0),
				"max_float64":        math.MaxFloat64,
				"max_float32":        math.MaxFloat32,
				"float_slice_mixed":  []float64{1.5, 2.0, 0.0, -1.0},
				"net_string":         common.NetString("hello"),
				// Encoded as 5.0e-324 instead of 5e-324 due to radix point in ES exporter
				// See https://github.com/elastic/elastic-agent/issues/14610
				// "smallest_nonzero_f64": math.SmallestNonzeroFloat64,
				// "smallest_nonzero_f32": math.SmallestNonzeroFloat32,
			},
		},
		{
			Name: "reflective_types",
			Src: mapstr.M{
				"struct":            fixtureStruct{Name: "svc", Count: 7},
				"struct_pointer":    &fixtureStruct{Name: "svc-p", Count: 9},
				"array":             [3]any{1, "two", true},
				"named_string":      fixtureSeverity("warn"),
				"named_uint":        fixturePort(8080),
				"json_number_int":   json.Number("9223372036854775808"),
				"json_number_float": json.Number("9.223372036854775808"),
			},
		},
		{
			Name: "kitchen_sink",
			Src: mapstr.M{
				"@timestamp": timestamp,
				"message":    "GET /api/orders 200 12ms",
				"trace_id":   "abc123def456",
				"severity":   fixtureSeverity("info"),
				"event": mapstr.M{
					"dataset":  "service.access",
					"created":  common.Time(timestamp.Add(-1500 * time.Millisecond)),
					"duration": 12 * time.Millisecond,
					"category": []string{"network", "web"},
				},
				"host": mapstr.M{
					"name": "edge-01",
					"ip":   []string{"10.0.0.10", "192.168.10.5"},
					"port": fixturePort(8080),
				},
				"http": mapstr.M{
					"request": mapstr.M{
						"method":   "GET",
						"bytes":    int64(1024),
						"headers":  []map[string]any{{"name": "Accept", "value": "*/*"}},
						"trailers": []any{nil},
					},
					"response": fixtureStruct{Name: "ok", Count: 200},
				},
				"labels": []any{"prod", "blue", true, 7},
				"flags": mapstr.M{
					"enabled": true,
					"score":   0.95,
				},
				"data_stream": mapstr.M{
					"type":      "logs",
					"dataset":   "service.access",
					"namespace": "prod",
				},
				"durations": []time.Duration{
					1 * time.Millisecond,
					2 * time.Millisecond,
				},
				"nested": []mapstr.M{
					{
						"id":   1,
						"tags": []string{"a", "b", "c"},
						"meta": mapstr.M{
							"codes": []int{200, 201, 202},
						},
					},
					{
						"id":     2,
						"struct": fixtureStruct{Name: "n2", Count: 4},
					},
				},
			},
		},
	}
}
