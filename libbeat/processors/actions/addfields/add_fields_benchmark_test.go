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

package addfields

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func newTestEvent() *beat.Event {
	return &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"message": "test log message",
			"host": mapstr.M{
				"name": "testhost",
			},
			"agent": mapstr.M{
				"type": "filebeat",
			},
		},
	}
}

// BenchmarkAddFieldsSimple benchmarks adding flat fields (shared=false, overwrite=true).
// This is the simplest case with no cloning needed.
func BenchmarkAddFieldsSimple(b *testing.B) {
	p := NewAddFields(mapstr.M{
		"fields": mapstr.M{"custom_field": "custom_value"},
	}, false, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		_, _ = p.Run(event)
	}
}

// BenchmarkAddFieldsShared benchmarks adding fields with shared=true (requires Clone per event).
// This is the common case for elastic agent injected processors.
func BenchmarkAddFieldsShared(b *testing.B) {
	p := NewAddFields(mapstr.M{
		"fields": mapstr.M{"custom_field": "custom_value"},
	}, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		_, _ = p.Run(event)
	}
}

// BenchmarkAddFieldsNoOverwrite benchmarks add_fields with overwrite=false.
func BenchmarkAddFieldsNoOverwrite(b *testing.B) {
	p := NewAddFields(mapstr.M{
		"fields": mapstr.M{"custom_field": "custom_value"},
	}, true, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		_, _ = p.Run(event)
	}
}

// BenchmarkAddFieldsWithMetadata benchmarks adding fields that target @metadata.
// This exercises the special @metadata handling in event.deepUpdate.
func BenchmarkAddFieldsWithMetadata(b *testing.B) {
	p := NewAddFields(mapstr.M{
		"@metadata": mapstr.M{"input_id": "test-input-123", "stream_id": "test-stream-456"},
	}, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		_, _ = p.Run(event)
	}
}

// BenchmarkAddFieldsAgentInfo simulates the elastic agent pattern of injecting
// agent metadata via add_fields (agent.id, agent.version, agent.snapshot).
func BenchmarkAddFieldsAgentInfo(b *testing.B) {
	p := NewAddFields(mapstr.M{
		"elastic_agent": mapstr.M{
			"id":       "agent-123",
			"snapshot": false,
			"version":  "8.12.0",
		},
	}, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		_, _ = p.Run(event)
	}
}

// BenchmarkAddFieldsDataStream simulates the elastic agent data_stream injection.
func BenchmarkAddFieldsDataStream(b *testing.B) {
	p := NewAddFields(mapstr.M{
		"data_stream": mapstr.M{
			"type":      "logs",
			"dataset":   "system.syslog",
			"namespace": "default",
		},
	}, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		_, _ = p.Run(event)
	}
}

// BenchmarkAddFieldsChain simulates a realistic elastic agent processor chain:
// multiple add_fields processors in sequence, as configured by generate.go.
func BenchmarkAddFieldsChain(b *testing.B) {
	processors := []beat.Processor{
		// Agent info -> elastic_agent
		NewAddFields(mapstr.M{
			"elastic_agent": mapstr.M{
				"id":       "agent-uuid-1234",
				"snapshot": false,
				"version":  "8.12.0",
			},
		}, true, true),
		// Agent info -> agent
		NewAddFields(mapstr.M{
			"agent": mapstr.M{"id": "agent-uuid-1234"},
		}, true, true),
		// Input ID -> @metadata
		NewAddFields(mapstr.M{
			"@metadata": mapstr.M{"input_id": "logfile-system-default"},
		}, true, true),
		// Data stream
		NewAddFields(mapstr.M{
			"data_stream": mapstr.M{
				"type":      "logs",
				"dataset":   "system.syslog",
				"namespace": "default",
			},
		}, true, true),
		// Event dataset
		NewAddFields(mapstr.M{
			"event": mapstr.M{"dataset": "system.syslog"},
		}, true, true),
		// Stream ID -> @metadata
		NewAddFields(mapstr.M{
			"@metadata": mapstr.M{"stream_id": "stream-uuid-5678"},
		}, true, true),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		for _, p := range processors {
			event, _ = p.Run(event)
		}
	}
}

