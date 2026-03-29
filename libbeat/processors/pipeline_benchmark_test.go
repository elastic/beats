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

//go:build linux || darwin || windows

package processors

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// processorFunc adapts a function to the beat.Processor interface.
type processorFunc func(*beat.Event) (*beat.Event, error)

func (f processorFunc) Run(event *beat.Event) (*beat.Event, error) { return f(event) }
func (f processorFunc) String() string                             { return "processorFunc" }

// BenchmarkPipelinePerEvent measures bytes allocated per event through a
// realistic Elastic Agent processor pipeline.
//
// Sub-benchmarks compare the pre-PR (Baseline) and post-PR (Optimized)
// allocation patterns. The pipeline mirrors a real deployment:
//
//   - 6 × add_fields  (agent, input, ecs, host, cloud, elastic_agent)
//   - 1 × add_docker_metadata
//   - 1 × add_kubernetes_metadata
//   - 1 × dissect (OverwriteKeys=true)
//   - 1 × timestamp parse
//   - 1 × add_meta (@metadata pipeline step)
//
// Each simulated processor reproduces the exact allocation-relevant code
// from the real Run() method — the lines that Clone, build maps, and
// DeepUpdate. Non-allocation work (field lookups, string matching, cache
// hits) is skipped because it doesn't affect B/op.
//
// The B/op delta between Baseline and Optimized is the per-event memory
// saving from this PR.
func BenchmarkPipelinePerEvent(b *testing.B) {
	b.Run("Baseline", func(b *testing.B) { benchPipeline(b, false) })
	b.Run("Optimized", func(b *testing.B) { benchPipeline(b, true) })
}

// ---------------------------------------------------------------------------
// Shared metadata fixtures
// ---------------------------------------------------------------------------

var dockerContainerMeta = mapstr.M{
	"container": mapstr.M{
		"labels": mapstr.M{
			"app":     "myapp",
			"version": "v1.2.3",
			"env":     "production",
		},
		"id":    "abc123def456",
		"image": mapstr.M{"name": "myrepo/myimage:latest"},
		"name":  "my-container",
	},
}

var k8sCachedMeta = mapstr.M{
	"kubernetes": mapstr.M{
		"pod": mapstr.M{
			"name": "test-pod", "uid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			"namespace": "default",
			"labels":    mapstr.M{"app": "myapp", "version": "v1.2.3", "env": "production"},
			"annotations": mapstr.M{
				"deployment.kubernetes.io/revision": "3",
			},
		},
		"node":      mapstr.M{"name": "node-1"},
		"namespace": "default",
		"container": mapstr.M{
			"name": "mycontainer", "image": "myrepo/myimage:latest",
			"id": "abc123container", "runtime": "containerd",
		},
	},
}

var eventMeta = mapstr.M{
	"pipeline": "filebeat-8.17.0-system-syslog-pipeline",
	"index":    "logs-system.syslog-default",
}

// ---------------------------------------------------------------------------
// Pipeline benchmark
// ---------------------------------------------------------------------------

