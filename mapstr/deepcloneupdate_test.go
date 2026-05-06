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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepCloneUpdateEquivalence(t *testing.T) {
	src := M{
		"cloud": M{
			"provider": "aws",
			"region":   "us-east-1",
			"account":  M{"id": "123"},
		},
		"agent": M{"id": "agent-123"},
	}

	// Clone+DeepUpdate
	dst1 := M{"existing": "value"}
	dst1.DeepUpdate(src.Clone())

	// DeepCloneUpdate
	dst2 := M{"existing": "value"}
	dst2.DeepCloneUpdate(src)

	assert.Equal(t, dst1, dst2)
}

func TestDeepCloneUpdateNoAliasing(t *testing.T) {
	src := M{
		"cloud": M{
			"provider": "aws",
			"region":   "us-east-1",
		},
	}
	srcCopy := src.Clone()

	dst := M{}
	dst.DeepCloneUpdate(src)

	// Mutate the destination.
	cloud, err := dst.GetValue("cloud")
	require.NoError(t, err)
	cloudMap, ok := cloud.(M)
	require.True(t, ok)
	cloudMap["provider"] = "MUTATED"

	// Source must be unchanged.
	assert.Equal(t, srcCopy, src, "source must not be affected by mutations to destination")
}

func TestDeepCloneUpdateIndependentEvents(t *testing.T) {
	shared := M{
		"agent": M{"id": "agent-123", "version": "8.12.0"},
	}

	event1 := M{"message": "event1"}
	event1.DeepCloneUpdate(shared)

	event2 := M{"message": "event2"}
	event2.DeepCloneUpdate(shared)

	// Mutate event1.
	agent1, _ := event1.GetValue("agent")
	agent1Map, ok := agent1.(M)
	require.True(t, ok)
	agent1Map["id"] = "MUTATED"

	// event2 must be unaffected.
	v, _ := event2.GetValue("agent.id")
	assert.Equal(t, "agent-123", v)
}

func TestDeepCloneUpdateMergesExistingMaps(t *testing.T) {
	dst := M{
		"agent": M{"type": "filebeat"},
	}
	src := M{
		"agent": M{"id": "agent-123"},
	}

	dst.DeepCloneUpdate(src)

	// Both keys should exist.
	v, _ := dst.GetValue("agent.type")
	assert.Equal(t, "filebeat", v)

	v, _ = dst.GetValue("agent.id")
	assert.Equal(t, "agent-123", v)
}

func TestDeepCloneUpdateOverwrites(t *testing.T) {
	dst := M{"key": "old"}
	src := M{"key": "new"}

	dst.DeepCloneUpdate(src)
	assert.Equal(t, "new", dst["key"])
}

func TestDeepCloneUpdateDeepNesting(t *testing.T) {
	src := M{
		"a": M{"b": M{"c": M{"d": "deep"}}},
	}
	srcCopy := src.Clone()

	dst := M{}
	dst.DeepCloneUpdate(src)

	// Mutate deep in dst.
	a, err := dst.GetValue("a")
	require.NoError(t, err)
	aMap, ok := a.(M)
	require.True(t, ok)
	bVal, err := aMap.GetValue("b")
	require.NoError(t, err)
	bMap, ok := bVal.(M)
	require.True(t, ok)
	cVal, err := bMap.GetValue("c")
	require.NoError(t, err)
	cMap, ok := cVal.(M)
	require.True(t, ok)
	cMap["d"] = "MUTATED"

	// Source must be unchanged.
	assert.Equal(t, srcCopy, src)
}

// --- DeepCloneUpdateNoOverwrite ---

func TestDeepCloneUpdateNoOverwriteSkipsExisting(t *testing.T) {
	dst := M{
		"agent": M{"type": "filebeat", "id": "existing"},
	}
	src := M{
		"agent": M{"id": "new-id", "version": "8.12.0"},
		"ecs":   M{"version": "8.0.0"},
	}

	dst.DeepCloneUpdateNoOverwrite(src)

	// Existing key not overwritten.
	v, _ := dst.GetValue("agent.id")
	assert.Equal(t, "existing", v)

	// Existing sibling preserved.
	v, _ = dst.GetValue("agent.type")
	assert.Equal(t, "filebeat", v)

	// New nested key added.
	v, _ = dst.GetValue("agent.version")
	assert.Equal(t, "8.12.0", v)

	// New top-level key added.
	v, _ = dst.GetValue("ecs.version")
	assert.Equal(t, "8.0.0", v)
}

