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

package add_kubernetes_metadata

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors/shared"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func metaMap(containerID string) mapstr.M {
	return mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name":      "mypod",
				"uid":       "uid-" + containerID,
				"namespace": "default",
				"labels":    mapstr.M{"app": "myapp"},
			},
			"node": mapstr.M{"name": "node-1"},
			"container": mapstr.M{
				"name":    "mycontainer",
				"image":   "myimage:latest",
				"id":      containerID,
				"runtime": "containerd",
			},
		},
	}
}

func newAnnotatorWithMeta(t testing.TB, containerID string) *kubernetesAnnotator {
	t.Helper()
	return newAnnotatorForTest(t, containerID, metaMap(containerID))
}

func containerEvent(containerID string) *beat.Event {
	return &beat.Event{
		Fields: mapstr.M{
			"container": mapstr.M{"id": containerID},
		},
	}
}

func TestAnnotatorRun_ConcurrentRace(t *testing.T) {
	const goroutines = 100
	const containerID = "race-cid"

	processor := newAnnotatorWithMeta(t, containerID)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			event := containerEvent(containerID)
			out, err := processor.Run(event)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if out == nil {
				t.Error("Run returned nil event")
			}
		}()
	}
	wg.Wait()
}

func TestAnnotatorRun_ConcurrentRace_CacheWrite(t *testing.T) {
	const goroutines = 50
	const containerID = "cache-write-cid"

	processor := newAnnotatorWithMeta(t, containerID)
	meta := metaMap(containerID)

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Half of the goroutines read via Run.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = processor.Run(containerEvent(containerID))
		}()
	}

	// Half write via the cache directly (simulating the watcher callbacks).
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			processor.cache.set(containerID, meta)
		}()
	}

	wg.Wait()
}

// verify that when many  goroutines each process a distinct event, every event is enriched with the
// correct kubernetes metadata.
func TestAnnotatorRun_EachEventAnnotatedIndependently(t *testing.T) {
	const goroutines = 50

	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"lookup_fields": []string{"container.id"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	processor := &kubernetesAnnotator{
		log:                 logptest.NewTestingLogger(t, selector),
		cache:               newCache(10 * time.Second),
		matchers:            &Matchers{matchers: []Matcher{matcher}},
		kubernetesAvailable: true,
	}

	for i := 0; i < goroutines; i++ {
		cid := fmt.Sprintf("cid-%d", i)
		processor.cache.set(cid, metaMap(cid))
	}

	type result struct {
		event *beat.Event
		err   error
	}
	results := make([]result, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			cid := fmt.Sprintf("cid-%d", i)
			out, runErr := processor.Run(containerEvent(cid))
			results[i] = result{event: out, err: runErr}
		}(i)
	}
	wg.Wait()

	for i, r := range results {
		cid := fmt.Sprintf("cid-%d", i)
		require.NoError(t, r.err, "goroutine %d must not return an error", i)
		require.NotNil(t, r.event, "goroutine %d must receive a non-nil event", i)

		containerRaw, err := r.event.Fields.GetValue("container")
		require.NoError(t, err, "goroutine %d: event must have container field", i)
		container := containerRaw.(mapstr.M) //nolint:errcheck // it's a test
		assert.Equal(t, cid, container["id"], "goroutine %d: container.id must match its own cache key", i)

		k8sRaw, err := r.event.Fields.GetValue("kubernetes")
		require.NoError(t, err)
		k8s := k8sRaw.(mapstr.M)     //nolint:errcheck // it's a test
		pod := k8s["pod"].(mapstr.M) //nolint:errcheck // it's a test
		assert.Equal(t, "uid-"+cid, pod["uid"], "goroutine %d: kubernetes.pod.uid must match its own cache key", i)
	}
}

func TestAnnotatorRun_MutatingOneResultDoesNotAffectOthers(t *testing.T) {
	const goroutines = 30
	const containerID = "shared-cid"

	processor := newAnnotatorWithMeta(t, containerID)

	events := make([]*beat.Event, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			out, err := processor.Run(containerEvent(containerID))
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", i, err)
				return
			}
			events[i] = out
		}()
	}
	wg.Wait()

	// mutate the first event's container map.
	if events[0] != nil {
		containerRaw, err := events[0].Fields.GetValue("container")
		require.NoError(t, err)
		container := containerRaw.(mapstr.M) //nolint:errcheck // it's a test
		container["_injected_by_goroutine_0"] = true
	}

	// all other events must be unaffected.
	for i := 1; i < goroutines; i++ {
		if events[i] == nil {
			continue
		}
		containerRaw, err := events[i].Fields.GetValue("container")
		require.NoError(t, err, "goroutine %d event must still have container field", i)
		container := containerRaw.(mapstr.M) //nolint:errcheck // it's a test
		assert.NotContains(t, container, "_injected_by_goroutine_0",
			"goroutine %d's container must be independent from goroutine 0's result", i)
	}
}