// BenchmarkAddFieldsLargeNestedMap benchmarks adding a large nested map structure,
// similar to what add_host_metadata or add_kubernetes_metadata might produce.
func BenchmarkAddFieldsLargeNestedMap(b *testing.B) {
	p := NewAddFields(mapstr.M{
		"host": mapstr.M{
			"name":         "prod-server-01",
			"hostname":     "prod-server-01.example.com",
			"architecture": "x86_64",
			"os": mapstr.M{
				"type":     "linux",
				"platform": "ubuntu",
				"name":     "Ubuntu",
				"family":   "debian",
				"version":  "22.04",
				"kernel":   "5.15.0-91-generic",
			},
			"ip":  []string{"192.168.1.100", "10.0.0.1"},
			"mac": []string{"00:11:22:33:44:55"},
		},
	}, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		_, _ = p.Run(event)
	}
}

// BenchmarkEventDeepUpdate benchmarks the raw event.DeepUpdate call to measure
// the overhead of the @timestamp/@metadata special key handling.
func BenchmarkEventDeepUpdate(b *testing.B) {
	fields := mapstr.M{
		"elastic_agent": mapstr.M{
			"id":       "agent-123",
			"snapshot": false,
			"version":  "8.12.0",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		event.DeepUpdate(fields)
	}
}

// BenchmarkEventFieldsDeepUpdateDirect benchmarks calling Fields.DeepUpdate directly,
// bypassing the event.DeepUpdate @timestamp/@metadata checks.
func BenchmarkEventFieldsDeepUpdateDirect(b *testing.B) {
	fields := mapstr.M{
		"elastic_agent": mapstr.M{
			"id":       "agent-123",
			"snapshot": false,
			"version":  "8.12.0",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		event.Fields.DeepUpdate(fields)
	}
}

// BenchmarkFullPipeline simulates the complete elastic agent processing pipeline:
// builtin metadata (ecs, host, agent) + elastic agent injected processors (6 add_fields).
// This represents the real-world per-event cost of all add_fields processing.
func BenchmarkFullPipeline(b *testing.B) {
	// Builtin metadata (from MakeDefaultBeatSupport -> WithECS, WithHost, WithAgentMeta)
	builtinMeta := NewAddFields(mapstr.M{
		"ecs":   mapstr.M{"version": "8.0.0"},
		"host":  mapstr.M{"name": "prod-server-01"},
		"agent": mapstr.M{"ephemeral_id": "ephemeral-123", "id": "agent-uuid", "name": "prod-server-01", "type": "filebeat", "version": "8.12.0"},
	}, true, false)

	// Elastic agent injected processors (from generate.go)
	agentProcessors := []beat.Processor{
		NewAddFields(mapstr.M{
			"elastic_agent": mapstr.M{"id": "agent-uuid", "snapshot": false, "version": "8.12.0"},
		}, true, true),
		NewAddFields(mapstr.M{
			"agent": mapstr.M{"id": "agent-uuid"},
		}, true, true),
		NewAddFields(mapstr.M{
			"@metadata": mapstr.M{"input_id": "logfile-system-default"},
		}, true, true),
		NewAddFields(mapstr.M{
			"data_stream": mapstr.M{"type": "logs", "dataset": "system.syslog", "namespace": "default"},
		}, true, true),
		NewAddFields(mapstr.M{
			"event": mapstr.M{"dataset": "system.syslog"},
		}, true, true),
		NewAddFields(mapstr.M{
			"@metadata": mapstr.M{"stream_id": "stream-uuid-5678"},
		}, true, true),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event := newTestEvent()
		event, _ = builtinMeta.Run(event)
		for _, p := range agentProcessors {
			event, _ = p.Run(event)
		}
	}
}

// BenchmarkCloneSmallMap benchmarks cloning a small map (typical add_fields).
func BenchmarkCloneSmallMap(b *testing.B) {
	m := mapstr.M{
		"agent": mapstr.M{"id": "agent-123"},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = m.Clone()
	}
}

// BenchmarkCloneLargeMap benchmarks cloning a large nested map (host metadata).
func BenchmarkCloneLargeMap(b *testing.B) {
	m := mapstr.M{
		"host": mapstr.M{
			"name":         "prod-server-01",
			"hostname":     "prod-server-01.example.com",
			"architecture": "x86_64",
			"os": mapstr.M{
				"type":     "linux",
				"platform": "ubuntu",
				"name":     "Ubuntu",
				"family":   "debian",
				"version":  "22.04",
				"kernel":   "5.15.0-91-generic",
			},
			"ip":  []string{"192.168.1.100", "10.0.0.1"},
			"mac": []string{"00:11:22:33:44:55"},
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = m.Clone()
	}
}
