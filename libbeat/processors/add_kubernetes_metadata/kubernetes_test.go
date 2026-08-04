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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/goleak"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// assertRunPdataEquivalent verifies that RunPdata, given the same input fields
// used to produce result via Run, enriches a pcommon.Map with identical output.
func assertRunPdataEquivalent(t *testing.T, p beat.Processor, input, result mapstr.M) {
	t.Helper()

	pp, ok := p.(processors.PdataProcessor)
	require.True(t, ok, "processor must implement PdataProcessor")

	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input))
	drop, err := pp.RunPdata(body)
	require.NoError(t, err)
	require.False(t, drop)

	legacyNorm := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(legacyNorm, result))
	assert.Equal(t, otelmap.ToMapstr(legacyNorm), otelmap.ToMapstr(body),
		"Run and RunPdata must produce identical output")
}

// Test Annotator is skipped if kubernetes metadata already exist
func TestAnnotatorSkipped(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"lookup_fields": []string{"kubernetes.pod.name"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(t, ""))

	if err != nil {
		t.Fatal(err)
	}

	processor := kubernetesAnnotator{
		log:   logptest.NewTestingLogger(t, selector),
		cache: newCache(10 * time.Second),
	}
	setReady(&processor, &Matchers{matchers: []Matcher{matcher}})

	processor.cache.set("foo",
		mapstr.M{
			"kubernetes": mapstr.M{
				"pod": mapstr.M{
					"labels": mapstr.M{
						"added": "should not",
					},
				},
			},
		})

	input := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name": "foo",
				"id":   "pod_id",
				"metrics": mapstr.M{
					"a": 1,
					"b": 2,
				},
			},
		},
	}
	event, err := processor.Run(&beat.Event{Fields: input.Clone()})
	assert.NoError(t, err)

	assert.Equal(t, input, event.Fields)

	// RunPdata path: assert Run == RunPdata.
	assertRunPdataEquivalent(t, &processor, input, event.Fields)
}

// Test metadata are not included in the event
func TestAnnotatorWithNoKubernetesAvailable(t *testing.T) {
	// state is left nil, i.e. init never published (kubernetes unavailable).
	processor := kubernetesAnnotator{
		cache: newCache(10 * time.Second),
	}

	intialEventMap := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name": "foo",
				"id":   "pod_id",
				"metrics": mapstr.M{
					"a": 1,
					"b": 2,
				},
			},
		},
	}

	event, err := processor.Run(&beat.Event{
		Fields: intialEventMap.Clone(),
	})
	assert.NoError(t, err)

	assert.Equal(t, intialEventMap, event.Fields)

	// RunPdata path: assert Run == RunPdata.
	assertRunPdataEquivalent(t, &processor, intialEventMap, event.Fields)
}

// TestNewProcessorConfigDefaultIndexers validates the behaviour of default indexers and
// matchers settings
func TestNewProcessorConfigDefaultIndexers(t *testing.T) {
	emptyRegister := NewRegister()
	registerWithDefaults := NewRegister()
	registerWithDefaults.AddDefaultIndexerConfig("ip_port", *config.NewConfig())
	registerWithDefaults.AddDefaultMatcherConfig("field_format", *config.MustNewConfigFrom(map[string]any{
		"format": "%{[destination.ip]}:%{[destination.port]}",
	}))

	configWithIndexersAndMatchers := config.MustNewConfigFrom(map[string]any{
		"indexers": []map[string]any{
			{
				"container": map[string]any{},
			},
		},
		"matchers": []map[string]any{
			{
				"fields": map[string]any{
					"lookup_fields": []string{"container.id"},
				},
			},
		},
	})
	configOverrideDefaults := config.MustNewConfigFrom(map[string]any{
		"default_indexers.enabled": "false",
		"default_matchers.enabled": "false",
	})
	require.NoError(t, configOverrideDefaults.Merge(configWithIndexersAndMatchers))

	cases := map[string]struct {
		register         *Register
		config           *config.C
		expectedMatchers []string
		expectedIndexers []string
	}{
		"no matchers": {
			register: emptyRegister,
			config:   config.NewConfig(),
		},
		"one configured indexer and matcher": {
			register:         emptyRegister,
			config:           configWithIndexersAndMatchers,
			expectedIndexers: []string{"container"},
			expectedMatchers: []string{"fields"},
		},
		"default indexers and matchers": {
			register:         registerWithDefaults,
			config:           config.NewConfig(),
			expectedIndexers: []string{"ip_port"},
			expectedMatchers: []string{"field_format"},
		},
		"default indexers and matchers, don't use indexers": {
			register: registerWithDefaults,
			config: config.MustNewConfigFrom(map[string]any{
				"default_indexers.enabled": "false",
			}),
			expectedMatchers: []string{"field_format"},
		},
		"default indexers and matchers, don't use matchers": {
			register: registerWithDefaults,
			config: config.MustNewConfigFrom(map[string]any{
				"default_matchers.enabled": "false",
			}),
			expectedIndexers: []string{"ip_port"},
		},
		"one configured indexer and matcher and defaults, configured should come first": {
			register:         registerWithDefaults,
			config:           configWithIndexersAndMatchers,
			expectedIndexers: []string{"container", "ip_port"},
			expectedMatchers: []string{"fields", "field_format"},
		},
		"override defaults": {
			register:         registerWithDefaults,
			config:           configOverrideDefaults,
			expectedIndexers: []string{"container"},
			expectedMatchers: []string{"fields"},
		},
	}

	names := func(plugins PluginConfig) []string {
		var ns []string
		for _, plugin := range plugins {
			for name := range plugin {
				ns = append(ns, name)
			}
		}
		return ns
	}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			config, err := newProcessorConfig(c.config, c.register)
			require.NoError(t, err)
			assert.Equal(t, c.expectedMatchers, names(config.Matchers), "expected matchers")
			assert.Equal(t, c.expectedIndexers, names(config.Indexers), "expected indexers")
		})
	}
}