func TestDeepCloneUpdateNoOverwriteNoAliasing(t *testing.T) {
	src := M{"cloud": M{"provider": "aws"}}
	srcCopy := src.Clone()

	dst := M{}
	dst.DeepCloneUpdateNoOverwrite(src)

	cloud, _ := dst.GetValue("cloud")
	cloudMap, ok := cloud.(M)
	require.True(t, ok)
	cloudMap["provider"] = "MUTATED"

	assert.Equal(t, srcCopy, src, "source must not be affected")
}

// TestDeepCloneUpdateNoOverwriteDeepNested is the specific regression test for
// the scenario CodeRabbit flagged as critical: an optimization in add_fields
// short-circuited on top-level key presence, silently dropping enrichment for
// partially-populated nested objects. This test verifies that
// DeepCloneUpdateNoOverwrite descends into existing sub-maps and adds missing
// leaves at any depth rather than stopping at the first level.
func TestDeepCloneUpdateNoOverwriteDeepNested(t *testing.T) {
	// Three levels deep: dst has host.os.type, src has host.os.version.
	// The correct behavior is to add host.os.version without touching host.os.type.
	dst := M{
		"host": M{
			"os": M{"type": "linux"},
		},
	}
	src := M{
		"host": M{
			"os": M{"version": "22.04", "type": "windows"},
		},
	}

	dst.DeepCloneUpdateNoOverwrite(src)

	// Existing leaf at depth 3 must not be overwritten.
	v, err := dst.GetValue("host.os.type")
	require.NoError(t, err)
	assert.Equal(t, "linux", v)

	// Missing leaf at depth 3 must be added.
	v, err = dst.GetValue("host.os.version")
	require.NoError(t, err)
	assert.Equal(t, "22.04", v)
}

// TestDeepCloneUpdateNoOverwriteNewSubMapNoAliasing verifies that when
// DeepCloneUpdateNoOverwrite adds a new sub-map inside an existing sub-map,
// the newly inserted map is a fresh copy and not aliased to the source.
func TestDeepCloneUpdateNoOverwriteNewSubMapNoAliasing(t *testing.T) {
	src := M{
		"agent": M{
			"id":      "existing",
			"details": M{"version": "8.12.0"},
		},
	}
	srcCopy := src.Clone()

	dst := M{"agent": M{"id": "existing"}}
	dst.DeepCloneUpdateNoOverwrite(src)

	// Mutate the newly-inserted sub-map in the destination.
	details, err := dst.GetValue("agent.details")
	require.NoError(t, err)
	detailsMap, ok := details.(M)
	require.True(t, ok)
	detailsMap["version"] = "MUTATED"

	// Source must be unchanged.
	assert.Equal(t, srcCopy, src, "source must not be affected by mutations to destination")
}

func TestDeepCloneUpdateNoOverwriteEquivalence(t *testing.T) {
	src := M{
		"agent": M{"id": "new", "version": "8.12.0"},
		"ecs":   M{"version": "8.0.0"},
	}

	dst1 := M{"agent": M{"id": "existing"}}
	dst1.DeepUpdateNoOverwrite(src.Clone())

	dst2 := M{"agent": M{"id": "existing"}}
	dst2.DeepCloneUpdateNoOverwrite(src)

	assert.Equal(t, dst1, dst2)
}

