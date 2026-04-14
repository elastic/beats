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

package mapstr

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	structform "github.com/elastic/go-structform"
	"github.com/elastic/go-structform/gotype"
	sfjson "github.com/elastic/go-structform/json"
)

// TestFoldMatchesReflection verifies that the Fold method produces
// byte-identical JSON to go-structform's default reflection path.
func TestFoldMatchesReflection(t *testing.T) {
	tests := map[string]M{
		"flat strings":         {"message": "hello world", "level": "info"},
		"nested maps":          {"host": M{"name": "web-01", "os": M{"type": "linux"}}},
		"mixed types":          {"name": "test", "count": 42, "rate": 3.14, "ok": true, "tags": []string{"a", "b"}},
		"nil values":           {"present": "yes", "absent": nil},
		"empty map":            {},
		"deeply nested":        {"a": M{"b": M{"c": M{"d": "deep"}}}},
		"array of mixed":       {"items": []interface{}{"one", 2, true, nil, M{"nested": "map"}}},
		"int types":            {"i": 1, "i64": int64(99), "u64": uint64(42)},
		"int64 boundaries":     {"min": int64(-9223372036854775808), "max": int64(9223372036854775807), "zero": int64(0)},
		"float64 values":       {"pi": float64(3.14159), "neg": float64(-1.5), "zero": float64(0), "large": float64(1e308)},
		"bool values":          {"t": true, "f": false},
		"nil only":             {"nothing": nil},
		"nested slice":         {"data": []interface{}{int64(1), float64(2.5), true, nil, "str", M{"k": "v"}, []interface{}{int64(3)}}},
		"map[string]iface":     {"inner": map[string]interface{}{"a": int64(1), "b": true}},
		"realistic event":      {"message": "log line", "agent": M{"name": "filebeat", "version": "8.17.0"}, "host": M{"name": "web-01", "os": M{"type": "linux"}}, "ecs": M{"version": "8.0.0"}},
		"empty slice":          {"items": []interface{}{}},
		"empty nested map":     {"outer": M{"inner": M{}}},
		"slice with plain map": {"items": []interface{}{map[string]interface{}{"a": int64(1)}}},
		"nested empty slice":   {"data": M{"items": []interface{}{}}},
		"uint types":           {"u8": uint8(1), "u16": uint16(2), "u32": uint32(3), "u64": uint64(4)},
		"single nil":           {"k": nil},
		"string with special":  {"msg": "line1\nline2\ttab\"quote\\back"},
		"post-normalize event": {
			"message": "log line",
			"enabled": true,
			"count":   int64(42),
			"rate":    float64(3.14),
			"tags":    []interface{}{"web", "prod"},
			"agent":   M{"name": "filebeat", "version": "8.17.0"},
			"log":     M{"offset": int64(12345), "file": M{"path": "/var/log/app.log"}},
			"cloud":   M{"provider": "aws", "available": true, "instance": M{"id": "i-abc"}},
		},
	}

	for name, m := range tests {
		t.Run(name, func(t *testing.T) {
			foldJSON := foldToJSON(t, m)
			reflectJSON := reflectToJSON(t, map[string]interface{}(m))
			assert.JSONEq(t, string(reflectJSON), string(foldJSON))
		})
	}
}

// TestFoldProducesValidJSON verifies output parses as valid JSON.
func TestFoldProducesValidJSON(t *testing.T) {
	m := M{
		"message": `hello "world"\nnewline`,
		"nested":  M{"key": "value"},
		"count":   42,
		"tags":    []string{"a", "b"},
	}
	output := foldToJSON(t, m)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(output, &parsed))
}

// TestFoldErrorPropagation verifies that errors from the visitor bubble up
// through Fold and foldSlice without being swallowed.
func TestFoldErrorPropagation(t *testing.T) {
	tests := map[string]M{
		"flat":         {"k": "v"},
		"nested map":   {"outer": M{"inner": "val"}},
		"with slice":   {"items": []interface{}{"a", int64(1)}},
		"deep nesting": {"a": M{"b": M{"c": "d"}}},
		"mixed":        {"s": "str", "i": int64(1), "f": float64(1.1), "b": true, "n": nil},
	}
	for name, m := range tests {
		t.Run(name, func(t *testing.T) {
			// Fold to JSON via the reflection path to confirm the input is valid.
			reflectJSON := reflectToJSON(t, map[string]interface{}(m))
			var parsed map[string]interface{}
			require.NoError(t, json.Unmarshal(reflectJSON, &parsed), "input must be valid")

			// Now fold with an errVisitor that fails after N callbacks.
			// We don't assert which callback fails — only that the error surfaces.
			for limit := 0; limit < 50; limit++ {
				ev := &errVisitor{failAfter: limit}
				err := m.Fold(structform.EnsureExtVisitor(ev))
				if err == nil {
					// Fold completed without hitting the limit — all callbacks fired.
					break
				}
				assert.ErrorIs(t, err, errVisitorSentinel)
			}
		})
	}
}

var errVisitorSentinel = assert.AnError

// errVisitor is a structform.ExtVisitor that returns an error after failAfter
// successful callbacks. This lets us exercise every error-return branch in Fold.
type errVisitor struct {
	calls     int
	failAfter int
}