// newAnnotatorForTest builds a kubernetesAnnotator with a pre-populated cache
// (no network calls). The matcher looks up events by "container.id".
func newAnnotatorForTest(t testing.TB, cacheKey string, meta mapstr.M) *kubernetesAnnotator {
	t.Helper()

	cfg := config.MustNewConfigFrom(map[string]any{
		"lookup_fields": []string{"container.id"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	processor := &kubernetesAnnotator{
		log:   logptest.NewTestingLogger(t, selector),
		cache: newCache(10 * time.Second),
	}
	setReady(processor, &Matchers{matchers: []Matcher{matcher}})
	processor.cache.set(cacheKey, meta)
	return processor
}

// setReady publishes a minimal initializedState so the processor treats
// kubernetes as available, mirroring what init does on success. Tests that
// don't drive real watchers only need matchers populated.
func setReady(k *kubernetesAnnotator, matchers *Matchers) {
	k.state.Store(&initializedState{matchers: matchers})
}

// baseEvent returns an event that will match cacheKey via container.id.
func baseEvent(containerID string) *beat.Event {
	return &beat.Event{
		Fields: mapstr.M{
			"container": mapstr.M{
				"id": containerID,
			},
		},
	}
}

// TestAnnotatorRunFullContainerMetadata verifies the primary split behaviour:
// OCI container field gets id/runtime/image.name but NOT name or raw image;
// kubernetes field gets container.name but NOT id/runtime/image.
func TestAnnotatorRunFullContainerMetadata(t *testing.T) {
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
			"container": mapstr.M{
				"name":    "mycontainer",
				"image":   "myimage:latest",
				"id":      "abc123",
				"runtime": "containerd",
			},
		},
	}
	processor := newAnnotatorForTest(t, "abc123", meta)

	event, err := processor.Run(baseEvent("abc123"))
	require.NoError(t, err)

	// --- OCI container field ---
	containerRaw, err := event.Fields.GetValue("container")
	require.NoError(t, err, "event.Fields[\"container\"] must be set")
	require.IsType(t, mapstr.M{}, containerRaw, "container must be a mapstr.M")
	container, _ := containerRaw.(mapstr.M)

	assert.Equal(t, "abc123", container["id"], "container.id should be set")
	assert.Equal(t, "containerd", container["runtime"], "container.runtime should be set")

	imageRaw, err := container.GetValue("image")
	require.NoError(t, err, "container.image must be set")
	require.IsType(t, mapstr.M{}, imageRaw, "container.image must be a mapstr.M")
	imageMap, _ := imageRaw.(mapstr.M)
	assert.Equal(t, "myimage:latest", imageMap["name"], "container.image.name should match original image value")

	assert.NotContains(t, container, "name", "container must NOT have a 'name' key")
	_, hasRawImage := container["image"].(string)
	assert.False(t, hasRawImage, "container.image must not be a raw string")

	// --- kubernetes field ---
	k8sRaw, err := event.Fields.GetValue("kubernetes")
	require.NoError(t, err, "event.Fields[\"kubernetes\"] must be set")
	require.IsType(t, mapstr.M{}, k8sRaw)
	k8s, _ := k8sRaw.(mapstr.M)

	k8sContainerRaw, err := k8s.GetValue("container")
	require.NoError(t, err, "kubernetes.container must be present")
	require.IsType(t, mapstr.M{}, k8sContainerRaw)
	k8sContainer, _ := k8sContainerRaw.(mapstr.M)

	assert.Equal(t, "mycontainer", k8sContainer["name"], "kubernetes.container.name should be kept")
	assert.NotContains(t, k8sContainer, "id", "kubernetes.container must NOT have id")
	assert.NotContains(t, k8sContainer, "runtime", "kubernetes.container must NOT have runtime")
	assert.NotContains(t, k8sContainer, "image", "kubernetes.container must NOT have image")

	// RunPdata path: assert Run == RunPdata.
	assertRunPdataEquivalent(t, processor, mapstr.M{"container": mapstr.M{"id": "abc123"}}, event.Fields)
}

// TestAnnotatorRunContainerWithoutImage verifies that when there is no image in
// the metadata, the OCI container field has id and runtime but no image key.
func TestAnnotatorRunContainerWithoutImage(t *testing.T) {
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
			"container": mapstr.M{
				"name":    "mycontainer",
				"id":      "abc456",
				"runtime": "docker",
			},
		},
	}
	processor := newAnnotatorForTest(t, "abc456", meta)

	event, err := processor.Run(baseEvent("abc456"))
	require.NoError(t, err)

	containerRaw, err := event.Fields.GetValue("container")
	require.NoError(t, err)
	require.IsType(t, mapstr.M{}, containerRaw)
	container, _ := containerRaw.(mapstr.M)

	assert.Equal(t, "abc456", container["id"])
	assert.Equal(t, "docker", container["runtime"])
	assert.NotContains(t, container, "image", "container must NOT have image key when no image in metadata")

	// RunPdata path: assert Run == RunPdata.
	assertRunPdataEquivalent(t, processor, baseEvent("abc456").Fields, event.Fields)
}

