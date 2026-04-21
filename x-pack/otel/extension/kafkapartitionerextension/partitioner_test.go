// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kafkapartitionerextension

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getPartitioner(t *testing.T, cfg map[string]any) (kgo.Partitioner, kgo.TopicPartitioner) {
	t.Helper()

	log := logptest.NewTestingLogger(t, "")
	p, err := makePartitioner(log, cfg)
	require.NoError(t, err)

	return p, p.ForTopic("test-topic")
}

func makeJSONRecord(t *testing.T, fields map[string]any) *kgo.Record {
	t.Helper()
	b, err := json.Marshal(fields)
	require.NoError(t, err)
	return &kgo.Record{Value: b}
}
func TestUnmarshalLogs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, m mapstr.M)
	}{
		{
			name:        "valid map",
			input:       `{"key":"value","num":42}`,
			expectError: false,
			validate: func(t *testing.T, m mapstr.M) {
				assert.Equal(t, mapstr.M{"key": "value", "num": float64(42)}, m)
			},
		},
		{
			name:        "invalid json",
			input:       `not json`,
			expectError: true,
		},
		{
			name:        "json array",
			input:       `["a","b"]`,
			expectError: true,
			errorMsg:    "shoud be a map",
		},
		{
			name:        "json string",
			input:       `"just a string"`,
			expectError: true,
		},
		{
			name:        "json number",
			input:       `42`,
			expectError: true,
		},
		{
			name:        "json null",
			input:       `null`,
			expectError: true,
		},
		{
			name:        "nested fields",
			input:       `{"service":{"name":"web"},"host":"myhost"}`,
			expectError: false,
			validate: func(t *testing.T, m mapstr.M) {
				v, err := m.GetValue("service.name")
				require.NoError(t, err)
				assert.Equal(t, "web", v)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := unmarshalLogs([]byte(tc.input))

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
				return
			}

			require.NoError(t, err)

			if tc.validate != nil {
				tc.validate(t, m)
			}
		})
	}
}

func TestMakePartitioner(t *testing.T) {
	tests := []struct {
		name        string
		cfg         map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty config (nil)",
			cfg:         nil,
			expectError: false,
		},
		{
			name:        "empty map",
			cfg:         map[string]any{},
			expectError: false,
		},
		{
			name: "too many partitioners",
			cfg: map[string]any{
				"random":      map[string]any{},
				"round_robin": map[string]any{},
			},
			expectError: true,
			errorMsg:    "too many partitioners",
		},
		{
			name: "unknown partitioner",
			cfg: map[string]any{
				"bogus_partitioner": map[string]any{},
			},
			expectError: true,
			errorMsg:    "unknown kafka partition mode",
		},
		{
			name: "random config",
			cfg: map[string]any{
				"random": map[string]any{
					"group_events": 1,
				},
			},
			expectError: false,
		},
		{
			name: "round robin config",
			cfg: map[string]any{
				"round_robin": map[string]any{
					"group_events": 2,
				},
			},
			expectError: false,
		},
		{
			name: "hash config",
			cfg: map[string]any{
				"hash": map[string]any{},
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log := logptest.NewTestingLogger(t, "")

			p, err := makePartitioner(log, tc.cfg)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, p)
		})
	}
}

func TestRandomPartitioner(t *testing.T) {

	t.Run("group events", func(t *testing.T) {
		const groupSize = 3
		const numPartitions = 10

		_, tp := getPartitioner(t, map[string]any{
			"random": map[string]any{"group_events": groupSize},
		})

		first := tp.Partition(makeJSONRecord(t, map[string]any{"i": 0}), numPartitions)

		for i := 1; i < groupSize; i++ {
			got := tp.Partition(makeJSONRecord(t, map[string]any{"i": i}), numPartitions)
			assert.Equal(t, first, got, "records in the same group must share a partition")
		}
	})

	t.Run("valid range", func(t *testing.T) {
		const numPartitions = 8

		_, tp := getPartitioner(t, map[string]any{
			"random": map[string]any{},
		})

		for i := 0; i < 50; i++ {
			part := tp.Partition(makeJSONRecord(t, map[string]any{"i": i}), numPartitions)
			assert.GreaterOrEqual(t, part, 0)
			assert.Less(t, part, numPartitions)
		}
	})
}
func TestRoundRobinPartitioner(t *testing.T) {
	t.Run("cycles through all partitions", func(t *testing.T) {
		const numPartitions = 4

		_, tp := getPartitioner(t, map[string]any{
			"round_robin": map[string]any{"group_events": 1},
		})

		seen := make(map[int]bool)

		for i := 0; i < numPartitions*4; i++ {
			part := tp.Partition(makeJSONRecord(t, map[string]any{"i": i}), numPartitions)

			assert.GreaterOrEqual(t, part, 0)
			assert.Less(t, part, numPartitions)

			seen[part] = true
		}

		assert.Len(t, seen, numPartitions, "round-robin must visit every partition")
	})

	t.Run("group events", func(t *testing.T) {
		const groupSize = 2
		const numPartitions = 4

		_, tp := getPartitioner(t, map[string]any{
			"round_robin": map[string]any{"group_events": groupSize},
		})

		// First group
		p1a := tp.Partition(makeJSONRecord(t, map[string]any{}), numPartitions)
		p1b := tp.Partition(makeJSONRecord(t, map[string]any{}), numPartitions)
		assert.Equal(t, p1a, p1b, "records in the same group must share a partition")

		// Second group
		p2a := tp.Partition(makeJSONRecord(t, map[string]any{}), numPartitions)
		p2b := tp.Partition(makeJSONRecord(t, map[string]any{}), numPartitions)
		assert.Equal(t, p2a, p2b, "records in the same group must share a partition")
	})
}

