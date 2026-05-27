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

package add_cloud_metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func Test_addCloudMetadata_String(t *testing.T) {
	const timeout = 100 * time.Millisecond
	cfg := conf.MustNewConfigFrom(map[string]any{
		"providers": []string{"openstack"},
		"host":      "fake:1234",
		"timeout":   timeout.String(),
	})
	p, err := New(cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	assert.Eventually(t, func() bool { return p.String() == "add_cloud_metadata=<uninitialized>" }, timeout, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return p.String() == "add_cloud_metadata={}" }, 2*timeout, 10*time.Millisecond)
}

// makeProcessorWithMetadata creates an addCloudMetadata processor whose
// p.metadata is pre-seeded without hitting the network, for use in unit tests.
func makeProcessorWithMetadata(t *testing.T, meta mapstr.M, overwrite bool) *addCloudMetadata {
	t.Helper()
	p := &addCloudMetadata{
		initData: &initData{
			overwrite: overwrite,
		},
		initDone: make(chan struct{}),
		metadata: meta,
		logger:   logptest.NewTestingLogger(t, ""),
	}
	// Mark init as complete without actually fetching anything.
	close(p.initDone)
	p.initOnce.Do(func() {})
	return p
}

// TestAddMetaEventsAreIndependent verifies that two successive Run() calls
// produce events whose nested cloud map is independent: mutating the cloud
// field on one event must not affect the other.
func TestAddMetaEventsAreIndependent(t *testing.T) {
	meta := mapstr.M{
		"cloud": mapstr.M{
			"provider": "test",
			"instance": mapstr.M{"id": "i-123"},
		},
	}
	p := makeProcessorWithMetadata(t, meta, true)

	event1 := &beat.Event{Fields: mapstr.M{}}
	event2 := &beat.Event{Fields: mapstr.M{}}

	err := p.addMeta(event1, p.metadata)
	require.NoError(t, err)
	err = p.addMeta(event2, p.metadata)
	require.NoError(t, err)

	// The cloud maps on the two events must be independent copies.
	cloud1Raw, err := event1.Fields.GetValue("cloud")
	require.NoError(t, err)
	cloud2Raw, err := event2.Fields.GetValue("cloud")
	require.NoError(t, err)
	cloud1Map, ok := cloud1Raw.(mapstr.M)
	require.True(t, ok, "expected cloud on event1 to be mapstr.M")
	cloud2Map, ok := cloud2Raw.(mapstr.M)
	require.True(t, ok, "expected cloud on event2 to be mapstr.M")
	assert.NotSame(t, &cloud1Map, &cloud2Map, "cloud maps on different events must not be aliased")

	// Mutate the cloud map on event1.
	cloud1Map["provider"] = "mutated"

	// event2's cloud.provider must be unaffected.
	cloud2, err := event2.Fields.GetValue("cloud.provider")
	require.NoError(t, err)
	assert.Equal(t, "test", cloud2, "mutating event1's cloud field must not affect event2")
}

// TestAddMetaSharedMetadataUnmutated verifies that mutating an event's cloud
// field does not corrupt the shared p.metadata map used for subsequent events.
func TestAddMetaSharedMetadataUnmutated(t *testing.T) {
	meta := mapstr.M{
		"cloud": mapstr.M{
			"provider": "test",
			"region":   "us-east-1",
		},
	}
	p := makeProcessorWithMetadata(t, meta, true)

	event1 := &beat.Event{Fields: mapstr.M{}}
	err := p.addMeta(event1, p.metadata)
	require.NoError(t, err)

	// The event's cloud map must not be the same pointer as the shared metadata.
	cloud1Raw, err := event1.Fields.GetValue("cloud")
	require.NoError(t, err)
	cloud1Map, ok := cloud1Raw.(mapstr.M)
	require.True(t, ok, "expected cloud on event1 to be mapstr.M")
	metaCloudMap, ok := meta["cloud"].(mapstr.M)
	require.True(t, ok, "expected meta[cloud] to be mapstr.M")
	assert.NotSame(t, &cloud1Map, &metaCloudMap, "event cloud map must not alias shared metadata")

	// Aggressively mutate event1's cloud map.
	cloud1Map["region"] = "CORRUPTED"
	cloud1Map["extra"] = "bad"

	// Process a second event — it must still see the original metadata.
	event2 := &beat.Event{Fields: mapstr.M{}}
	err = p.addMeta(event2, p.metadata)
	require.NoError(t, err)

	region2, err := event2.Fields.GetValue("cloud.region")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", region2, "shared metadata must not be corrupted by event mutation")

	// The shared p.metadata itself must also be clean.
	cloudMeta, ok := meta["cloud"].(mapstr.M)
	require.True(t, ok, "expected meta[cloud] to be mapstr.M")
	assert.Equal(t, "us-east-1", cloudMeta["region"],
		"p.metadata must remain unchanged after events are processed")
}