// TestAnnotatorRunContainerWithoutName verifies that missing container.name
// does not panic and the OCI container field still has id and image.name.
func TestAnnotatorRunContainerWithoutName(t *testing.T) {
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
			"container": mapstr.M{
				"image": "busybox:latest",
				"id":    "abc789",
			},
		},
	}
	processor := newAnnotatorForTest(t, "abc789", meta)

	event, err := processor.Run(baseEvent("abc789"))
	require.NoError(t, err)

	containerRaw, err := event.Fields.GetValue("container")
	require.NoError(t, err)
	require.IsType(t, mapstr.M{}, containerRaw)
	container, _ := containerRaw.(mapstr.M)

	assert.Equal(t, "abc789", container["id"])
	imageRaw, err := container.GetValue("image")
	require.NoError(t, err, "container.image must be set")
	require.IsType(t, mapstr.M{}, imageRaw)
	imageMap, _ := imageRaw.(mapstr.M)
	assert.Equal(t, "busybox:latest", imageMap["name"])

	// RunPdata path: assert Run == RunPdata.
	assertRunPdataEquivalent(t, processor, baseEvent("abc789").Fields, event.Fields)
}

// TestAnnotatorRunNoContainerSubMap verifies that when the metadata has no
// kubernetes.container key at all, the OCI container field is not created and
// the kubernetes field is correctly populated.
func TestAnnotatorRunNoContainerSubMap(t *testing.T) {
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name": "mypod",
				"uid":  "uid-001",
			},
		},
	}

	// Use pod.name as the lookup field since there's no container sub-map.
	cfg := config.MustNewConfigFrom(map[string]any{
		"lookup_fields": []string{"pod.name"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	processor := &kubernetesAnnotator{
		log:   logptest.NewTestingLogger(t, selector),
		cache: newCache(10 * time.Second),
	}
	setReady(processor, &Matchers{matchers: []Matcher{matcher}})
	processor.cache.set("mypod", meta)

	event, err := processor.Run(&beat.Event{
		Fields: mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
		},
	})
	require.NoError(t, err)

	// OCI container field should NOT be set.
	_, containerErr := event.Fields.GetValue("container")
	assert.Error(t, containerErr, "event.Fields[\"container\"] must NOT be set when there is no kubernetes.container")

	// kubernetes field should be present and correct.
	k8sRaw, err := event.Fields.GetValue("kubernetes")
	require.NoError(t, err, "event.Fields[\"kubernetes\"] must be set")
	require.IsType(t, mapstr.M{}, k8sRaw)
	k8s, _ := k8sRaw.(mapstr.M)

	podRaw, err := k8s.GetValue("pod")
	require.NoError(t, err)
	require.IsType(t, mapstr.M{}, podRaw)
	pod, _ := podRaw.(mapstr.M)
	assert.Equal(t, "mypod", pod["name"])

	// RunPdata path: assert Run == RunPdata.
	assertRunPdataEquivalent(t, processor, mapstr.M{"pod": mapstr.M{"name": "mypod"}}, event.Fields)
}

// TestAnnotatorRunExtraContainerFieldsPreserved verifies that unknown extra
// fields in kubernetes.container are forwarded to the OCI container field.
func TestAnnotatorRunExtraContainerFieldsPreserved(t *testing.T) {
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
			"container": mapstr.M{
				"name":         "mycontainer",
				"image":        "myimage:v1",
				"id":           "xtra001",
				"runtime":      "containerd",
				"custom_field": "extra",
			},
		},
	}
	processor := newAnnotatorForTest(t, "xtra001", meta)

	event, err := processor.Run(baseEvent("xtra001"))
	require.NoError(t, err)

	containerRaw, err := event.Fields.GetValue("container")
	require.NoError(t, err)
	require.IsType(t, mapstr.M{}, containerRaw)
	container, _ := containerRaw.(mapstr.M)

	assert.Equal(t, "extra", container["custom_field"], "extra container fields must be preserved in OCI container")

	// RunPdata path: assert Run == RunPdata.
	assertRunPdataEquivalent(t, processor, baseEvent("xtra001").Fields, event.Fields)
}

