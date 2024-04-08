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

package kubernetes

import (
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	interfaces "k8s.io/client-go/kubernetes"
	caches "k8s.io/client-go/tools/cache"

	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes/metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		event  bus.Event
		result bus.Event
	}{
		// Empty events should return empty hints
		{
			event:  bus.Event{},
			result: bus.Event{},
		},
		// Only kubernetes payload must return only kubernetes as part of the hint
		{
			event: bus.Event{
				"kubernetes": mapstr.M{
					"pod": mapstr.M{
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"kubernetes": mapstr.M{
					"pod": mapstr.M{
						"name": "foobar",
					},
				},
			},
		},
		// Kubernetes payload with container info must be bubbled to top level
		{
			event: bus.Event{
				"kubernetes": mapstr.M{
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "rkt",
					},
				},
			},
			result: bus.Event{
				"kubernetes": mapstr.M{
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "rkt",
					},
				},
				"container": mapstr.M{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "rkt",
				},
			},
		},
		// Scenarios being tested:
		// logs/multiline.pattern must be a nested mapstr.M under hints.logs
		// logs/json.keys_under_root must be a nested mapstr.M under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			event: bus.Event{
				"kubernetes": mapstr.M{
					"annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.logs/multiline.pattern":    "^test",
						"co.elastic.logs/json.keys_under_root": "true",
						"co.elastic.metrics/module":            "prometheus",
						"co.elastic.metrics/period":            "10s",
						"co.elastic.metrics.foobar/period":     "15s",
						"not.to.include":                       "true",
					}),
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
			},
			result: bus.Event{
				"kubernetes": mapstr.M{
					"annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.logs/multiline.pattern":    "^test",
						"co.elastic.logs/json.keys_under_root": "true",
						"co.elastic.metrics/module":            "prometheus",
						"not.to.include":                       "true",
						"co.elastic.metrics/period":            "10s",
						"co.elastic.metrics.foobar/period":     "15s",
					}),
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
				"hints": mapstr.M{
					"logs": mapstr.M{
						"multiline": mapstr.M{
							"pattern": "^test",
						},
						"json": mapstr.M{
							"keys_under_root": "true",
						},
					},
					"metrics": mapstr.M{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": mapstr.M{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "docker",
				},
			},
		},
		// Scenarios tested:
		// Have one set of hints come from the pod and the other come from namespaces
		// The resultant hints should have a combination of both
		{
			event: bus.Event{
				"kubernetes": mapstr.M{
					"annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.logs/multiline.pattern":    "^test",
						"co.elastic.logs/json.keys_under_root": "true",
						"not.to.include":                       "true",
					}),
					"namespace_annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
					}),
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
			},
			result: bus.Event{
				"kubernetes": mapstr.M{
					"annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.logs/multiline.pattern":    "^test",
						"co.elastic.logs/json.keys_under_root": "true",
						"not.to.include":                       "true",
					}),
					"namespace_annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
						"co.elastic.metrics/module":        "prometheus",
					}),
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
				"hints": mapstr.M{
					"logs": mapstr.M{
						"multiline": mapstr.M{
							"pattern": "^test",
						},
						"json": mapstr.M{
							"keys_under_root": "true",
						},
					},
					"metrics": mapstr.M{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": mapstr.M{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "docker",
				},
			},
		},
		// Scenarios tested:
		// Have one set of hints come from the pod and the same keys come from namespaces
		// The resultant hints should honor only pods and not namespace.
		{
			event: bus.Event{
				"kubernetes": mapstr.M{
					"annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
						"not.to.include":                   "true",
					}),
					"namespace_annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.metrics/module":        "dropwizard",
						"co.elastic.metrics/period":        "60s",
						"co.elastic.metrics.foobar/period": "25s",
					}),
					"namespace": "ns",
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
			},
			result: bus.Event{
				"kubernetes": mapstr.M{
					"annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
						"not.to.include":                   "true",
					}),
					"namespace_annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.metrics/module":        "dropwizard",
						"co.elastic.metrics/period":        "60s",
						"co.elastic.metrics.foobar/period": "25s",
					}),
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": mapstr.M{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "docker",
				},
			},
		},
		// Scenarios tested:
		// Have no hints on the pod and have namespace level defaults.
		// The resultant hints should honor only namespace defaults.
		{
			event: bus.Event{
				"kubernetes": mapstr.M{
					"namespace_annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
					}),
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
			},
			result: bus.Event{
				"kubernetes": mapstr.M{
					"namespace_annotations": getNestedAnnotations(mapstr.M{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
					}),
					"container": mapstr.M{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": mapstr.M{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "docker",
				},
			},
		},
	}

	cfg := defaultConfig()

	p := pod{
		config: cfg,
		logger: logp.NewLogger("kubernetes.pod"),
	}
	for _, test := range tests {
		assert.Equal(t, p.GenerateHints(test.event), test.result)
	}
}