// TestAddMetaNestedMapsAreCloned verifies that deeply nested maps in
// metadata are independently cloned for each event.
func TestAddMetaNestedMapsAreCloned(t *testing.T) {
	meta := mapstr.M{
		"cloud": mapstr.M{
			"instance": mapstr.M{
				"id":   "i-abc",
				"tags": mapstr.M{"env": "prod"},
			},
		},
	}
	p := makeProcessorWithMetadata(t, meta, true)

	event1 := &beat.Event{Fields: mapstr.M{}}
	event2 := &beat.Event{Fields: mapstr.M{}}

	err := p.addMeta(event1, p.metadata)
	require.NoError(t, err)
	err = p.addMeta(event2, p.metadata)
	require.NoError(t, err)

	// Mutate a deeply nested value in event1.
	tags1Raw, err := event1.Fields.GetValue("cloud.instance.tags")
	require.NoError(t, err)
	tags1Map, ok := tags1Raw.(mapstr.M)
	require.True(t, ok, "expected tags on event1 to be mapstr.M")
	tags2Raw, err := event2.Fields.GetValue("cloud.instance.tags")
	require.NoError(t, err)
	tags2Map, ok := tags2Raw.(mapstr.M)
	require.True(t, ok, "expected tags on event2 to be mapstr.M")
	assert.NotSame(t, &tags1Map, &tags2Map, "tags maps on different events must not be aliased")
	tags1Map["env"] = "dev"

	// event2's nested tags must be unchanged.
	tags2env, err := event2.Fields.GetValue("cloud.instance.tags.env")
	require.NoError(t, err)
	assert.Equal(t, "prod", tags2env, "deeply nested maps must be independently cloned per event")

	// p.metadata must also be unchanged.
	metaCloud, ok := meta["cloud"].(mapstr.M)
	require.True(t, ok, "expected meta[cloud] to be mapstr.M")
	metaInstance, ok := metaCloud["instance"].(mapstr.M)
	require.True(t, ok, "expected meta[cloud][instance] to be mapstr.M")
	metaTags, ok := metaInstance["tags"].(mapstr.M)
	require.True(t, ok, "expected meta[cloud][instance][tags] to be mapstr.M")
	assert.Equal(t, "prod", metaTags["env"],
		"p.metadata nested map must remain unchanged")
}

// TestAddMetaOverwriteFalseSkipsExistingKeys verifies that when overwrite=false,
// existing event fields at the top-level metadata keys are not replaced.
func TestAddMetaOverwriteFalseSkipsExistingKeys(t *testing.T) {
	meta := mapstr.M{
		"cloud": mapstr.M{
			"provider": "aws",
			"region":   "us-west-2",
		},
	}
	p := makeProcessorWithMetadata(t, meta, false /* overwrite=false */)

	event := &beat.Event{
		Fields: mapstr.M{
			"cloud": mapstr.M{"provider": "original"},
		},
	}
	err := p.addMeta(event, p.metadata)
	require.NoError(t, err)

	// The existing "cloud" key must not be overwritten.
	provider, err := event.Fields.GetValue("cloud.provider")
	require.NoError(t, err)
	assert.Equal(t, "original", provider, "overwrite=false must not replace existing cloud field")
}