// TestAnnotatorRunCacheNotMutated verifies that running the processor multiple
// times on different events does not mutate the cached metadata entry.
func TestAnnotatorRunCacheNotMutated(t *testing.T) {
	originalMeta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
			"container": mapstr.M{
				"name":    "mycontainer",
				"image":   "myimage:v2",
				"id":      "cache001",
				"runtime": "containerd",
			},
		},
	}
	processor := newAnnotatorForTest(t, "cache001", originalMeta)

	// Run three times.
	for range 3 {
		_, err := processor.Run(baseEvent("cache001"))
		require.NoError(t, err)
	}

	// Inspect the cache directly.
	cached := processor.cache.get("cache001")
	require.NotNil(t, cached, "cache entry must still exist")

	k8sRaw, err := cached.GetValue("kubernetes")
	require.NoError(t, err)
	require.IsType(t, mapstr.M{}, k8sRaw)
	k8s, _ := k8sRaw.(mapstr.M)

	k8sContainerRaw, err := k8s.GetValue("container")
	require.NoError(t, err, "kubernetes.container must still be in cache")
	require.IsType(t, mapstr.M{}, k8sContainerRaw)
	k8sContainer, _ := k8sContainerRaw.(mapstr.M)

	assert.Equal(t, "mycontainer", k8sContainer["name"], "cache must still have container.name")
	assert.Equal(t, "myimage:v2", k8sContainer["image"], "cache must still have container.image as a raw string")
	assert.Equal(t, "cache001", k8sContainer["id"], "cache must still have container.id")
	assert.Equal(t, "containerd", k8sContainer["runtime"], "cache must still have container.runtime")
}

// TestAnnotatorRunEventIndependence verifies that mutating the container field
// on the result of one Run() call does not affect the result of a subsequent call.
func TestAnnotatorRunEventIndependence(t *testing.T) {
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
			"container": mapstr.M{
				"name":    "mycontainer",
				"image":   "myimage:v3",
				"id":      "indep001",
				"runtime": "containerd",
			},
		},
	}
	processor := newAnnotatorForTest(t, "indep001", meta)

	event1, err := processor.Run(baseEvent("indep001"))
	require.NoError(t, err)

	event2, err := processor.Run(baseEvent("indep001"))
	require.NoError(t, err)

	// Mutate event1's container field.
	containerRaw1, err := event1.Fields.GetValue("container")
	require.NoError(t, err)
	require.IsType(t, mapstr.M{}, containerRaw1)
	container1, _ := containerRaw1.(mapstr.M)
	container1["injected"] = "mutation"

	// event2's container field must be unaffected.
	containerRaw2, err := event2.Fields.GetValue("container")
	require.NoError(t, err)
	require.IsType(t, mapstr.M{}, containerRaw2)
	container2, _ := containerRaw2.(mapstr.M)

	assert.NotContains(t, container2, "injected", "mutating first result must not affect second result")
}

