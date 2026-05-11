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
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

type benchTextMarshaler struct {
	value string
}

func (b benchTextMarshaler) MarshalText() ([]byte, error) {
	return []byte("marshalled:" + b.value), nil
}

type benchStruct struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type (
	benchSeverity string
	benchPort     uint16
)

type benchCase struct {
	name string
	src  mapstr.M
}

// benchmarkCases returns the fixture set shared by tests and benchmarks
func benchmarkCases() []benchCase {
	timestamp := time.Date(2026, 4, 23, 12, 34, 56, 789000000, time.UTC)

	return []benchCase{
		{
			name: "minimal",
			src: mapstr.M{
				"message": "hello world",
			},
		},
		{
			name: "primitives",
			src: mapstr.M{
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
			name: "typed_slices",
			src: mapstr.M{
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
			name: "times",
			src: mapstr.M{
				"timestamp":   timestamp,
				"common_time": common.Time(timestamp),
				"duration":    1500 * time.Millisecond,
			},
		},
		{
			name: "nested_maps",
			src: mapstr.M{
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
		{
			name: "reflective_types",
			src: mapstr.M{
				"struct":         benchStruct{Name: "svc", Count: 7},
				"struct_pointer": &benchStruct{Name: "svc-p", Count: 9},
				"array":          [3]any{1, "two", true},
				"named_string":   benchSeverity("warn"),
				"named_uint":     benchPort(8080),
				"json_number":    json.Number("9223372036854775808"),
				"marshaler":      benchTextMarshaler{value: "ok"},
			},
		},
		{
			name: "kitchen_sink",
			src: mapstr.M{
				"@timestamp": timestamp,
				"message":    "GET /api/orders 200 12ms",
				"trace_id":   "abc123def456",
				"severity":   benchSeverity("info"),
				"event": mapstr.M{
					"dataset":  "service.access",
					"created":  common.Time(timestamp.Add(-1500 * time.Millisecond)),
					"duration": 12 * time.Millisecond,
					"category": []string{"network", "web"},
				},
				"host": mapstr.M{
					"name": "edge-01",
					"ip":   []string{"10.0.0.10", "192.168.10.5"},
					"port": benchPort(8080),
				},
				"http": mapstr.M{
					"request": mapstr.M{
						"method":   "GET",
						"bytes":    int64(1024),
						"headers":  []map[string]any{{"name": "Accept", "value": "*/*"}},
						"trailers": []any{nil},
					},
					"response": benchStruct{Name: "ok", Count: 200},
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
						"struct": benchStruct{Name: "n2", Count: 4},
					},
				},
			},
		},
	}
}

// BenchmarkFromMapstr measures encoding a mapstr.M into a pcommon.Map.
func BenchmarkFromMapstr(b *testing.B) {
	impls := []struct {
		name string
		fn   func(dst pcommon.Map, src mapstr.M) error
	}{
		{name: "default", fn: FromMapstr[mapstr.M]},
		{name: "legacy", fn: func(dst pcommon.Map, src mapstr.M) error {
			clone := src.Clone()
			legacyConvertNonPrimitive(clone)
			return dst.FromRaw(map[string]any(clone))
		}},
	}

	for _, impl := range impls {
		for _, tc := range benchmarkCases() {
			b.Run(impl.name+"/"+tc.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					dst := pcommon.NewMap()
					if err := impl.fn(dst, tc.src); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkToMapstr measures encoding a pcommon.Map into a new mapstr.M
func BenchmarkToMapstr(b *testing.B) {
	for _, tc := range benchmarkCases() {
		src := pcommon.NewMap()
		if err := FromMapstr(src, tc.src); err != nil {
			b.Fatalf("setup: %v", err)
		}

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = ToMapstr(src)
			}
		})
	}
}