func TestPod_EmitEvent(t *testing.T) {
	name := "filebeat"
	namespace := "default"
	podIP := "127.0.0.1"
	containerID := "docker://foobar"
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	containerImage := "elastic/filebeat:6.3.0"
	node := "node"
	cid := "005f3b90-4b9d-12f8-acf0-31020a840133.filebeat"
	UUID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	typeMeta := metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}

	tests := []struct {
		Message  string
		Flag     string
		Pod      *kubernetes.Pod
		Expected []bus.Event
	}{
		{
			Message: "Test common pod start",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodRunning,
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"start":    true,
					"host":     "127.0.0.1",
					"id":       uid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(0),
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			Message: "Test common pod start with multiple ports exposed",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodRunning,
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 8080,
									Name:          "port1",
								},
								{
									ContainerPort: 9090,
									Name:          "port2",
								},
							},
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"start":    true,
					"host":     "127.0.0.1",
					"id":       uid,
					"provider": UUID,
					"ports": mapstr.M{
						"port1": int32(8080),
						"port2": int32(9090),
					},
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(8080),
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"runtime": "docker",
							"id":      "foobar",
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(9090),
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			// This could be a succeeded pod from a short-living cron job.
			Message: "Test succeeded pod start with multiple ports exposed",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodSucceeded,
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{},
							},
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 8080,
									Name:          "port1",
								},
								{
									ContainerPort: 9090,
									Name:          "port2",
								},
							},
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"start":    true,
					"id":       uid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"runtime": "docker",
							"id":      "foobar",
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			Message: "Test pod without host",
			Flag:    "start",
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					Phase: kubernetes.PodPending,
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name: name,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
						},
					},
				},
			},
			Expected: nil,
		},
		{
			Message: "Test pod without container id",
			Flag:    "start",
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodPending,
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name: name,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
						},
					},
				},
			},
			Expected: nil,
		},
		{
			Message: "Test stop pod without host",
			Flag:    "stop",
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name: name,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"stop":     true,
					"id":       uid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"stop":     true,
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"runtime": "",
							"id":      "",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			Message: "Test stop pod without container id",
			Flag:    "stop",
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name: name,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"stop":     true,
					"id":       uid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"stop":     true,
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "",
							"runtime": "",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			// This could be a succeeded pod from a short-living cron job.
			Message: "Test succeeded pod stop with multiple ports exposed",
			Flag:    "stop",
			Pod: &kubernetes.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodSucceeded,
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{},
							},
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 8080,
									Name:          "port1",
								},
								{
									ContainerPort: 9090,
									Name:          "port2",
								},
							},
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"stop":     true,
					"id":       uid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"stop":     true,
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"runtime": "docker",
							"id":      "foobar",
						},
					},
					"config": []*conf.C{},
				},
				{
					"stop":     true,
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			Message: "Test terminated init container in started common pod",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodRunning,
					InitContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name + "-init",
							ContainerID: containerID,
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{},
							},
						},
					},
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 8080,
									Name:          "http",
								},
							},
						},
					},
					InitContainers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name + "-init",
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"start": true,
					"host":  "127.0.0.1",
					"id":    uid,
					"ports": mapstr.M{
						"http": int32(8080),
					},
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   "127.0.0.1",
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   "127.0.0.1",
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(8080),
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"id":       cid + "-init",
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat-init",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat-init",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			Message: "Test init container in common pod",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodPending,
					InitContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					InitContainers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"start":    true,
					"host":     "127.0.0.1",
					"id":       uid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(0),
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			Message: "Test ephemeral container in common pod",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodRunning,
					EphemeralContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					EphemeralContainers: []v1.EphemeralContainer{
						v1.EphemeralContainer{
							EphemeralContainerCommon: v1.EphemeralContainerCommon{
								Image: containerImage,
								Name:  name,
							},
						},
					},
				},
			},
			Expected: []bus.Event{
				{
					"start":    true,
					"host":     "127.0.0.1",
					"id":       uid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(0),
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
		{
			Message: "Test pod with ephemeral, init and normal container",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: kubernetes.PodRunning,
					InitContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name + "-init",
							ContainerID: containerID + "-init",
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
					EphemeralContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name + "-ephemeral",
							ContainerID: containerID + "-ephemeral",
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: node,
					Containers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name,
						},
					},
					InitContainers: []kubernetes.Container{
						{
							Image: containerImage,
							Name:  name + "-init",
						},
					},
					EphemeralContainers: []v1.EphemeralContainer{
						v1.EphemeralContainer{
							EphemeralContainerCommon: v1.EphemeralContainerCommon{
								Image: containerImage,
								Name:  name + "-ephemeral",
							},
						},
					},
				},
			},
			Expected: []bus.Event{
				// Single pod
				{
					"start":    true,
					"host":     "127.0.0.1",
					"id":       uid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							},
						},
					},
					"config": []*conf.C{},
				},
				// Container
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(0),
					"id":       cid,
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
				// Init container
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(0),
					"id":       cid + "-init",
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar-init",
							"name":    "filebeat-init",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat-init",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar-init",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
				// Ephemeral container
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(0),
					"id":       cid + "-ephemeral",
					"provider": UUID,
					"kubernetes": mapstr.M{
						"container": mapstr.M{
							"id":      "foobar-ephemeral",
							"name":    "filebeat-ephemeral",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": mapstr.M{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							"ip":   podIP,
						},
						"node": mapstr.M{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": mapstr.M{},
						"labels":      mapstr.M{},
					},
					"meta": mapstr.M{
						"kubernetes": mapstr.M{
							"namespace": "default",
							"pod": mapstr.M{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
								"ip":   podIP,
							}, "node": mapstr.M{
								"name": "node",
							}, "container": mapstr.M{
								"name": "filebeat-ephemeral",
							},
						},
						"container": mapstr.M{
							"image":   mapstr.M{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar-ephemeral",
							"runtime": "docker",
						},
					},
					"config": []*conf.C{},
				},
			},
		},
	}

	client := k8sfake.NewSimpleClientset()
	addResourceMetadata := metadata.GetDefaultResourceMetadataConfig()
	for _, test := range tests {
		t.Run(test.Message, func(t *testing.T) {
			mapper, err := template.NewConfigMapper(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			metaGen := metadata.NewPodMetadataGenerator(conf.NewConfig(), nil, client, nil, nil, nil, nil, addResourceMetadata)
			p := &Provider{
				config:    defaultConfig(),
				bus:       bus.New(logp.NewLogger("bus"), "test"),
				templates: mapper,
				logger:    logp.NewLogger("kubernetes"),
			}

			pub := &publisher{b: p.bus}
			pod := &pod{
				metagen:     metaGen,
				config:      defaultConfig(),
				publishFunc: pub.publish,
				uuid:        UUID,
				logger:      logp.NewLogger("kubernetes.pod"),
			}

			p.eventManager = NewMockPodEventerManager(pod)

			listener := p.bus.Subscribe()

			pod.emit(test.Pod, test.Flag)

			for i := 0; i < len(test.Expected); i++ {
				select {
				case event := <-listener.Events():
					assert.Equalf(t, test.Expected[i], event, "%s/#%d", test.Message, i)
				case <-time.After(2 * time.Second):
					if test.Expected != nil {
						t.Fatalf("Timeout while waiting for event #%d", i)
					}
				}
			}

			select {
			case <-listener.Events():
				t.Error("More events received than expected")
			default:
			}
		})
	}
}