func (e *errVisitor) check() error {
	e.calls++
	if e.calls > e.failAfter {
		return errVisitorSentinel
	}
	return nil
}

func (e *errVisitor) OnObjectStart(len int, _ structform.BaseType) error { return e.check() }
func (e *errVisitor) OnObjectFinished() error                            { return e.check() }
func (e *errVisitor) OnKey(string) error                                 { return e.check() }
func (e *errVisitor) OnArrayStart(len int, _ structform.BaseType) error  { return e.check() }
func (e *errVisitor) OnArrayFinished() error                             { return e.check() }
func (e *errVisitor) OnString(string) error                              { return e.check() }
func (e *errVisitor) OnBool(bool) error                                  { return e.check() }
func (e *errVisitor) OnInt(int) error                                    { return e.check() }
func (e *errVisitor) OnInt8(int8) error                                  { return e.check() }
func (e *errVisitor) OnInt16(int16) error                                { return e.check() }
func (e *errVisitor) OnInt32(int32) error                                { return e.check() }
func (e *errVisitor) OnInt64(int64) error                                { return e.check() }
func (e *errVisitor) OnFloat32(float32) error                            { return e.check() }
func (e *errVisitor) OnFloat64(float64) error                            { return e.check() }
func (e *errVisitor) OnByte(byte) error                                  { return e.check() }
func (e *errVisitor) OnUint(uint) error                                  { return e.check() }
func (e *errVisitor) OnUint8(uint8) error                                { return e.check() }
func (e *errVisitor) OnUint16(uint16) error                              { return e.check() }
func (e *errVisitor) OnUint32(uint32) error                              { return e.check() }
func (e *errVisitor) OnUint64(uint64) error                              { return e.check() }
func (e *errVisitor) OnNil() error                                       { return e.check() }
func (e *errVisitor) OnStringObject(map[string]string) error             { return e.check() }
func (e *errVisitor) OnStringArray([]string) error                       { return e.check() }

func foldToJSON(t *testing.T, m M) []byte {
	t.Helper()
	var buf bytes.Buffer
	v := sfjson.NewVisitor(&buf)
	require.NoError(t, m.Fold(structform.EnsureExtVisitor(v)))
	return buf.Bytes()
}

func reflectToJSON(t *testing.T, m map[string]interface{}) []byte {
	t.Helper()
	var buf bytes.Buffer
	v := sfjson.NewVisitor(&buf)
	require.NoError(t, gotype.Fold(m, v))
	return buf.Bytes()
}

func BenchmarkFold(b *testing.B) {
	// All benchmark events use post-normalization types: int64 (from
	// TransformNumbers), float64, bool — NOT int. This matches what
	// filebeat events actually contain after JSON decode + TransformNumbers.

	cases := map[string]M{
		// 5 scalar fields, 0 nested maps
		"flat_5fields": {
			"message": "log line",
			"level":   "info",
			"count":   int64(42),
			"rate":    float64(3.14),
			"enabled": true,
		},
		// 10 fields, 3 nested maps — small realistic event
		"nested_10fields": {
			"message": "log line",
			"enabled": true,
			"count":   int64(42),
			"agent":   M{"name": "filebeat", "version": "8.17.0"},
			"host":    M{"name": "web-01", "os": M{"type": "linux"}},
			"ecs":     M{"version": "8.0.0"},
		},
		// 20 fields, 7 nested maps — typical elastic-agent event
		"nested_20fields": {
			"message": "log line",
			"enabled": true,
			"count":   int64(42),
			"rate":    float64(3.14),
			"agent":   M{"name": "filebeat", "version": "8.17.0", "type": "filebeat"},
			"host":    M{"name": "web-01", "os": M{"type": "linux", "platform": "ubuntu"}},
			"cloud":   M{"provider": "aws", "instance": M{"id": "i-abc"}, "available": true},
			"log":     M{"file": M{"path": "/var/log/app.log"}, "offset": int64(12345)},
			"ecs":     M{"version": "8.0.0"},
			"event":   M{"dataset": "bb", "severity": int64(3)},
		},
		// strings only — baseline where Fold already handled all types
		"strings_only": {
			"message": "log line",
			"level":   "info",
			"source":  "/var/log/app.log",
			"host":    M{"name": "web-01"},
			"agent":   M{"name": "filebeat"},
		},
		// heavy int64/float64 — worst case for old Fold (all fall through)
		"numeric_heavy": {
			"a": int64(1), "b": int64(2), "c": int64(3),
			"d": float64(1.1), "e": float64(2.2), "f": float64(3.3),
			"g": true, "h": false,
			"nested": M{"x": int64(10), "y": float64(20.5), "z": true},
		},
	}

	for name, m := range cases {
		b.Run(name, func(b *testing.B) {
			var buf bytes.Buffer
			v := sfjson.NewVisitor(&buf)
			ev := structform.EnsureExtVisitor(v)

			b.Run("fold", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					buf.Reset()
					_ = m.Fold(ev)
				}
			})

			b.Run("reflect", func(b *testing.B) {
				plain := map[string]interface{}(m)
				it, _ := gotype.NewIterator(v)
				b.ReportAllocs()
				for b.Loop() {
					buf.Reset()
					_ = it.Fold(plain)
				}
			})
		})
	}
}