// TestDeepCloneUpdateMapStringInterfaceDst is the regression test for the bug
// where DeepCloneUpdate overwrote a map[string]interface{} destination subtree
// instead of merging into it. This caused event fields like @timestamp,
// event.start, and process.start to be silently dropped or replaced with
// empty maps when processors merged their metadata into events whose fields
// contained map[string]interface{} values (e.g. decoded from JSON).
func TestDeepCloneUpdateMapStringInterfaceDst(t *testing.T) {
	// Simulates an event where "event" subtree is map[string]interface{} (as
	// decoded from JSON/wire), and a processor adds "event.dataset".
	dst := M{
		"event": map[string]interface{}{
			"start": "2024-01-01T00:00:00Z",
			"end":   "2024-01-01T01:00:00Z",
		},
	}
	src := M{
		"event": M{"dataset": "mydata"},
	}

	dst.DeepCloneUpdate(src)

	// Existing fields in the map[string]interface{} subtree must be preserved.
	v, err := dst.GetValue("event.start")
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00Z", v)

	v, err = dst.GetValue("event.end")
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T01:00:00Z", v)

	// New field from source must be added.
	v, err = dst.GetValue("event.dataset")
	require.NoError(t, err)
	assert.Equal(t, "mydata", v)
}

func TestDeepCloneUpdateNoOverwriteMapStringInterfaceDst(t *testing.T) {
	dst := M{
		"process": map[string]interface{}{
			"start": "2024-01-01T00:00:00Z",
			"pid":   1234,
		},
	}
	src := M{
		"process": M{"start": "SHOULD-NOT-OVERWRITE", "name": "myapp"},
	}

	dst.DeepCloneUpdateNoOverwrite(src)

	// Existing field must not be overwritten.
	v, err := dst.GetValue("process.start")
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00Z", v)

	// Existing field from original map preserved.
	v, err = dst.GetValue("process.pid")
	require.NoError(t, err)
	assert.Equal(t, 1234, v)

	// New field added.
	v, err = dst.GetValue("process.name")
	require.NoError(t, err)
	assert.Equal(t, "myapp", v)
}

// TestDeepCloneUpdateMapStringInterfaceDstEquivalence is the oracle test for
// the map[string]interface{} destination fix: verifies DeepCloneUpdate
// produces bit-for-bit the same result as DeepUpdate(src.Clone()) when the
// destination tree contains map[string]interface{} nodes (as commonly occurs
// with JSON-decoded event data).
func TestDeepCloneUpdateMapStringInterfaceDstEquivalence(t *testing.T) {
	makeDst := func() M {
		return M{
			"event": map[string]interface{}{
				"start": "2024-01-01T00:00:00Z",
				"end":   "2024-01-01T01:00:00Z",
			},
			"process": map[string]interface{}{
				"start": "2024-01-01T00:00:00Z",
				"pid":   1234,
			},
			"host": M{"name": "server1"},
		}
	}
	src := M{
		"event":   M{"dataset": "mydata"},
		"process": M{"name": "myapp"},
		"host":    M{"os": M{"type": "linux"}},
	}

	dst1 := makeDst()
	dst1.DeepUpdate(src.Clone())

	dst2 := makeDst()
	dst2.DeepCloneUpdate(src)

	assert.Equal(t, dst1, dst2, "DeepCloneUpdate must be equivalent to DeepUpdate(src.Clone()) for map[string]interface{} destinations")
}

func TestDeepCloneUpdateNoOverwriteMapStringInterfaceDstEquivalence(t *testing.T) {
	makeDst := func() M {
		return M{
			"event": map[string]interface{}{
				"start":   "2024-01-01T00:00:00Z",
				"dataset": "original",
			},
			"process": map[string]interface{}{
				"start": "2024-01-01T00:00:00Z",
				"pid":   1234,
			},
		}
	}
	src := M{
		"event":   M{"dataset": "new-dataset", "end": "2024-01-01T01:00:00Z"},
		"process": M{"start": "SHOULD-NOT-OVERWRITE", "name": "myapp"},
	}

	dst1 := makeDst()
	dst1.DeepUpdateNoOverwrite(src.Clone())

	dst2 := makeDst()
	dst2.DeepCloneUpdateNoOverwrite(src)

	assert.Equal(t, dst1, dst2, "DeepCloneUpdateNoOverwrite must be equivalent to DeepUpdateNoOverwrite(src.Clone()) for map[string]interface{} destinations")
}

func TestDeepCloneUpdateNilSource(t *testing.T) {
	dst := M{"key": "value"}
	dst.DeepCloneUpdate(nil)
	assert.Equal(t, M{"key": "value"}, dst)
}