func TestNamespacePodUpdater(t *testing.T) {
	pod := func(name, namespace string) *kubernetes.Pod {
		return &kubernetes.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}

	namespace := &kubernetes.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		}}

	cases := map[string]struct {
		pods     []interface{}
		expected []interface{}
	}{
		"no pods": {},
		"two pods but only one in namespace": {
			pods: []interface{}{
				pod("onepod", "foo"),
				pod("onepod", "bar"),
			},
			expected: []interface{}{
				pod("onepod", "foo"),
			},
		},
		"two pods but none in namespace": {
			pods: []interface{}{
				pod("onepod", "bar"),
				pod("otherpod", "bar"),
			},
		},
	}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			handler := &mockUpdaterHandler{}
			store := &mockUpdaterStore{objects: c.pods}
			//We simulate an update on the namespace with the addition of one label
			namespace1 := &kubernetes.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						"beta.kubernetes.io/arch": "arm64",
					},
				}}

			watcher := &mockUpdaterWatcher{cachedObject: namespace}
			updater := kubernetes.NewNamespacePodUpdater(handler.OnUpdate, store, watcher, &sync.Mutex{})

			updater.OnUpdate(namespace1)

			assert.EqualValues(t, c.expected, handler.objects)
		})
	}
}

func TestNodePodUpdater(t *testing.T) {
	pod := func(name, node string) *kubernetes.Pod {
		return &kubernetes.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1.PodSpec{
				NodeName: node,
			},
		}
	}

	node := &kubernetes.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}

	cases := map[string]struct {
		pods []interface{}

		expected []interface{}
	}{
		"no pods": {},
		"two pods but only one in node": {
			pods: []interface{}{
				pod("onepod", "foo"),
				pod("onepod", "bar"),
			},
			expected: []interface{}{
				pod("onepod", "foo"),
			},
		},
		"two pods but none in node": {
			pods: []interface{}{
				pod("onepod", "bar"),
				pod("otherpod", "bar"),
			},
		},
	}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			handler := &mockUpdaterHandler{}
			store := &mockUpdaterStore{objects: c.pods}

			//We simulate an update on the node with the addition of one label
			node1 := &kubernetes.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
					Annotations: map[string]string{
						"beta.kubernetes.io/arch": "arm64",
					},
				}}

			watcher := &mockUpdaterWatcher{cachedObject: node}
			updater := kubernetes.NewNodePodUpdater(handler.OnUpdate, store, watcher, &sync.Mutex{})

			//This is when the update happens.
			updater.OnUpdate(node1)

			assert.EqualValues(t, c.expected, handler.objects)
		})
	}
}