func TestAnnotatorRun_CacheMutationDoesNotAffectInFlightEvents(t *testing.T) {
	const containerID = "inflight-cid"
	processor := newAnnotatorWithMeta(t, containerID)

	const readers = 40
	const writers = 10

	results := make([]*beat.Event, readers)
	var wg sync.WaitGroup
	wg.Add(readers + writers)

	for i := 0; i < readers; i++ {
		i := i
		go func() {
			defer wg.Done()
			out, err := processor.Run(containerEvent(containerID))
			if err != nil {
				t.Errorf("reader %d: unexpected error: %v", i, err)
				return
			}
			results[i] = out
		}()
	}

	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			// Write a completely different metadata map to the same key.
			newMeta := metaMap(containerID)
			newMeta["kubernetes"].(mapstr.M)["pod"].(mapstr.M)["name"] = "replaced-pod" //nolint:errcheck // it's a test
			processor.cache.set(containerID, newMeta)
		}()
	}

	wg.Wait()

	// every event that was annotated should have pod.name that is either
	// "mypod" (original) or "replaced-pod" (updated)
	for i, event := range results {
		if event == nil {
			continue
		}
		k8sRaw, err := event.Fields.GetValue("kubernetes")
		if err != nil {
			// event may have been processed before the first cache.set.
			continue
		}
		k8s := k8sRaw.(mapstr.M)     //nolint:errcheck // it's a test
		pod := k8s["pod"].(mapstr.M) //nolint:errcheck // it's a test
		name, ok := pod["name"].(string)
		assert.True(t, ok, "event %d: pod.name must be a string", i)
		assert.Contains(t, []string{"mypod", "replaced-pod"}, name,
			"event %d: pod.name must be one of the two valid values", i)
	}
}

func TestAnnotatorRun_SharedWrapper_EventIndependenceUnderConcurrency(t *testing.T) {
	const goroutines = 60

	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"lookup_fields": []string{"container.id"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	cache := newCache(10 * time.Second)
	annotator := &kubernetesAnnotator{
		log:                 logptest.NewTestingLogger(t, selector),
		cache:               cache,
		matchers:            &Matchers{matchers: []Matcher{matcher}},
		kubernetesAvailable: true,
	}

	// each goroutine has its own container ID in the cache.
	for i := range goroutines {
		cid := fmt.Sprintf("ev-%d", i)
		cache.set(cid, metaMap(cid))
	}

	sharedConstructor := shared.New(func(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
		return annotator, nil
	})
	proc, err := sharedConstructor(nil, nil)
	require.NoError(t, err)

	type result struct {
		event *beat.Event
		cid   string
		err   error
	}
	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		cid := fmt.Sprintf("ev-%d", i)
		go func() {
			defer wg.Done()
			out, runErr := proc.Run(containerEvent(cid))
			results[i] = result{event: out, cid: cid, err: runErr}
		}()
	}
	wg.Wait()

	for i, r := range results {
		require.NoError(t, r.err, "goroutine %d must not return an error", i)
		require.NotNil(t, r.event, "goroutine %d must receive a non-nil event", i)

		// container.id must match this goroutine's own cache key.
		containerRaw, getErr := r.event.Fields.GetValue("container")
		require.NoError(t, getErr, "goroutine %d: event must have container field", i)
		container := containerRaw.(mapstr.M) //nolint:errcheck // it's a test
		assert.Equal(t, r.cid, container["id"],
			"goroutine %d: container.id must equal its own cache key, not another goroutine's", i)

		// kubernetes.pod.uid must also be specific to this goroutine's entry.
		k8sRaw, getErr := r.event.Fields.GetValue("kubernetes")
		require.NoError(t, getErr, "goroutine %d: event must have kubernetes field", i)
		k8s := k8sRaw.(mapstr.M)     //nolint:errcheck // it's a test
		pod := k8s["pod"].(mapstr.M) //nolint:errcheck // it's a test
		assert.Equal(t, "uid-"+r.cid, pod["uid"],
			"goroutine %d: kubernetes.pod.uid must belong to its own cache entry", i)
	}
}
