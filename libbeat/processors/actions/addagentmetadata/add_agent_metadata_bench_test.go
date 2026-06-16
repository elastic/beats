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

package addagentmetadata

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func makeEmptyEvent() *beat.Event {
	return &beat.Event{
		Timestamp: time.Now(),
		Fields:    mapstr.M{},
	}
}

func makePopulatedEvent() *beat.Event {
	return &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"host": mapstr.M{
				"name":     "prod-host-01",
				"ip":       "10.0.0.1",
				"hostname": "prod-host-01.example.com",
			},
			"log": mapstr.M{
				"level": "info",
				"file":  mapstr.M{"path": "/var/log/app.log"},
			},
			"message": "application started",
		},
	}
}

// BenchmarkAddAgentMetadata measures the throughput of the single combined
// add_agent_metadata processor across several realistic event shapes.
func BenchmarkAddAgentMetadata(b *testing.B) {
	p := New(testCfg)

	b.Run("empty_event", func(b *testing.B) {
		event := makeEmptyEvent()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := p.Run(event); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("populated_event", func(b *testing.B) {
		event := makePopulatedEvent()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := p.Run(event); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkAddFieldsChain measures the throughput of the equivalent chain of
// individual add_fields processors — the baseline that add_agent_metadata
// is intended to replace.
func BenchmarkAddFieldsChain(b *testing.B) {
	chain := equivalentAddFieldsProcessors(testCfg)

	b.Run("empty_event", func(b *testing.B) {
		event := makeEmptyEvent()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := chain.Run(event); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("populated_event", func(b *testing.B) {
		event := makePopulatedEvent()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := chain.Run(event); err != nil {
				b.Fatal(err)
			}
		}
	})
}