func TestPodEventer_Namespace_Node_Watcher(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	uuid, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		cfg         mapstr.M
		expectedNil bool
		name        string
		msg         string
	}{
		{
			cfg: mapstr.M{
				"resource": "pod",
				"node":     "node-1",
				"add_resource_metadata": mapstr.M{
					"namespace.enabled": false,
					"node.enabled":      false,
				},
				"hints.enabled": false,
				"builders": []mapstr.M{
					{
						"mock": mapstr.M{},
					},
				},
			},
			expectedNil: true,
			name:        "add_resource_metadata.namespace and add_resource_metadata.node disabled and hints disabled.",
			msg:         "Watcher should be nil.",
		},
		{
			cfg: mapstr.M{
				"resource": "pod",
				"node":     "node-1",
				"add_resource_metadata": mapstr.M{
					"namespace.enabled": false,
					"node.enabled":      false,
				},
				"hints.enabled": true,
			},
			expectedNil: false,
			name:        "add_resource_metadata.namespace and add_resource_metadata.node disabled and hints enabled.",
			msg:         "Watcher should not be nil.",
		},
		{
			cfg: mapstr.M{
				"resource": "pod",
				"node":     "node-1",
				"add_resource_metadata": mapstr.M{
					"namespace.enabled": true,
					"node.enabled":      true,
				},
				"hints.enabled": false,
				"builders": []mapstr.M{
					{
						"mock": mapstr.M{},
					},
				},
			},
			expectedNil: false,
			name:        "add_resource_metadata.namespace and add_resource_metadata.node enabled and hints disabled.",
			msg:         "Watcher should not be nil.",
		},
		{
			cfg: mapstr.M{
				"resource": "pod",
				"node":     "node-1",
				"builders": []mapstr.M{
					{
						"mock": mapstr.M{},
					},
				},
			},
			expectedNil: false,
			name:        "add_resource_metadata default and hints default.",
			msg:         "Watcher should not be nil.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := conf.MustNewConfigFrom(&test.cfg)
			c := defaultConfig()
			err = config.Unpack(&c)
			assert.NoError(t, err)

			eventer, err := NewPodEventer(uuid, config, client, nil)
			if err != nil {
				t.Fatal(err)
			}

			namespaceWatcher := eventer.(*pod).namespaceWatcher
			nodeWatcher := eventer.(*pod).nodeWatcher

			if test.expectedNil {
				assert.Equalf(t, nil, namespaceWatcher, "Namespace "+test.msg)
				assert.Equalf(t, nil, nodeWatcher, "Node "+test.msg)
			} else {
				assert.NotEqualf(t, nil, namespaceWatcher, "Namespace "+test.msg)
				assert.NotEqualf(t, nil, nodeWatcher, "Node "+test.msg)
			}
		})
	}
}