func TestHashPartitioner(t *testing.T) {
	t.Run("nil key produces valid partition", func(t *testing.T) {
		const numPartitions = 8

		_, tp := getPartitioner(t, map[string]any{
			"hash": map[string]any{},
		})

		rec := &kgo.Record{Key: nil, Value: []byte(`{}`)}

		part := tp.Partition(rec, numPartitions)
		assert.GreaterOrEqual(t, part, 0)
		assert.Less(t, part, numPartitions)
	})

	t.Run("same key is consistent across partitioners", func(t *testing.T) {
		const numPartitions = 16

		log := logptest.NewTestingLogger(t, "")
		p, err := makePartitioner(log, map[string]any{"hash": map[string]any{}})
		require.NoError(t, err)

		tp1 := p.ForTopic("test-topic")
		tp2 := p.ForTopic("test-topic")

		rec := &kgo.Record{Key: []byte("stable-key"), Value: []byte(`{}`)}

		assert.Equal(t,
			tp1.Partition(rec, numPartitions),
			tp2.Partition(rec, numPartitions),
			"same key must always map to same partition",
		)
	})

	t.Run("different keys stay in valid range", func(t *testing.T) {
		const numPartitions = 32

		_, tp := getPartitioner(t, map[string]any{
			"hash": map[string]any{},
		})

		for _, key := range []string{"alpha", "beta", "gamma", "delta", "epsilon"} {
			rec := &kgo.Record{Key: []byte(key), Value: []byte(`{}`)}

			part := tp.Partition(rec, numPartitions)
			assert.GreaterOrEqual(t, part, 0, "key=%s", key)
			assert.Less(t, part, numPartitions, "key=%s", key)
		}
	})

	t.Run("field-based same value stable partition", func(t *testing.T) {
		const numPartitions = 16

		_, tp := getPartitioner(t, map[string]any{
			"hash": map[string]any{"hash": []string{"user_id"}},
		})

		rec1 := makeJSONRecord(t, map[string]any{"user_id": "alice", "extra": "a"})
		rec2 := makeJSONRecord(t, map[string]any{"user_id": "alice", "extra": "b"})

		assert.Equal(t,
			tp.Partition(rec1, numPartitions),
			tp.Partition(rec2, numPartitions),
			"same field value must produce same partition",
		)
	})

	t.Run("field-based different values spread", func(t *testing.T) {
		const numPartitions = 128

		_, tp := getPartitioner(t, map[string]any{
			"hash": map[string]any{"hash": []string{"user_id"}},
		})

		seen := map[int]bool{}

		for _, user := range []string{"alice", "bob", "charlie", "dave", "eve"} {
			rec := makeJSONRecord(t, map[string]any{"user_id": user})
			seen[tp.Partition(rec, numPartitions)] = true
		}

		assert.Greater(t, len(seen), 1, "values should spread across partitions")
	})

	t.Run("missing field with random fallback", func(t *testing.T) {
		const numPartitions = 8

		_, tp := getPartitioner(t, map[string]any{
			"hash": map[string]any{
				"hash":   []string{"nonexistent"},
				"random": true,
			},
		})

		rec := makeJSONRecord(t, map[string]any{"other_field": "value"})

		part := tp.Partition(rec, numPartitions)
		assert.GreaterOrEqual(t, part, 0)
		assert.Less(t, part, numPartitions)
	})

	t.Run("multiple fields stable", func(t *testing.T) {
		const numPartitions = 16

		_, tp := getPartitioner(t, map[string]any{
			"hash": map[string]any{"hash": []string{"service", "host"}},
		})

		rec1 := makeJSONRecord(t, map[string]any{"service": "web", "host": "host1"})
		rec2 := makeJSONRecord(t, map[string]any{"service": "web", "host": "host1"})

		assert.Equal(t,
			tp.Partition(rec1, numPartitions),
			tp.Partition(rec2, numPartitions),
		)
	})

	t.Run("field-based valid range", func(t *testing.T) {
		const numPartitions = 32

		_, tp := getPartitioner(t, map[string]any{
			"hash": map[string]any{"hash": []string{"service"}},
		})

		for _, svc := range []string{"auth", "payment", "inventory", "notification", "search"} {
			rec := makeJSONRecord(t, map[string]any{"service": svc})

			part := tp.Partition(rec, numPartitions)
			assert.GreaterOrEqual(t, part, 0, "service=%s", svc)
			assert.Less(t, part, numPartitions, "service=%s", svc)
		}
	})
}

func TestHash2Partition(t *testing.T) {
	tests := []struct {
		name          string
		hash          uint32
		numPartitions int
		expected      int
	}{
		{"zero hash", 0, 10, 0},
		{"hash equals numPartitions", 10, 10, 0},
		{"hash mod", 11, 10, 1},
		{"high bit masked", 0x80000000, 10, 0},
		{"all bits set", 0xFFFFFFFF, 10, (0x7FFFFFFF) % 10},
		{"single partition", 42, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hash2Partition(tt.hash, tt.numPartitions)
			assert.Equal(t, tt.expected, got)
		})
	}
}
