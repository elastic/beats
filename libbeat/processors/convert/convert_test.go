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

package convert

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestConvert(t *testing.T) {
	t.Run("ignore_missing", func(t *testing.T) {
		c := defaultConfig()
		c.Fields = append(c.Fields, field{From: "src", To: "dst", Type: Integer})

		p, err := newConvert(c)
		if err != nil {
			t.Fatal(err)
		}

		evt := &beat.Event{Fields: common.MapStr{}}

		// Defaults.
		p.IgnoreMissing = false
		p.FailOnError = true
		_, err = p.Run(evt)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "field [src] is missing")
		}

		p.IgnoreMissing = true
		p.FailOnError = true
		_, err = p.Run(evt)
		if err != nil {
			t.Fatal(err)
		}

		p.IgnoreMissing = true
		p.FailOnError = false
		_, err = p.Run(evt)
		if err != nil {
			t.Fatal(err)
		}

		p.IgnoreMissing = false
		p.FailOnError = false
		_, err = p.Run(evt)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("fail_on_error", func(t *testing.T) {
		c := defaultConfig()
		c.Fields = append(c.Fields, field{From: "source.address", To: "source.ip", Type: IP})

		p, err := newConvert(c)
		if err != nil {
			t.Fatal(err)
		}

		evt := &beat.Event{Fields: common.MapStr{"source": common.MapStr{"address": "host.local"}}}

		_, err = p.Run(evt)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "unable to convert")
		}

		p.FailOnError = false
		_, err = p.Run(evt)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("mode", func(t *testing.T) {
		c := defaultConfig()
		c.Fields = append(c.Fields, field{From: "source.address", To: "source.ip", Type: IP})

		p, err := newConvert(c)
		if err != nil {
			t.Fatal(err)
		}

		const loopback = "127.0.0.1"
		fields := common.MapStr{"source": common.MapStr{"address": loopback}}

		t.Run("copy", func(t *testing.T) {
			evt := &beat.Event{Fields: fields.Clone()}
			evt, err = p.Run(evt)
			if err != nil {
				t.Fatal(err)
			}
			address, _ := evt.GetValue("source.address")
			assert.Equal(t, loopback, address)
			ip, _ := evt.GetValue("source.ip")
			assert.Equal(t, loopback, ip)
		})

		t.Run("rename", func(t *testing.T) {
			p.Mode = renameMode

			evt := &beat.Event{Fields: fields.Clone()}
			evt, err = p.Run(evt)
			if err != nil {
				t.Fatal(err)
			}
			_, err := evt.GetValue("source.address")
			assert.Error(t, err)
			ip, _ := evt.GetValue("source.ip")
			assert.Equal(t, loopback, ip)
		})
	})

	t.Run("string", func(t *testing.T) {
		c := defaultConfig()
		c.Tag = "convert_ip"
		c.Fields = append(c.Fields, field{From: "source.address", To: "source.ip", Type: IP})

		p, err := newConvert(c)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, `convert={"Fields":`+
			`[{"From":"source.address","To":"source.ip","Type":"ip"}],`+
			`"Tag":"convert_ip","IgnoreMissing":false,"FailOnError":true,"Mode":"copy"}`,
			p.String())
	})

	t.Run("metadata as a target", func(t *testing.T) {
		c := defaultConfig()
		c.Tag = "convert_ip"
		c.Fields = append(c.Fields, field{From: "@metadata.source", To: "@metadata.dest", Type: Integer})

		evt := &beat.Event{
			Meta: common.MapStr{
				"source": "1",
			},
		}
		expMeta := common.MapStr{
			"source": "1",
			"dest":   int32(1),
		}

		p, err := newConvert(c)
		assert.NoError(t, err)

		newEvt, err := p.Run(evt)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, newEvt.Meta)
		assert.Equal(t, evt.Fields, newEvt.Fields)
	})
}