func TestDeepCloneUpdateEmptySource(t *testing.T) {
	dst := M{"key": "value"}
	dst.DeepCloneUpdate(M{})
	assert.Equal(t, M{"key": "value"}, dst)
}

// TestDeepCloneUpdateNoOverwriteNilMapDst verifies that when the destination
// holds a nil map[string]interface{} value, DeepCloneUpdateNoOverwrite does not
// panic and instead replaces it with a fresh copy of the source map.
func TestDeepCloneUpdateNoOverwriteNilMapDst(t *testing.T) {
	var nilMap map[string]interface{}
	dst := M{"host": nilMap}
	src := M{"host": M{"name": "server1"}}

	assert.NotPanics(t, func() {
		dst.DeepCloneUpdateNoOverwrite(src)
	})

	v, err := dst.GetValue("host.name")
	require.NoError(t, err)
	assert.Equal(t, "server1", v)
}

// TestDeepCloneUpdateNilDsts verifies that DeepCloneUpdate and
// DeepCloneUpdateNoOverwrite never panic on nil destination sub-maps —
// neither for typed-nil map[string]interface{} nor for nil M — and that
// the result matches the DeepUpdate(src.Clone()) oracle in every case.
func TestDeepCloneUpdateNilDsts(t *testing.T) {
	cases := []struct {
		name string
		dst  func() M
	}{
		{
			name: "nil map[string]interface{}",
			dst: func() M {
				var nilMap map[string]interface{}
				return M{"host": nilMap}
			},
		},
		{
			name: "nil M",
			dst: func() M {
				var nilM M
				return M{"host": nilM}
			},
		},
	}

	src := M{"host": M{"name": "server1"}}

	for _, tc := range cases {
		t.Run(tc.name+"/DeepCloneUpdate", func(t *testing.T) {
			dst := tc.dst()
			oracle := tc.dst()
			oracle.DeepUpdate(src.Clone())

			assert.NotPanics(t, func() {
				dst.DeepCloneUpdate(src)
			})
			assert.Equal(t, oracle, dst)
		})

		t.Run(tc.name+"/DeepCloneUpdateNoOverwrite", func(t *testing.T) {
			dst := tc.dst()
			oracle := tc.dst()
			oracle.DeepUpdateNoOverwrite(src.Clone())

			assert.NotPanics(t, func() {
				dst.DeepCloneUpdateNoOverwrite(src)
			})
			assert.Equal(t, oracle, dst)
		})
	}
}

func TestDeepCloneUpdateMapStringInterface(t *testing.T) {
	// Test that map[string]interface{} values are handled.
	src := M{
		"data": map[string]interface{}{"key": "value"},
	}
	dst := M{}
	dst.DeepCloneUpdate(src)

	v, err := dst.GetValue("data.key")
	require.NoError(t, err)
	assert.Equal(t, "value", v)

	// Mutate dst and verify source is unaffected.
	_, _ = dst.Put("data.key", "MUTATED")

	srcDataVal, _ := src.GetValue("data.key")
	assert.Equal(t, "value", srcDataVal, "source must not be affected")
}

// --- Benchmarks ---

var (
	benchSinkM  M
	benchShared = []M{
		{"elastic_agent": M{"id": "agent-uuid", "snapshot": false, "version": "8.12.0"}},
		{"agent": M{"id": "agent-uuid"}},
		{"data_stream": M{"type": "logs", "dataset": "system.syslog", "namespace": "default"}},
		{"event": M{"dataset": "system.syslog"}},
		{"cloud": M{
			"provider": "aws", "region": "us-east-1", "availability_zone": "us-east-1a",
			"account": M{"id": "123456789012"}, "instance": M{"id": "i-0abcdef"},
			"machine": M{"type": "m5.xlarge"}, "service": M{"name": "EC2"},
		}},
	}
)