// TestAnnotatorRunCacheMiss verifies that when the matcher returns an index key
// but the cache has no entry for it, both Run and RunPdata leave the event
// unchanged and return no error.
func TestAnnotatorRunCacheMiss(t *testing.T) {
	// Seed the cache with a different key so the matcher can fire but miss.
	processor := newAnnotatorForTest(t, "known-key", mapstr.M{
		"kubernetes": mapstr.M{"pod": mapstr.M{"name": "mypod"}},
	})

	input := mapstr.M{"container": mapstr.M{"id": "unknown-key"}}
	event, err := processor.Run(&beat.Event{Fields: input.Clone()})
	require.NoError(t, err)
	assert.Equal(t, input, event.Fields, "Run must leave the event unchanged on a cache miss")

	// RunPdata path: assert Run == RunPdata.
	assertRunPdataEquivalent(t, processor, input, event.Fields)
}

func BenchmarkKubernetesAnnotatorRun(b *testing.B) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"lookup_fields": []string{"container.id"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(b, ""))
	if err != nil {
		b.Fatal(err)
	}

	processor := &kubernetesAnnotator{
		log:   logptest.NewTestingLogger(b, selector),
		cache: newCache(10 * time.Second),
	}
	setReady(processor, &Matchers{matchers: []Matcher{matcher}})

	const cacheKey = "abc123container"

	processor.cache.set(cacheKey, mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name":      "test-pod",
				"uid":       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				"namespace": "default",
				"labels": mapstr.M{
					"app":     "myapp",
					"version": "v1.2.3",
					"env":     "production",
				},
				"annotations": mapstr.M{
					"deployment.kubernetes.io/revision": "3",
				},
			},
			"node": mapstr.M{
				"name": "node-1",
			},
			"namespace": "default",
			"container": mapstr.M{
				"name":    "mycontainer",
				"image":   "myrepo/myimage:latest",
				"id":      cacheKey,
				"runtime": "containerd",
			},
		},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Construct a minimal event with only the lookup field — no Clone() overhead
		// counted against the benchmark. Run() will add kubernetes.* and container.*
		// fields to this fresh event.
		event := &beat.Event{
			Fields: mapstr.M{
				"container": mapstr.M{
					"id": cacheKey,
				},
				"message": "some log line",
			},
		}
		_, err := processor.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// unavailableK8sClient returns a client whose discovery always fails, so
// isKubernetesAvailableWithTimeout loops until timeout or context cancel.
func unavailableK8sClient() k8sclient.Interface {
	client := k8sfake.NewSimpleClientset()
	client.PrependReactor("get", "version", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("kubernetes unavailable")
	})
	return client
}

// TestCloseUnblocksAsyncInit verifies that Close cancels an async init stuck in the
// indefinite kubernetes availability wait (timeout=0) and does not leak goroutines.
func TestCloseUnblocksAsyncInit(t *testing.T) {
	// Ignore cache.cleanup goroutines from other tests in this package that never call cache.stop().
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	logger := logptest.NewTestingLogger(t, selector)
	ctx, cancel := context.WithCancel(context.Background())

	proc := &kubernetesAnnotator{
		log:       logger,
		cache:     newCache(10 * time.Second),
		cancelCtx: cancel,
	}

	proc.wg.Add(1)
	initStarted := make(chan struct{})
	go func() {
		defer proc.wg.Done()
		proc.initOnce.Do(func() {
			close(initStarted)
			_, _ = isKubernetesAvailableWithTimeout(
				ctx,
				unavailableK8sClient(),
				0, // wait indefinitely (same as wait_for_metadata_timeout: 0)
				10*time.Millisecond,
				logger,
			)
		})
	}()

	<-initStarted
	// Ensure we are past the first failed discovery and inside the select/wait loop.
	time.Sleep(30 * time.Millisecond)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- proc.Close()
	}()

	select {
	case err := <-closeDone:
		require.NoError(t, err, "Close must return after cancelling the availability wait")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Close blocked: on main Close() waits on initOnce while init is stuck in the availability loop")
	}
}

func TestIsKubernetesAvailableWithTimeout(t *testing.T) {
	logger := logptest.NewTestingLogger(t, selector)
	unavailable := unavailableK8sClient()
	available := k8sfake.NewSimpleClientset()

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		ok, err := isKubernetesAvailableWithTimeout(ctx, available, 0, time.Millisecond, logger)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("timeout", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()
		ok, err := isKubernetesAvailableWithTimeout(ctx, unavailable, 80*time.Millisecond, 10*time.Millisecond, logger)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.False(t, ok)
		assert.Contains(t, err.Error(), "timeout waiting for kubernetes")
		assert.GreaterOrEqual(t, elapsed, 80*time.Millisecond)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ok, err := isKubernetesAvailableWithTimeout(ctx, unavailable, 0, 10*time.Millisecond, logger)
		require.Error(t, err)
		assert.False(t, ok)
		assert.Contains(t, err.Error(), "context cancelled")
		assert.ErrorIs(t, err, context.Canceled)
	})
}
