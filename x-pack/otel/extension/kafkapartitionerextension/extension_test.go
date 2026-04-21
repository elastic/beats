// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kafkapartitionerextension

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.uber.org/zap/zaptest"
)

func newTestPartitioner(t *testing.T, partitionerConfig map[string]any) *kafkaPartitioner {
	t.Helper()
	settings := extensiontest.NewNopSettings(component.MustNewType("kafkapartitioner"))
	settings.Logger = zaptest.NewLogger(t)
	ext, err := newExtension(context.Background(), settings, &Config{
		PartitionerConfig: partitionerConfig,
	})
	require.NoError(t, err)
	kp, ok := ext.(*kafkaPartitioner)
	require.True(t, ok)
	return kp
}

func TestExtension_StartShutdown(t *testing.T) {
	kp := newTestPartitioner(t, nil)
	host := componenttest.NewNopHost()
	require.NoError(t, kp.Start(context.Background(), host))
	require.NoError(t, kp.Shutdown(context.Background()))
}

func TestGetPartitioner(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]any
		expectedErr string
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name:   "empty config",
			config: map[string]any{},
		},
		{
			name: "random config",
			config: map[string]any{
				"random": map[string]any{"group_events": 2},
			},
		},
		{
			name: "round robin config",
			config: map[string]any{
				"round_robin": map[string]any{"group_events": 1},
			},
		},
		{
			name: "hash config no fields",
			config: map[string]any{
				"hash": map[string]any{},
			},
		},
		{
			name: "hash config with fields",
			config: map[string]any{
				"hash": map[string]any{
					"hash": []string{"user_id", "service"},
				},
			},
		},
		{
			name: "invalid config - multiple",
			config: map[string]any{
				"random":      map[string]any{},
				"round_robin": map[string]any{},
			},
			expectedErr: "too many partitioners configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kp := newTestPartitioner(t, tt.config)

			err := kp.Start(context.Background(), componenttest.NewNopHost())
			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)

			p := kp.GetPartitioner()
			require.NotNil(t, p)

			require.NoError(t, kp.Shutdown(context.Background()))
		})
	}
}

// TestE2E_HashPartitioner_MapMessages exercises the complete extension
// lifecycle and verifies that field-based partitioning on JSON map messages
// is deterministic.
func TestE2E_HashPartitioner_MapMessages(t *testing.T) {
	kp := newTestPartitioner(t, map[string]any{
		"hash": map[string]any{"hash": []string{"trace_id"}},
	})

	host := componenttest.NewNopHost()
	require.NoError(t, kp.Start(context.Background(), host))
	t.Cleanup(func() { _ = kp.Shutdown(context.Background()) })

	p := kp.GetPartitioner()
	require.NotNil(t, p)

	const numPartitions = 16
	tp := p.ForTopic("traces")

	// Same trace_id must always land on the same partition.
	for _, traceID := range []string{"trace-aaa", "trace-bbb", "trace-ccc"} {
		rec1 := makeJSONRecord(t, map[string]any{"trace_id": traceID, "span": 1})
		rec2 := makeJSONRecord(t, map[string]any{"trace_id": traceID, "span": 2})
		p1 := tp.Partition(rec1, numPartitions)
		p2 := tp.Partition(rec2, numPartitions)
		assert.Equal(t, p1, p2,
			"trace_id=%q: both records must land on the same partition", traceID)
		assert.GreaterOrEqual(t, p1, 0)
		assert.Less(t, p1, numPartitions)
	}
}

// TestE2E_RandomPartitioner_MapMessages verifies that the random partitioner
// returns valid partition indices for JSON map messages.
func TestE2E_RandomPartitioner_MapMessages(t *testing.T) {
	kp := newTestPartitioner(t, map[string]any{
		"random": map[string]any{"group_events": 1},
	})

	host := componenttest.NewNopHost()
	require.NoError(t, kp.Start(context.Background(), host))
	t.Cleanup(func() { _ = kp.Shutdown(context.Background()) })

	p := kp.GetPartitioner()
	const numPartitions = 12
	tp := p.ForTopic("events")

	for i := 0; i < 30; i++ {
		rec := makeJSONRecord(t, map[string]any{"event_id": fmt.Sprintf("evt-%d", i)})
		part := tp.Partition(rec, numPartitions)
		assert.GreaterOrEqual(t, part, 0, "event %d: partition must be ≥ 0", i)
		assert.Less(t, part, numPartitions, "event %d: partition must be < %d", i, numPartitions)
	}
}

// TestE2E_RoundRobinPartitioner_MapMessages verifies that the round-robin
// partitioner cycles through all partitions when given JSON map messages.
func TestE2E_RoundRobinPartitioner_MapMessages(t *testing.T) {
	kp := newTestPartitioner(t, map[string]any{
		"round_robin": map[string]any{"group_events": 1},
	})

	host := componenttest.NewNopHost()
	require.NoError(t, kp.Start(context.Background(), host))
	t.Cleanup(func() { _ = kp.Shutdown(context.Background()) })

	p := kp.GetPartitioner()
	const numPartitions = 5
	tp := p.ForTopic("logs")

	seen := make(map[int]bool)
	for i := 0; i < numPartitions*3; i++ {
		rec := makeJSONRecord(t, map[string]any{"log_line": fmt.Sprintf("line %d", i)})
		part := tp.Partition(rec, numPartitions)
		assert.GreaterOrEqual(t, part, 0)
		assert.Less(t, part, numPartitions)
		seen[part] = true
	}
	assert.Len(t, seen, numPartitions,
		"round-robin must visit every partition over sufficient messages")
}

// TestE2E_MultiFieldHash_MapMessages verifies multi-field hashing with
// real JSON map messages across a full extension lifecycle.
func TestE2E_MultiFieldHash_MapMessages(t *testing.T) {
	kp := newTestPartitioner(t, map[string]any{
		"hash": map[string]any{"hash": []string{"service", "environment"}},
	})

	host := componenttest.NewNopHost()
	require.NoError(t, kp.Start(context.Background(), host))
	t.Cleanup(func() { _ = kp.Shutdown(context.Background()) })

	p := kp.GetPartitioner()
	const numPartitions = 20
	tp := p.ForTopic("metrics")

	type serviceEnv struct{ service, env string }
	combos := []serviceEnv{
		{"frontend", "production"},
		{"backend", "staging"},
		{"database", "production"},
		{"cache", "staging"},
	}

	partitionOf := make(map[serviceEnv]int)
	for _, se := range combos {
		rec := makeJSONRecord(t, map[string]any{
			"service":     se.service,
			"environment": se.env,
			"timestamp":   "2026-01-01T00:00:00Z",
		})
		part := tp.Partition(rec, numPartitions)
		assert.GreaterOrEqual(t, part, 0)
		assert.Less(t, part, numPartitions)
		partitionOf[se] = part
	}

	// Repeating with different unrelated fields must yield the same partition.
	for _, se := range combos {
		rec := makeJSONRecord(t, map[string]any{
			"service":     se.service,
			"environment": se.env,
			"extra":       "ignored-by-hash",
		})
		got := tp.Partition(rec, numPartitions)
		assert.Equal(t, partitionOf[se], got,
			"service=%s env=%s: second pass must land on the same partition",
			se.service, se.env)
	}
}