// TestAddMetaOverwriteTrueReplacesExistingKeys verifies that when overwrite=true,
// existing event fields at the top-level metadata keys are replaced.
func TestAddMetaOverwriteTrueReplacesExistingKeys(t *testing.T) {
	meta := mapstr.M{
		"cloud": mapstr.M{
			"provider": "aws",
			"region":   "us-west-2",
		},
	}
	p := makeProcessorWithMetadata(t, meta, true /* overwrite=true */)

	event := &beat.Event{
		Fields: mapstr.M{
			"cloud": mapstr.M{"provider": "original"},
		},
	}
	err := p.addMeta(event, p.metadata)
	require.NoError(t, err)

	provider, err := event.Fields.GetValue("cloud.provider")
	require.NoError(t, err)
	assert.Equal(t, "aws", provider, "overwrite=true must replace existing cloud field")
}

// TestAddMetaNilMetadataReturnsEventUnchanged verifies that when no cloud
// provider was detected (p.metadata is nil), the event passes through unchanged.
func TestAddMetaNilMetadataReturnsEventUnchanged(t *testing.T) {
	p := makeProcessorWithMetadata(t, nil /* no metadata */, true)

	event := &beat.Event{Fields: mapstr.M{"existing": "value"}}
	out, err := p.Run(event)
	require.NoError(t, err)
	assert.Equal(t, event, out, "nil metadata must return event unchanged")
	assert.Equal(t, "value", out.Fields["existing"], "existing fields must be preserved")
}

// TestAddMetaScalarValuesPassedWithoutClone verifies that scalar (non-map)
// metadata values (strings, numbers) are stored correctly in the event.
func TestAddMetaScalarValuesPassedWithoutClone(t *testing.T) {
	meta := mapstr.M{
		"cloud": mapstr.M{
			"provider": "digitalocean",
			"region":   "nyc3",
		},
	}
	p := makeProcessorWithMetadata(t, meta, true)

	event := &beat.Event{Fields: mapstr.M{}}
	err := p.addMeta(event, p.metadata)
	require.NoError(t, err)

	provider, err := event.Fields.GetValue("cloud.provider")
	require.NoError(t, err)
	assert.Equal(t, "digitalocean", provider)

	region, err := event.Fields.GetValue("cloud.region")
	require.NoError(t, err)
	assert.Equal(t, "nyc3", region)
}

// TestAddMetaWithMultipleTopLevelKeys verifies that providers emitting
// multiple top-level keys (e.g. "cloud" and "orchestrator") have all keys
// added and independently cloned.
func TestAddMetaWithMultipleTopLevelKeys(t *testing.T) {
	meta := mapstr.M{
		"cloud": mapstr.M{
			"provider": "aws",
			"region":   "us-east-1",
		},
		"orchestrator": mapstr.M{
			"cluster": mapstr.M{"name": "my-cluster"},
		},
	}
	p := makeProcessorWithMetadata(t, meta, true)

	event1 := &beat.Event{Fields: mapstr.M{}}
	event2 := &beat.Event{Fields: mapstr.M{}}

	err := p.addMeta(event1, p.metadata)
	require.NoError(t, err)
	err = p.addMeta(event2, p.metadata)
	require.NoError(t, err)

	// Mutate event1's orchestrator — the cluster maps must be independent copies.
	cluster1Raw, err := event1.Fields.GetValue("orchestrator.cluster")
	require.NoError(t, err)
	cluster2Raw, err := event2.Fields.GetValue("orchestrator.cluster")
	require.NoError(t, err)
	cluster1Map, ok := cluster1Raw.(mapstr.M)
	require.True(t, ok, "expected cluster on event1 to be mapstr.M")
	cluster2Map, ok := cluster2Raw.(mapstr.M)
	require.True(t, ok, "expected cluster on event2 to be mapstr.M")
	assert.NotSame(t, &cluster1Map, &cluster2Map, "cluster maps on different events must not be aliased")
	cluster1Map["name"] = "mutated"

	// event2 must be unaffected.
	name2, err := event2.Fields.GetValue("orchestrator.cluster.name")
	require.NoError(t, err)
	assert.Equal(t, "my-cluster", name2)

	// cloud key must be present on both events.
	provider1, _ := event1.Fields.GetValue("cloud.provider")
	provider2, _ := event2.Fields.GetValue("cloud.provider")
	assert.Equal(t, "aws", provider1)
	assert.Equal(t, "aws", provider2)
}

