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

//go:build !integration
// +build !integration

package publish

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/ecs"
	"github.com/elastic/beats/v8/packetbeat/pb"
)

func testEvent() beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"type": "test",
			"src":  &common.Endpoint{},
			"dst":  &common.Endpoint{},
		},
	}
}

// Test that FilterEvent detects events that do not contain the required fields
// and returns error.
func TestFilterEvent(t *testing.T) {
	testCases := []struct {
		f   func() beat.Event
		err string
	}{
		{testEvent, ""},
		{
			func() beat.Event {
				e := testEvent()
				e.Fields["@timestamp"] = time.Now()
				return e
			},
			"duplicate '@timestamp'",
		},
		{
			func() beat.Event {
				e := testEvent()
				e.Timestamp = time.Time{}
				return e
			},
			"missing '@timestamp'",
		},
		{
			func() beat.Event {
				e := testEvent()
				delete(e.Fields, "type")
				return e
			},
			"missing 'type'",
		},
		{
			func() beat.Event {
				e := testEvent()
				e.Fields["type"] = 123
				return e
			},
			"invalid 'type'",
		},
	}

	for _, test := range testCases {
		event := test.f()
		assert.Regexp(t, test.err, validateEvent(&event))
	}
}

func TestPublish(t *testing.T) {
	srcIP, dstIP := "192.145.2.4", "192.145.2.5"

	event := func() *beat.Event {
		return &beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type": "test",
				"_packetbeat": &pb.Fields{
					Source: &ecs.Source{
						IP:   srcIP,
						Port: 3267,
					},
					Destination: &ecs.Destination{
						IP:   dstIP,
						Port: 32232,
					},
				},
			},
		}
	}

	t.Run("direction/inbound", func(t *testing.T) {
		processor := transProcessor{
			localIPs: []net.IP{net.ParseIP(dstIP)},
			name:     "test",
		}

		res, _ := processor.Run(event())
		if res == nil {
			t.Fatalf("event has been filtered out")
		}

		dir, _ := res.GetValue("network.direction")
		assert.Equal(t, "ingress", dir)
	})

	t.Run("direction/outbound", func(t *testing.T) {
		processor := transProcessor{
			localIPs: []net.IP{net.ParseIP(srcIP)},
			name:     "test",
		}

		res, _ := processor.Run(event())
		if res == nil {
			t.Fatalf("event has been filtered out")
		}

		dir, _ := res.GetValue("network.direction")
		assert.Equal(t, "egress", dir)
	})

	t.Run("direction/internal", func(t *testing.T) {
		processor := transProcessor{
			localIPs: []net.IP{net.ParseIP(srcIP), net.ParseIP(dstIP)},
			name:     "test",
		}

		res, _ := processor.Run(event())
		if res == nil {
			t.Fatalf("event has been filtered out")
		}

		dir, _ := res.GetValue("network.direction")
		assert.Equal(t, "ingress", dir)
	})

	t.Run("direction/none", func(t *testing.T) {
		processor := transProcessor{
			localIPs: []net.IP{net.ParseIP(dstIP + "1")},
			name:     "test",
		}

		res, _ := processor.Run(event())
		if res == nil {
			t.Fatalf("event has been filtered out")
		}

		dir, _ := res.GetValue("network.direction")
		assert.Equal(t, "unknown", dir)
	})

	t.Run("ignore_outgoing", func(t *testing.T) {
		processor := transProcessor{
			localIPs:       []net.IP{net.ParseIP(srcIP)},
			ignoreOutgoing: true,
			name:           "test",
		}

		res, err := processor.Run(event())
		if assert.NoError(t, err) {
			assert.Nil(t, res)
		}
	})
}