func benchPipeline(b *testing.B, optimized bool) {
	b.Helper()

	procs := make([]beat.Processor, 0, 12)

	// --- 1. Six add_fields processors (shared=true, real processor) ---
	// These are identical between baseline and optimized — they show the
	// constant per-event cost of the agent metadata pipeline.
	for _, f := range []mapstr.M{
		{"fields": mapstr.M{"agent": mapstr.M{
			"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "name": "my-agent-host",
			"version": "8.17.0", "type": "filebeat", "hostname": "my-agent-host",
		}}},
		{"fields": mapstr.M{"input": mapstr.M{"type": "filestream"},
			"log": mapstr.M{"file": mapstr.M{"path": "/var/log/app/app.log", "inode": "12345678", "device": "64769"}, "offset": 42},
		}},
		{"fields": mapstr.M{"ecs": mapstr.M{"version": "8.0.0"},
			"data_stream": mapstr.M{"type": "logs", "dataset": "generic", "namespace": "default"},
		}},
		{"fields": mapstr.M{"host": mapstr.M{
			"name": "my-agent-host", "hostname": "my-agent-host", "architecture": "x86_64",
			"os": mapstr.M{"type": "linux", "platform": "ubuntu", "name": "Ubuntu", "family": "debian", "version": "22.04", "kernel": "5.15.0-91-generic"},
			"ip": []string{"10.0.0.5", "172.17.0.1"}, "mac": []string{"02:42:ac:11:00:01"}, "id": "a1b2c3d4e5f67890",
		}}},
		{"fields": mapstr.M{"cloud": mapstr.M{
			"provider": "gcp", "availability_zone": "us-central1-a", "region": "us-central1",
			"instance": mapstr.M{"id": "1234567890123456789", "name": "gke-cluster-default-pool-abc123"},
			"machine": mapstr.M{"type": "e2-standard-4"}, "project": mapstr.M{"id": "my-project-123456"},
			"account": mapstr.M{"id": "my-project-123456"},
		}}},
		{"fields": mapstr.M{"elastic_agent": mapstr.M{
			"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "version": "8.17.0", "snapshot": false,
		}}},
	} {
		procs = append(procs, addfields.NewAddFields(f, true, true))
	}

	// --- 2. add_docker_metadata ---
	if optimized {
		procs = append(procs, processorFunc(dockerRunOptimized))
	} else {
		procs = append(procs, processorFunc(dockerRunBaseline))
	}

	// --- 3. add_kubernetes_metadata ---
	if optimized {
		procs = append(procs, processorFunc(k8sRunOptimized))
	} else {
		procs = append(procs, processorFunc(k8sRunBaseline))
	}

	// --- 4. dissect (OverwriteKeys=true) ---
	// Dissect itself (tokenizing a string) has the same cost either way.
	// The optimization is skipping event.Clone() for the rollback backup.
	// We simulate only the Clone-or-not decision and a PutValue to model
	// the mapper writing dissected fields.
	if optimized {
		procs = append(procs, processorFunc(dissectRunOptimized))
	} else {
		procs = append(procs, processorFunc(dissectRunBaseline))
	}

	// --- 5. timestamp parse (success path) ---
	// Old code: allocates &parseError{} before trying layouts.
	// New code: only allocates parseError on failure.
	// On the success path the difference is one fewer heap allocation.
	if optimized {
		procs = append(procs, processorFunc(timestampRunOptimized))
	} else {
		procs = append(procs, processorFunc(timestampRunBaseline))
	}

	// --- 6. addMeta (@metadata pipeline step) ---
	if optimized {
		procs = append(procs, processorFunc(addMetaRunOptimized))
	} else {
		procs = append(procs, processorFunc(addMetaRunBaseline))
	}

	// --- Benchmark loop ---
	tsStr := time.Date(2025, 3, 7, 11, 6, 39, 123456789, time.UTC).Format(time.RFC3339Nano)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &beat.Event{
			Fields: mapstr.M{
				"message":         "2025-03-07T11:06:39.123456789Z INFO application started successfully status=ok user=admin",
				"timestamp_field": tsStr,
			},
		}
		var err error
		for _, p := range procs {
			event, err = p.Run(event)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// add_docker_metadata — allocation-relevant code from Run()
// ---------------------------------------------------------------------------

// Baseline (main): builds a local meta map, then Clones it before DeepUpdate.
// Source: event.Fields.DeepUpdate(meta.Clone())
func dockerRunBaseline(event *beat.Event) (*beat.Event, error) {
	meta := dockerContainerMeta.Clone()
	event.Fields.DeepUpdate(meta.Clone())
	return event, nil
}

// Optimized (PR): same local meta map, no redundant Clone.
// Source: event.Fields.DeepUpdate(meta)
func dockerRunOptimized(event *beat.Event) (*beat.Event, error) {
	meta := dockerContainerMeta.Clone()
	event.Fields.DeepUpdate(meta)
	return event, nil
}

// ---------------------------------------------------------------------------
// add_kubernetes_metadata — allocation-relevant code from Run()
// ---------------------------------------------------------------------------

// Baseline (main): 2 full metadata Clones + 1 GetValue on a Clone.
// Source: kubernetes.go lines 352-379 on main.
//
//	metaClone := metadata.Clone()                         // Clone 1
//	metaClone.Delete / Put (transform container.image)
//	cmeta, _ := metaClone.Clone().GetValue(...)           // Clone 2 (only to read)
//	event.Fields.DeepUpdate(container: cmeta)
//	kubeMeta := metadata.Clone()                          // Clone 3
//	kubeMeta.Delete(container.id/runtime/image)
//	event.Fields.DeepUpdate(kubeMeta)
func k8sRunBaseline(event *beat.Event) (*beat.Event, error) {
	metaClone := k8sCachedMeta.Clone()
	_ = metaClone.Delete("kubernetes.container.name")
	containerImage, err := k8sCachedMeta.GetValue("kubernetes.container.image")
	if err == nil {
		_ = metaClone.Delete("kubernetes.container.image")
		_, _ = metaClone.Put("kubernetes.container.image.name", containerImage)
	}
	cmeta, err := metaClone.Clone().GetValue("kubernetes.container")
	if err == nil {
		event.Fields.DeepUpdate(mapstr.M{"container": cmeta})
	}

	kubeMeta := k8sCachedMeta.Clone()
	_ = kubeMeta.Delete("kubernetes.container.id")
	_ = kubeMeta.Delete("kubernetes.container.runtime")
	_ = kubeMeta.Delete("kubernetes.container.image")
	event.Fields.DeepUpdate(kubeMeta)

	return event, nil
}

// Optimized (PR): 1 full Clone + 1 container sub-Clone.
// Source: kubernetes.go lines 356-379 on PR branch.
func k8sRunOptimized(event *beat.Event) (*beat.Event, error) {
	kubeMeta := k8sCachedMeta.Clone()

	if containerVal, err := kubeMeta.GetValue("kubernetes.container"); err == nil {
		if cm, ok := containerVal.(mapstr.M); ok {
			ociContainer := cm.Clone()
			_ = ociContainer.Delete("name")
			if img, imgErr := ociContainer.GetValue("image"); imgErr == nil {
				_ = ociContainer.Delete("image")
				ociContainer["image"] = mapstr.M{"name": img}
			}
			event.Fields.DeepUpdate(mapstr.M{"container": ociContainer})
		}
	}

	_ = kubeMeta.Delete("kubernetes.container.id")
	_ = kubeMeta.Delete("kubernetes.container.runtime")
	_ = kubeMeta.Delete("kubernetes.container.image")
	event.Fields.DeepUpdate(kubeMeta)

	return event, nil
}

// ---------------------------------------------------------------------------
// dissect — allocation-relevant code from Run()
// ---------------------------------------------------------------------------

// Baseline (main): unconditionally Clones the entire event for rollback,
// even when OverwriteKeys=true makes failure impossible.
// Source: backup := event.Clone()
func dissectRunBaseline(event *beat.Event) (*beat.Event, error) {
	_ = event.Clone() // unconditional backup
	// mapper() puts dissected fields — simulate one PutValue
	_, _ = event.PutValue("dissect.status", "ok")
	return event, nil
}

// Optimized (PR): skips Clone when OverwriteKeys=true.
// Source: if !p.config.OverwriteKeys { backup = event.Clone() }
func dissectRunOptimized(event *beat.Event) (*beat.Event, error) {
	// no Clone — OverwriteKeys=true means mapper() cannot fail
	_, _ = event.PutValue("dissect.status", "ok")
	return event, nil
}

// ---------------------------------------------------------------------------
// timestamp — allocation-relevant code from parseValue()
// ---------------------------------------------------------------------------

// parseError matches the real parseError struct size for accurate allocation
// measurement. The real struct has: field string, time interface{}, causes []error.
type benchParseError struct {
	field  string
	time   interface{}
	causes []error
}

func (e *benchParseError) Error() string { return "parse error" }

// Baseline (main): allocates &parseError{} eagerly before trying layouts.
// Source: detailedErr := &parseError{}
func timestampRunBaseline(event *beat.Event) (*beat.Event, error) {
	_ = &benchParseError{} // eager allocation (wasted on success path)

	val, _ := event.GetValue("timestamp_field")
	if s, ok := val.(string); ok {
		if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
			_, _ = event.PutValue("@timestamp", ts.UTC())
		}
	}
	return event, nil
}

// Optimized (PR): no allocation on success path; parseError only created
// after all layouts fail.
func timestampRunOptimized(event *beat.Event) (*beat.Event, error) {
	val, _ := event.GetValue("timestamp_field")
	if s, ok := val.(string); ok {
		if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
			_, _ = event.PutValue("@timestamp", ts.UTC())
		}
	}
	return event, nil
}

// ---------------------------------------------------------------------------
// addMeta — allocation-relevant code from processors.go addMeta()
// ---------------------------------------------------------------------------

// Baseline (main): calls event.Meta.Clone() but discards the result (dead code).
// Source: event.Meta.Clone()  (return value not captured)
func addMetaRunBaseline(event *beat.Event) (*beat.Event, error) {
	meta := eventMeta.Clone()
	if event.Meta == nil {
		event.Meta = meta
	} else {
		event.Meta.Clone() // dead code — result discarded
		event.Meta.DeepUpdate(meta)
	}
	return event, nil
}

// Optimized (PR): dead Clone removed.
func addMetaRunOptimized(event *beat.Event) (*beat.Event, error) {
	meta := eventMeta.Clone()
	if event.Meta == nil {
		event.Meta = meta
	} else {
		event.Meta.DeepUpdate(meta)
	}
	return event, nil
}
