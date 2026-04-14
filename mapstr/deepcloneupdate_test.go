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