func BenchmarkSingleMerge(b *testing.B) {
	src := M{
		"elastic_agent": M{"id": "agent-uuid", "snapshot": false, "version": "8.12.0"},
		"cloud": M{
			"provider": "aws", "region": "us-east-1",
			"account": M{"id": "123"}, "instance": M{"id": "i-abc"},
			"machine": M{"type": "m5.xlarge"}, "service": M{"name": "EC2"},
		},
	}

	b.Run("clone_and_deep_update", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst := M{"message": "test", "host": M{"name": "server1"}}
			dst.DeepUpdate(src.Clone())
			benchSinkM = dst
		}
	})

	b.Run("deep_copy_update", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst := M{"message": "test", "host": M{"name": "server1"}}
			dst.DeepCloneUpdate(src)
			benchSinkM = dst
		}
	})
}

// BenchmarkRealisticPipeline simulates a full elastic agent pipeline:
// 6 addFields (agent info, agent id, data stream, event dataset, 2 metadata),
// cloud metadata, host metadata, and kubernetes metadata.
func BenchmarkRealisticPipeline(b *testing.B) {
	hostMeta := M{
		"host": M{
			"name":         "prod-server-01.example.com",
			"hostname":     "prod-server-01",
			"architecture": "x86_64",
			"id":           "4CA0A3FC-4AEA-5CAE-85A3-C0AC3725AD24",
			"ip":           []string{"192.168.1.100", "10.0.0.1"},
			"mac":          []string{"00:11:22:33:44:55"},
			"os": M{
				"type":     "linux",
				"platform": "ubuntu",
				"name":     "Ubuntu",
				"family":   "debian",
				"version":  "22.04.3 LTS",
				"kernel":   "5.15.0-91-generic",
			},
		},
	}

	k8sMeta := M{
		"kubernetes": M{
			"pod": M{
				"name": "myapp-7b9d6c5f4-xk2j9",
				"uid":  "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			},
			"node":      M{"name": "node-1"},
			"namespace": "production",
			"container": M{
				"name":    "myapp",
				"image":   "myrepo/myapp:v2.1.0",
				"id":      "abc123def456",
				"runtime": "containerd",
			},
			"labels": M{
				"app":     "myapp",
				"version": "v2.1.0",
				"env":     "production",
			},
		},
	}

	allShared := make([]M, 0, len(benchShared)+2)
	allShared = append(allShared, benchShared...)
	allShared = append(allShared, hostMeta, k8sMeta)

	b.Run("clone_and_deep_update", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst := M{
				"message": "Mar 27 00:04:33 prod-server-01 myapp[12345]: request completed in 42ms",
				"agent":   M{"type": "filebeat"},
			}
			for _, src := range allShared {
				dst.DeepUpdate(src.Clone())
			}
			benchSinkM = dst
		}
	})

	b.Run("deep_copy_update", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst := M{
				"message": "Mar 27 00:04:33 prod-server-01 myapp[12345]: request completed in 42ms",
				"agent":   M{"type": "filebeat"},
			}
			for _, src := range allShared {
				dst.DeepCloneUpdate(src)
			}
			benchSinkM = dst
		}
	})
}

// BenchmarkHeavyPipeline simulates a pipeline with 20 processors, each
// adding their own shared metadata. This represents a heavily-configured
// deployment with many integrations.
func BenchmarkHeavyPipeline(b *testing.B) {
	var manyShared []M
	// 6 addFields (agent info)
	manyShared = append(manyShared, benchShared...)
	// Add 15 more processors simulating integration-specific metadata
	for i := 0; i < 15; i++ {
		manyShared = append(manyShared, M{
			fmt.Sprintf("integration_%d", i): M{
				"name":    fmt.Sprintf("integration-%d", i),
				"version": "1.0.0",
				"config": M{
					"enabled": true,
					"level":   "info",
				},
			},
		})
	}

	b.Run("clone_and_deep_update", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst := M{"message": "test", "agent": M{"type": "filebeat"}}
			for _, src := range manyShared {
				dst.DeepUpdate(src.Clone())
			}
			benchSinkM = dst
		}
	})

	b.Run("deep_copy_update", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst := M{"message": "test", "agent": M{"type": "filebeat"}}
			for _, src := range manyShared {
				dst.DeepCloneUpdate(src)
			}
			benchSinkM = dst
		}
	})
}