// TestDeepCloneUpdateNoAliasing verifies that deepCopyUpdate creates
// independent copies of nested maps in the cloud metadata.
func TestDeepCloneUpdateNoAliasing(t *testing.T) {
	src := mapstr.M{
		"cloud": mapstr.M{
			"provider": "aws",
			"account":  mapstr.M{"id": "123"},
		},
	}
	srcCopy := src.Clone()

	dst := mapstr.M{}
	dst.DeepCloneUpdate(src)

	// src and dst cloud maps must be independent copies.
	srcCloud, err := src.GetValue("cloud")
	require.NoError(t, err)
	dstCloud, err := dst.GetValue("cloud")
	require.NoError(t, err)
	srcCloudMap, ok := srcCloud.(mapstr.M)
	require.True(t, ok, "expected src cloud to be mapstr.M")
	dstCloudMap, ok := dstCloud.(mapstr.M)
	require.True(t, ok, "expected dst cloud to be mapstr.M")
	assert.NotSame(t, &srcCloudMap, &dstCloudMap, "src and dst cloud maps must not be aliased after DeepCloneUpdate")

	// Mutate dst.
	dstCloudMap["provider"] = "MUTATED"

	// Source must be unchanged.
	assert.Equal(t, srcCopy, src, "source must not be affected by mutations to destination")
}

// BenchmarkAddCloudMetadata measures allocations per Run() call for a
// realistic cloud metadata payload (matching the OpenStack/AWS shape).
// Run with: go test ./libbeat/processors/add_cloud_metadata/... -run '^$' -bench '^BenchmarkAddCloudMetadata' -benchmem -count=10
func BenchmarkAddCloudMetadata(b *testing.B) {
	_ = logp.TestingSetup() //nolint:staticcheck // global logger needed for benchmark with httptest server

	server := httptest.NewServer(openstackNovaMetadataHandler())
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"providers": []string{"openstack"},
		"host":      server.Listener.Addr().String(),
	})
	if err != nil {
		b.Fatal(err)
	}

	p, err := New(config, logptest.NewTestingLogger(b, ""))
	if err != nil {
		b.Fatal(err)
	}
	// Ensure init completes before benchmarking.
	if acp, ok := p.(*addCloudMetadata); ok {
		acp.init()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &beat.Event{Fields: mapstr.M{}}
		_, err := p.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAddCloudMetadataParallel measures allocations under concurrent load.
func BenchmarkAddCloudMetadataParallel(b *testing.B) {
	_ = logp.TestingSetup() //nolint:staticcheck // global logger needed for benchmark with httptest server

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case osMetadataInstanceIDURI:
			_, _ = w.Write([]byte("i-bench-001"))
		case osMetadataInstanceTypeURI:
			_, _ = w.Write([]byte("m5.large"))
		case osMetadataHostnameURI:
			_, _ = w.Write([]byte("bench.host.example"))
		case osMetadataZoneURI:
			_, _ = w.Write([]byte("us-east-1a"))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"providers": []string{"openstack"},
		"host":      server.Listener.Addr().String(),
	})
	if err != nil {
		b.Fatal(err)
	}

	p, err := New(config, logptest.NewTestingLogger(b, ""))
	if err != nil {
		b.Fatal(err)
	}
	if acp, ok := p.(*addCloudMetadata); ok {
		acp.init()
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := &beat.Event{Fields: mapstr.M{}}
			_, err := p.Run(event)
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
}
