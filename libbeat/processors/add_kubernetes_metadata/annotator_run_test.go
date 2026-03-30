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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// newAnnotatorForTest builds a kubernetesAnnotator with a pre-populated cache
// (no network calls). The matcher looks up events by "container.id".
func newAnnotatorForTest(t *testing.T, cacheKey string, meta mapstr.M) *kubernetesAnnotator {
	t.Helper()

	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"lookup_fields": []string{"container.id"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	processor := &kubernetesAnnotator{
		log:   logptest.NewTestingLogger(t, selector),
		cache: newCache(10 * time.Second),
		matchers: &Matchers{
			matchers: []Matcher{matcher},
		},
		kubernetesAvailable: true,
	}
	processor.cache.set(cacheKey, meta)
	return processor
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
	container, ok := containerRaw.(mapstr.M)
	require.True(t, ok, "container must be a mapstr.M")

	assert.Equal(t, "abc123", container["id"], "container.id should be set")
	assert.Equal(t, "containerd", container["runtime"], "container.runtime should be set")

	imageRaw, err := container.GetValue("image")
	require.NoError(t, err, "container.image must be set")
	imageMap, ok := imageRaw.(mapstr.M)
	require.True(t, ok, "container.image must be a mapstr.M")
	assert.Equal(t, "myimage:latest", imageMap["name"], "container.image.name should match original image value")

	assert.NotContains(t, container, "name", "container must NOT have a 'name' key")
	_, hasRawImage := container["image"].(string)
	assert.False(t, hasRawImage, "container.image must not be a raw string")

	// --- kubernetes field ---
	k8sRaw, err := event.Fields.GetValue("kubernetes")
	require.NoError(t, err, "event.Fields[\"kubernetes\"] must be set")
	k8s, ok := k8sRaw.(mapstr.M)
	require.True(t, ok)

	k8sContainerRaw, err := k8s.GetValue("container")
	require.NoError(t, err, "kubernetes.container must be present")
	k8sContainer, ok := k8sContainerRaw.(mapstr.M)
	require.True(t, ok)

	assert.Equal(t, "mycontainer", k8sContainer["name"], "kubernetes.container.name should be kept")
	assert.NotContains(t, k8sContainer, "id", "kubernetes.container must NOT have id")
	assert.NotContains(t, k8sContainer, "runtime", "kubernetes.container must NOT have runtime")
	assert.NotContains(t, k8sContainer, "image", "kubernetes.container must NOT have image")
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
	container, ok := containerRaw.(mapstr.M)
	require.True(t, ok)

	assert.Equal(t, "abc456", container["id"])
	assert.Equal(t, "docker", container["runtime"])
	assert.NotContains(t, container, "image", "container must NOT have image key when no image in metadata")
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
	container, ok := containerRaw.(mapstr.M)
	require.True(t, ok)

	assert.Equal(t, "abc789", container["id"])
	imageRaw, err := container.GetValue("image")
	require.NoError(t, err, "container.image must be set")
	imageMap, ok := imageRaw.(mapstr.M)
	require.True(t, ok)
	assert.Equal(t, "busybox:latest", imageMap["name"])
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
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"lookup_fields": []string{"pod.name"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	processor := &kubernetesAnnotator{
		log:   logptest.NewTestingLogger(t, selector),
		cache: newCache(10 * time.Second),
		matchers: &Matchers{
			matchers: []Matcher{matcher},
		},
		kubernetesAvailable: true,
	}
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
	k8s, ok := k8sRaw.(mapstr.M)
	require.True(t, ok)

	podRaw, err := k8s.GetValue("pod")
	require.NoError(t, err)
	pod, ok := podRaw.(mapstr.M)
	require.True(t, ok)
	assert.Equal(t, "mypod", pod["name"])
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
	container, ok := containerRaw.(mapstr.M)
	require.True(t, ok)

	assert.Equal(t, "extra", container["custom_field"], "extra container fields must be preserved in OCI container")
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
	for i := 0; i < 3; i++ {
		_, err := processor.Run(baseEvent("cache001"))
		require.NoError(t, err)
	}

	// Inspect the cache directly.
	cached := processor.cache.get("cache001")
	require.NotNil(t, cached, "cache entry must still exist")

	k8sRaw, err := cached.GetValue("kubernetes")
	require.NoError(t, err)
	k8s, ok := k8sRaw.(mapstr.M)
	require.True(t, ok)

	k8sContainerRaw, err := k8s.GetValue("container")
	require.NoError(t, err, "kubernetes.container must still be in cache")
	k8sContainer, ok := k8sContainerRaw.(mapstr.M)
	require.True(t, ok)

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
	container1, ok := containerRaw1.(mapstr.M)
	require.True(t, ok)
	container1["injected"] = "mutation"

	// event2's container field must be unaffected.
	containerRaw2, err := event2.Fields.GetValue("container")
	require.NoError(t, err)
	container2, ok := containerRaw2.(mapstr.M)
	require.True(t, ok)

	assert.NotContains(t, container2, "injected", "mutating first result must not affect second result")
}