// BenchmarkMixedTypePipeline benchmarks merging into an event whose field tree
// contains map[string]interface{} subtrees — the common case when events are
// decoded from JSON before processors run. This exercises the map[string]interface{}
// destination branch added to fix the v0.36.0 regression.
func BenchmarkMixedTypePipeline(b *testing.B) {
	// Processor metadata: pure mapstr.M (as built by add_fields, add_cloud_metadata, etc.)
	sharedMeta := []M{
		{"elastic_agent": M{"id": "agent-uuid", "snapshot": false, "version": "8.12.0"}},
		{"agent": M{"id": "agent-uuid"}},
		{"data_stream": M{"type": "logs", "dataset": "system.syslog", "namespace": "default"}},
		{"event": M{"dataset": "system.syslog"}},
		{"cloud": M{
			"provider": "aws", "region": "us-east-1",
			"account": M{"id": "123456789012"}, "instance": M{"id": "i-0abcdef"},
		}},
	}

	// makeDst returns an event whose subtrees are map[string]interface{} —
	// simulating JSON-decoded input where the decoder returns native Go maps.
	makeDst := func() M {
		return M{
			"message": "request completed in 42ms",
			"event": map[string]interface{}{
				"start": "2024-01-01T00:00:00Z",
				"end":   "2024-01-01T00:00:01Z",
			},
			"process": map[string]interface{}{
				"start": "2024-01-01T00:00:00Z",
				"pid":   1234,
				"name":  "myapp",
			},
			"host": map[string]interface{}{
				"name": "prod-server-01",
				"os":   map[string]interface{}{"type": "linux", "version": "22.04"},
			},
		}
	}

	b.Run("clone_and_deep_update", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			dst := makeDst()
			for _, src := range sharedMeta {
				dst.DeepUpdate(src.Clone())
			}
			benchSinkM = dst
		}
	})

	b.Run("deep_clone_update", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			dst := makeDst()
			for _, src := range sharedMeta {
				dst.DeepCloneUpdate(src)
			}
			benchSinkM = dst
		}
	})
}

func TestDeepCloneUpdateNoOverwriteMapReplacesScalar(t *testing.T) {
	dst := M{
		"host":    "26.101.84.62",
		"message": "log line",
	}
	src := M{
		"host":  M{"name": "my-beat"},
		"agent": M{"name": "my-beat", "type": "filebeat"},
	}

	expected := M{
		"host":    "26.101.84.62",
		"message": "log line",
	}
	expected.DeepUpdateNoOverwrite(src.Clone())

	dst.DeepCloneUpdateNoOverwrite(src)

	assert.Equal(t, expected, dst, "DeepCloneUpdateNoOverwrite must match DeepUpdateNoOverwrite semantics when map replaces scalar")

	hostVal, ok := dst["host"].(M)
	require.True(t, ok, "host must be a map after map-over-scalar merge, got %T", dst["host"])
	assert.Equal(t, "my-beat", hostVal["name"])
}

func TestDeepCloneUpdateNoOverwriteMapReplacesScalarNoAliasing(t *testing.T) {
	src := M{
		"host": M{"name": "my-beat"},
	}
	srcCopy := src.Clone()

	dst := M{"host": "1.2.3.4"}
	dst.DeepCloneUpdateNoOverwrite(src)

	hostVal, ok := dst["host"].(M)
	require.True(t, ok)
	hostVal["name"] = "MUTATED"

	assert.Equal(t, srcCopy, src, "source must not be affected by mutations to destination")
}

// BenchmarkChainedMerge simulates the elastic agent pipeline where
// 5 shared processor maps are merged into each event.
func BenchmarkChainedMerge(b *testing.B) {
	b.Run("clone_and_deep_update", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst := M{"message": "test", "host": M{"name": "server1"}, "agent": M{"type": "filebeat"}}
			for _, src := range benchShared {
				dst.DeepUpdate(src.Clone())
			}
			benchSinkM = dst
		}
	})

	b.Run("deep_copy_update", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dst := M{"message": "test", "host": M{"name": "server1"}, "agent": M{"type": "filebeat"}}
			for _, src := range benchShared {
				dst.DeepCloneUpdate(src)
			}
			benchSinkM = dst
		}
	})
}