func TestConvertRun(t *testing.T) {
	tests := map[string]struct {
		config      common.MapStr
		input       beat.Event
		expected    beat.Event
		fail        bool
		errContains string
	}{
		"missing field": {
			config: common.MapStr{
				"fields": []common.MapStr{
					{"from": "port", "type": "integer"},
					{"from": "address", "to": "ip", "type": "ip"},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"port": "80",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"port": "80",
				},
			},
			fail: true,
		},
		"put error no clone": {
			config: common.MapStr{
				"fields": []common.MapStr{
					{"from": "port", "to": "port.number", "type": "integer"},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"port": "80",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"port": "80",
				},
			},
			fail: true,
		},
		"put error with clone": {
			config: common.MapStr{
				"fields": []common.MapStr{
					{"from": "id", "to": "event.id", "type": "integer"},
					{"from": "port", "to": "port.number", "type": "integer"},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"id":   "32",
					"port": "80",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"id":   "32",
					"port": "80",
				},
			},
			fail: true,
		},
		"invalid conversion": {
			config: common.MapStr{
				"fields": []common.MapStr{
					{"from": "address", "to": "ip", "type": "ip"},
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"address": "-",
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"address": "-",
				},
			},
			fail:        true,
			errContains: "unable to convert value [-]: value is not a valid IP address",
		},
	}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			processor, err := New(conf.MustNewConfigFrom(tt.config))
			if err != nil {
				t.Fatal(err)
			}
			result, err := processor.Run(&tt.input)
			if tt.expected.Fields != nil {
				assert.Equal(t, tt.expected.Fields.Flatten(), result.Fields.Flatten())
				assert.Equal(t, tt.expected.Meta.Flatten(), result.Meta.Flatten())
				assert.Equal(t, tt.expected.Timestamp, result.Timestamp)
			}
			if tt.fail {
				assert.Error(t, err)
				t.Log("got expected error", err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

type testCase struct {
	Type dataType
	In   interface{}
	Out  interface{}
	Err  bool
}

var testCases = []testCase{
	{String, nil, nil, true},
	{String, "x", "x", false},
	{String, 1, "1", false},
	{String, 1.1, "1.1", false},
	{String, true, "true", false},

	{Long, nil, nil, true},
	{Long, "x", nil, true},
	{Long, "0x", nil, true},
	{Long, "0b1", nil, true},
	{Long, "1x2", nil, true},
	{Long, true, nil, true},
	{Long, "1", int64(1), false},
	{Long, "-1", int64(-1), false},
	{Long, "017", int64(17), false},
	{Long, "08", int64(8), false},
	{Long, "0X0A", int64(10), false},
	{Long, "-0x12", int64(-18), false},
	{Long, int(1), int64(1), false},
	{Long, int8(1), int64(1), false},
	{Long, int16(1), int64(1), false},
	{Long, int32(1), int64(1), false},
	{Long, int64(1), int64(1), false},
	{Long, uint(1), int64(1), false},
	{Long, uint8(1), int64(1), false},
	{Long, uint16(1), int64(1), false},
	{Long, uint32(1), int64(1), false},
	{Long, uint64(1), int64(1), false},
	{Long, float32(1), int64(1), false},
	{Long, float64(1), int64(1), false},

	{Integer, nil, nil, true},
	{Integer, "x", nil, true},
	{Integer, true, nil, true},
	{Integer, "x", nil, true},
	{Integer, "0x", nil, true},
	{Integer, "0b1", nil, true},
	{Integer, "1x2", nil, true},
	{Integer, true, nil, true},
	{Integer, "1", int32(1), false},
	{Integer, "-1", int32(-1), false},
	{Integer, "017", int32(17), false},
	{Integer, "08", int32(8), false},
	{Integer, "0X0A", int32(10), false},
	{Integer, "-0x12", int32(-18), false},
	{Integer, "1", int32(1), false},
	{Integer, int(1), int32(1), false},
	{Integer, int8(1), int32(1), false},
	{Integer, int16(1), int32(1), false},
	{Integer, int32(1), int32(1), false},
	{Integer, int64(1), int32(1), false},
	{Integer, uint(1), int32(1), false},
	{Integer, uint8(1), int32(1), false},
	{Integer, uint16(1), int32(1), false},
	{Integer, uint32(1), int32(1), false},
	{Integer, uint64(1), int32(1), false},
	{Integer, float32(1), int32(1), false},
	{Integer, float64(1), int32(1), false},

	{Float, nil, nil, true},
	{Float, "x", nil, true},
	{Float, true, nil, true},
	{Float, "1", float32(1), false},
	{Float, "1.1", float32(1.1), false},
	{Float, int(1), float32(1), false},
	{Float, int8(1), float32(1), false},
	{Float, int16(1), float32(1), false},
	{Float, int32(1), float32(1), false},
	{Float, int64(1), float32(1), false},
	{Float, uint(1), float32(1), false},
	{Float, uint8(1), float32(1), false},
	{Float, uint16(1), float32(1), false},
	{Float, uint32(1), float32(1), false},
	{Float, uint64(1), float32(1), false},
	{Float, float32(1), float32(1), false},
	{Float, float64(1), float32(1), false},

	{Double, nil, nil, true},
	{Double, "x", nil, true},
	{Double, true, nil, true},
	{Double, "1", float64(1), false},
	{Double, "1.1", float64(1.1), false},
	{Double, int(1), float64(1), false},
	{Double, int8(1), float64(1), false},
	{Double, int16(1), float64(1), false},
	{Double, int32(1), float64(1), false},
	{Double, int64(1), float64(1), false},
	{Double, uint(1), float64(1), false},
	{Double, uint8(1), float64(1), false},
	{Double, uint16(1), float64(1), false},
	{Double, uint32(1), float64(1), false},
	{Double, uint64(1), float64(1), false},
	{Double, float32(1), float64(1), false},
	{Double, float64(1), float64(1), false},

	{Boolean, nil, nil, true},
	{Boolean, "x", nil, true},
	{Boolean, 1, nil, true},
	{Boolean, 0, nil, true},
	{Boolean, "TrUe", nil, true},
	{Boolean, true, true, false},
	{Boolean, "1", true, false},
	{Boolean, "t", true, false},
	{Boolean, "T", true, false},
	{Boolean, "TRUE", true, false},
	{Boolean, "true", true, false},
	{Boolean, "True", true, false},
	{Boolean, false, false, false},
	{Boolean, "0", false, false},
	{Boolean, "f", false, false},
	{Boolean, "F", false, false},
	{Boolean, "FALSE", false, false},
	{Boolean, "false", false, false},
	{Boolean, "False", false, false},

	{IP, nil, nil, true},
	{IP, "x", nil, true},
	{IP, "365.0.0.0", "365.0.0.0", true},
	{IP, "0.0.0.0", "0.0.0.0", false},
	{IP, "::1", "::1", false},
}

func TestDataTypes(t *testing.T) {
	const key = "key"

	for _, tc := range testCases {
		// Give the test a friendly name.
		var prefix string
		if tc.Err {
			prefix = "cannot "
		}
		name := fmt.Sprintf("%v%T %v to %v", prefix, tc.In, tc.In, tc.Type)

		tc := tc
		t.Run(name, func(t *testing.T) {
			c := defaultConfig()
			c.Fields = append(c.Fields, field{From: key, Type: tc.Type})

			p, err := newConvert(c)
			if err != nil {
				t.Fatal(err)
			}

			event, err := p.Run(&beat.Event{Fields: common.MapStr{key: tc.In}})
			if tc.Err {
				assert.Error(t, err)
				return
			} else if err != nil {
				t.Fatalf("%+v", err)
			}

			v := event.Fields[key]
			assert.Equal(t, tc.Out, v)
		})
	}
}

func BenchmarkTestConvertRun(b *testing.B) {
	c := defaultConfig()
	c.IgnoreMissing = true
	c.Fields = append(c.Fields,
		field{From: "source.address", To: "source.ip", Type: IP},
		field{From: "destination.address", To: "destination.ip", Type: IP},
		field{From: "a", To: "b"},
		field{From: "c", To: "d"},
		field{From: "e", To: "f"},
		field{From: "g", To: "h"},
		field{From: "i", To: "j"},
		field{From: "k", To: "l"},
		field{From: "m", To: "n"},
		field{From: "o", To: "p"},
	)

	p, err := newConvert(c)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := &beat.Event{
				Fields: common.MapStr{
					"source": common.MapStr{
						"address": "192.51.100.1",
					},
					"destination": common.MapStr{
						"address": "192.0.2.51",
					},
				},
			}

			_, err := p.Run(event)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