type mockUpdaterHandler struct {
	objects []interface{}
}

func (h *mockUpdaterHandler) OnUpdate(obj interface{}) {
	h.objects = append(h.objects, obj)
}

type mockUpdaterStore struct {
	objects []interface{}
}

var store caches.Store
var client interfaces.Interface
var err error

type mockUpdaterWatcher struct {
	cachedObject runtime.Object
}

func (s *mockUpdaterWatcher) CachedObject() runtime.Object {
	return s.cachedObject
}

func (s *mockUpdaterWatcher) Client() interfaces.Interface {
	return client
}

func (s *mockUpdaterWatcher) Start() error {
	return err
}

func (s *mockUpdaterWatcher) Stop() {
}

func (s *mockUpdaterWatcher) Store() caches.Store {
	return store
}

func (s *mockUpdaterWatcher) AddEventHandler(kubernetes.ResourceEventHandler) {
}

func (s *mockUpdaterWatcher) GetEventHandler() kubernetes.ResourceEventHandler {
	return nil
}

func (s *mockUpdaterStore) List() []interface{} {
	return s.objects
}

func NewMockPodEventerManager(pod *pod) EventManager {
	em := &eventerManager{}
	em.eventer = pod
	return em
}

type publisher struct {
	b bus.Bus
}

func (p *publisher) publish(events []bus.Event) {
	if len(events) == 0 {
		return
	}
	for _, event := range events {
		event["config"] = []*conf.C{}
		p.b.Publish(event)
	}
}

func getNestedAnnotations(in mapstr.M) mapstr.M {
	out := mapstr.M{}

	for k, v := range in {
		_, _ = out.Put(k, v)
	}
	return out
}
