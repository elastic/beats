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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestAddFieldsBehavior tests that each code path in the optimized Run()
// produces identical results to the original (event.DeepUpdate-based) behavior.
func TestAddFieldsBehavior(t *testing.T) {
	t.Run("fast path - regular fields only", func(t *testing.T) {
		t.Run("overwrite=true, shared=true", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"elastic_agent": mapstr.M{
					"id":       "agent-123",
					"snapshot": false,
					"version":  "8.12.0",
				},
			}, true, true)

			event := &beat.Event{
				Timestamp: time.Now(),
				Fields: mapstr.M{
					"message": "hello",
					"elastic_agent": mapstr.M{
						"id":   "old-id",
						"name": "old-name",
					},
				},
			}

			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Verify overwrite happened
			ea, err := result.GetValue("elastic_agent.id")
			require.NoError(t, err)
			assert.Equal(t, "agent-123", ea)

			// Verify existing field merged (not deleted)
			name, err := result.GetValue("elastic_agent.name")
			require.NoError(t, err)
			assert.Equal(t, "old-name", name)

			// Verify other fields untouched
			msg, err := result.GetValue("message")
			require.NoError(t, err)
			assert.Equal(t, "hello", msg)
		})

		t.Run("overwrite=true, shared=false", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"agent": mapstr.M{"id": "new-id"},
			}, false, true)

			event := &beat.Event{
				Fields: mapstr.M{
					"agent": mapstr.M{"id": "old-id", "type": "filebeat"},
				},
			}

			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)

			id, err := result.GetValue("agent.id")
			require.NoError(t, err)
			assert.Equal(t, "new-id", id)

			typ, err := result.GetValue("agent.type")
			require.NoError(t, err)
			assert.Equal(t, "filebeat", typ)
		})

		t.Run("overwrite=false, shared=true", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"agent": mapstr.M{
					"id":   "new-id",
					"type": "heartbeat",
				},
			}, true, false)

			event := &beat.Event{
				Fields: mapstr.M{
					"agent": mapstr.M{"id": "existing-id"},
				},
			}

			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)

			// id should NOT be overwritten
			id, err := result.GetValue("agent.id")
			require.NoError(t, err)
			assert.Equal(t, "existing-id", id)

			// type should be added (it didn't exist)
			typ, err := result.GetValue("agent.type")
			require.NoError(t, err)
			assert.Equal(t, "heartbeat", typ)
		})

		t.Run("nil event fields", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"new_field": "value",
			}, false, true)

			event := &beat.Event{Fields: nil}
			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)

			v, err := result.GetValue("new_field")
			require.NoError(t, err)
			assert.Equal(t, "value", v)
		})

		t.Run("nil event", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"new_field": "value",
			}, false, true)

			result, err := p.Run(nil)
			require.NoError(t, err)
			require.Nil(t, result)
		})

		t.Run("empty fields", func(t *testing.T) {
			p := NewAddFields(mapstr.M{}, false, true)

			event := &beat.Event{
				Fields: mapstr.M{"keep": "this"},
			}
			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, mapstr.M{"keep": "this"}, result.Fields)
		})
	})

	t.Run("metadata path - @metadata only", func(t *testing.T) {
		t.Run("basic metadata injection", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"@metadata": mapstr.M{
					"input_id":  "test-input",
					"stream_id": "test-stream",
				},
			}, true, true)

			event := &beat.Event{
				Fields: mapstr.M{"message": "hello"},
			}

			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Fields should be untouched
			assert.Equal(t, mapstr.M{"message": "hello"}, result.Fields)

			// Meta should have the injected values
			v, err := result.GetValue("@metadata.input_id")
			require.NoError(t, err)
			assert.Equal(t, "test-input", v)

			v, err = result.GetValue("@metadata.stream_id")
			require.NoError(t, err)
			assert.Equal(t, "test-stream", v)
		})

		t.Run("merge with existing metadata", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"@metadata": mapstr.M{
					"stream_id": "new-stream",
				},
			}, true, true)

			event := &beat.Event{
				Meta: mapstr.M{
					"_id":       "doc-123",
					"stream_id": "old-stream",
				},
				Fields: mapstr.M{"message": "hello"},
			}

			result, err := p.Run(event)
			require.NoError(t, err)

			// _id should be preserved
			v, err := result.GetValue("@metadata._id")
			require.NoError(t, err)
			assert.Equal(t, "doc-123", v)

			// stream_id should be overwritten
			v, err = result.GetValue("@metadata.stream_id")
			require.NoError(t, err)
			assert.Equal(t, "new-stream", v)
		})

		t.Run("metadata no-overwrite", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"@metadata": mapstr.M{
					"stream_id": "new-stream",
					"new_field": "new-value",
				},
			}, true, false)

			event := &beat.Event{
				Meta: mapstr.M{
					"stream_id": "existing-stream",
				},
				Fields: mapstr.M{"message": "hello"},
			}

			result, err := p.Run(event)
			require.NoError(t, err)

			// stream_id should NOT be overwritten
			v, err := result.GetValue("@metadata.stream_id")
			require.NoError(t, err)
			assert.Equal(t, "existing-stream", v)

			// new_field should be added
			v, err = result.GetValue("@metadata.new_field")
			require.NoError(t, err)
			assert.Equal(t, "new-value", v)
		})

		t.Run("nil meta on event", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"@metadata": mapstr.M{"key": "value"},
			}, true, true)

			event := &beat.Event{
				Meta:   nil,
				Fields: mapstr.M{"message": "hello"},
			}

			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result.Meta)

			v, err := result.GetValue("@metadata.key")
			require.NoError(t, err)
			assert.Equal(t, "value", v)
		})
	})

	t.Run("metadata path - @metadata plus regular fields", func(t *testing.T) {
		t.Run("combined metadata and fields", func(t *testing.T) {
			p := NewAddFields(mapstr.M{
				"@metadata": mapstr.M{"input_id": "test-input"},
				"event":     mapstr.M{"dataset": "system.syslog"},
			}, true, true)

			event := &beat.Event{
				Fields: mapstr.M{"message": "hello"},
			}

			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Check metadata
			v, err := result.GetValue("@metadata.input_id")
			require.NoError(t, err)
			assert.Equal(t, "test-input", v)

			// Check fields
			v, err = result.GetValue("event.dataset")
			require.NoError(t, err)
			assert.Equal(t, "system.syslog", v)

			// Original fields preserved
			v, err = result.GetValue("message")
			require.NoError(t, err)
			assert.Equal(t, "hello", v)
		})
	})

	t.Run("slow path - @timestamp", func(t *testing.T) {
		t.Run("timestamp overwrite", func(t *testing.T) {
			now := time.Now()
			newTs := now.Add(time.Hour)

			p := NewAddFields(mapstr.M{
				"@timestamp": newTs,
				"agent":      mapstr.M{"id": "123"},
			}, true, true)

			event := &beat.Event{
				Timestamp: now,
				Fields:    mapstr.M{"message": "hello"},
			}

			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Timestamp should be updated
			assert.Equal(t, newTs, result.Timestamp)

			// Fields should also be updated
			v, err := result.GetValue("agent.id")
			require.NoError(t, err)
			assert.Equal(t, "123", v)
		})

		t.Run("timestamp no-overwrite", func(t *testing.T) {
			now := time.Now()
			newTs := now.Add(time.Hour)

			p := NewAddFields(mapstr.M{
				"@timestamp": newTs,
			}, true, false)

			event := &beat.Event{
				Timestamp: now,
				Fields:    mapstr.M{"message": "hello"},
			}

			result, err := p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Timestamp should NOT be overwritten in no-overwrite mode
			assert.Equal(t, now, result.Timestamp)
		})

		t.Run("timestamp with metadata and fields", func(t *testing.T) {
			now := time.Now()
			newTs := now.Add(time.Hour)

			p := NewAddFields(mapstr.M{
				"@timestamp": newTs,
				"@metadata":  mapstr.M{"key": "value"},
				"field":      "data",
			}, true, true)

			event := &beat.Event{
				Timestamp: now,
				Fields:    mapstr.M{},
			}

			result, err := p.Run(event)
			require.NoError(t, err)

			assert.Equal(t, newTs, result.Timestamp)

			v, err := result.GetValue("@metadata.key")
			require.NoError(t, err)
			assert.Equal(t, "value", v)

			v, err = result.GetValue("field")
			require.NoError(t, err)
			assert.Equal(t, "data", v)
		})
	})

	t.Run("shared flag prevents mutation of cached fields", func(t *testing.T) {
		original := mapstr.M{
			"agent": mapstr.M{"id": "cached-id"},
		}
		p := NewAddFields(original, true, true)

		// Run the processor multiple times
		for i := 0; i < 10; i++ {
			event := &beat.Event{
				Fields: mapstr.M{"message": "hello"},
			}
			result, err := p.Run(event)
			require.NoError(t, err)

			// Mutate the result event's fields
			_, _ = result.Fields.Put("agent.id", "mutated-by-downstream")
		}

		// The original processor fields should be unchanged
		id, err := original.GetValue("agent.id")
		require.NoError(t, err)
		assert.Equal(t, "cached-id", id)
	})

	t.Run("shared flag prevents mutation of cached @metadata fields", func(t *testing.T) {
		original := mapstr.M{
			"@metadata": mapstr.M{"input_id": "cached-input"},
		}
		p := NewAddFields(original, true, true)

		for i := 0; i < 10; i++ {
			event := &beat.Event{
				Fields: mapstr.M{"message": "hello"},
			}
			result, err := p.Run(event)
			require.NoError(t, err)

			// Mutate the result event's meta
			result.Meta["input_id"] = "mutated"
		}

		// Original must be unchanged
		require.IsType(t, mapstr.M{}, original["@metadata"])
		meta, _ := original["@metadata"].(mapstr.M)
		assert.Equal(t, "cached-input", meta["input_id"])
	})

	t.Run("input map not mutated by deepUpdate", func(t *testing.T) {
		// This verifies the contract that event.deepUpdate's delete/defer
		// restores the input map. Our fast path should also preserve this.
		fields := mapstr.M{
			"@metadata":  mapstr.M{"key": "value"},
			"@timestamp": time.Now(),
			"regular":    "field",
		}

		// Make a copy to compare against
		fieldsCopy := fields.Clone()

		p := NewAddFields(fields, false, true)
		event := &beat.Event{Fields: mapstr.M{}}
		_, err := p.Run(event)
		require.NoError(t, err)

		// The input fields map must be identical after Run
		assert.Equal(t, fieldsCopy, fields,
			"input map was mutated by Run()")
	})

	t.Run("processor chain produces correct cumulative result", func(t *testing.T) {
		// Simulates the exact elastic agent processor chain from generate.go
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

		event := &beat.Event{
			Timestamp: time.Now(),
			Fields: mapstr.M{
				"message": "test log message",
				"host":    mapstr.M{"name": "testhost"},
				"agent":   mapstr.M{"type": "filebeat"},
			},
		}

		var err error
		for _, p := range processors {
			event, err = p.Run(event)
			require.NoError(t, err)
			require.NotNil(t, event)
		}

		// Verify all fields were set correctly
		v, err := event.GetValue("elastic_agent.id")
		require.NoError(t, err)
		assert.Equal(t, "agent-uuid-1234", v)

		v, err = event.GetValue("elastic_agent.version")
		require.NoError(t, err)
		assert.Equal(t, "8.12.0", v)

		v, err = event.GetValue("agent.id")
		require.NoError(t, err)
		assert.Equal(t, "agent-uuid-1234", v)

		// agent.type should still exist (DeepUpdate merges, doesn't replace parent)
		v, err = event.GetValue("agent.type")
		require.NoError(t, err)
		assert.Equal(t, "filebeat", v)

		v, err = event.GetValue("@metadata.input_id")
		require.NoError(t, err)
		assert.Equal(t, "logfile-system-default", v)

		v, err = event.GetValue("@metadata.stream_id")
		require.NoError(t, err)
		assert.Equal(t, "stream-uuid-5678", v)

		v, err = event.GetValue("data_stream.type")
		require.NoError(t, err)
		assert.Equal(t, "logs", v)

		v, err = event.GetValue("data_stream.dataset")
		require.NoError(t, err)
		assert.Equal(t, "system.syslog", v)

		v, err = event.GetValue("event.dataset")
		require.NoError(t, err)
		assert.Equal(t, "system.syslog", v)

		// Original fields preserved
		v, err = event.GetValue("message")
		require.NoError(t, err)
		assert.Equal(t, "test log message", v)

		v, err = event.GetValue("host.name")
		require.NoError(t, err)
		assert.Equal(t, "testhost", v)
	})
}

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
