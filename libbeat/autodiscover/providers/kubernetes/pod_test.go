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
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/logp"
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
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "foobar",
					},
				},
			},
		},
		// Kubernetes payload with container info must be bubbled to top level
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "rkt",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "rkt",
					},
				},
				"container": common.MapStr{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "rkt",
				},
			},
		},
		// Scenarios being tested:
		// logs/multiline.pattern must be a nested common.MapStr under hints.logs
		// logs/json.keys_under_root must be a nested common.MapStr under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.logs/multiline.pattern":    "^test",
						"co.elastic.logs/json.keys_under_root": "true",
						"co.elastic.metrics/module":            "prometheus",
						"co.elastic.metrics/period":            "10s",
						"co.elastic.metrics.foobar/period":     "15s",
						"not.to.include":                       "true",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.logs/multiline.pattern":    "^test",
						"co.elastic.logs/json.keys_under_root": "true",
						"co.elastic.metrics/module":            "prometheus",
						"not.to.include":                       "true",
						"co.elastic.metrics/period":            "10s",
						"co.elastic.metrics.foobar/period":     "15s",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"multiline": common.MapStr{
							"pattern": "^test",
						},
						"json": common.MapStr{
							"keys_under_root": "true",
						},
					},
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": common.MapStr{
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
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.logs/multiline.pattern":    "^test",
						"co.elastic.logs/json.keys_under_root": "true",
						"not.to.include":                       "true",
					}),
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.logs/multiline.pattern":    "^test",
						"co.elastic.logs/json.keys_under_root": "true",
						"not.to.include":                       "true",
					}),
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
						"co.elastic.metrics/module":        "prometheus",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"multiline": common.MapStr{
							"pattern": "^test",
						},
						"json": common.MapStr{
							"keys_under_root": "true",
						},
					},
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": common.MapStr{
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
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
						"not.to.include":                   "true",
					}),
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module":        "dropwizard",
						"co.elastic.metrics/period":        "60s",
						"co.elastic.metrics.foobar/period": "25s",
					}),
					"namespace": "ns",
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
						"not.to.include":                   "true",
					}),
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module":        "dropwizard",
						"co.elastic.metrics/period":        "60s",
						"co.elastic.metrics.foobar/period": "25s",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": common.MapStr{
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
				"kubernetes": common.MapStr{
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module":        "prometheus",
						"co.elastic.metrics/period":        "10s",
						"co.elastic.metrics.foobar/period": "15s",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
					"namespace": "ns",
				},
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": common.MapStr{
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

func TestEmitEvent(t *testing.T) {
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
					"ports":    common.MapStr{},
					"kubernetes": common.MapStr{
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
					},
					"config": []*common.Config{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     0,
					"id":       cid,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*common.Config{},
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
					"ports": common.MapStr{
						"port1": int32(8080),
						"port2": int32(9090),
					},
					"kubernetes": common.MapStr{
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
					},
					"config": []*common.Config{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(8080),
					"id":       cid,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"runtime": "docker",
							"id":      "foobar",
						},
					},
					"config": []*common.Config{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     int32(9090),
					"id":       cid,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*common.Config{},
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
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        name,
							ContainerID: containerID,
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
					"host":     "",
					"id":       uid,
					"provider": UUID,
					"ports":    common.MapStr{},
					"kubernetes": common.MapStr{
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
					},
					"config": []*common.Config{},
				},
				{
					"stop":     true,
					"host":     "",
					"id":       cid,
					"port":     0,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"runtime": "",
							"id":      "",
						},
					},
					"config": []*common.Config{},
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
					"host":     "127.0.0.1",
					"id":       uid,
					"provider": UUID,
					"ports":    common.MapStr{},
					"kubernetes": common.MapStr{
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
					},
					"config": []*common.Config{},
				},
				{
					"stop":     true,
					"host":     "127.0.0.1",
					"port":     0,
					"id":       cid,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"id":      "",
							"runtime": "",
						},
					},
					"config": []*common.Config{},
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
					"host":     "127.0.0.1",
					"id":       uid,
					"provider": UUID,
					"ports":    common.MapStr{},
					"kubernetes": common.MapStr{
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
					},
					"config": []*common.Config{},
				},
				{
					"stop":     true,
					"host":     "127.0.0.1",
					"port":     0,
					"id":       cid,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"runtime": "",
							"id":      "",
						},
					},
					"config": []*common.Config{},
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
					"ports":    common.MapStr{},
					"kubernetes": common.MapStr{
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
					},
					"config": []*common.Config{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     0,
					"id":       cid,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*common.Config{},
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
					"ports":    common.MapStr{},
					"kubernetes": common.MapStr{
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
					},
					"config": []*common.Config{},
				},
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     0,
					"id":       cid,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*common.Config{},
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
					"ports":    common.MapStr{},
					"kubernetes": common.MapStr{
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
						},
					},
					"config": []*common.Config{},
				},
				// Container
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     0,
					"id":       cid,
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "foobar",
							"name":    "filebeat",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar",
							"runtime": "docker",
						},
					},
					"config": []*common.Config{},
				},
				// Init container
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     0,
					"id":       cid + "-init",
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "foobar-init",
							"name":    "filebeat-init",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat-init",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar-init",
							"runtime": "docker",
						},
					},
					"config": []*common.Config{},
				},
				// Ephemeral container
				{
					"start":    true,
					"host":     "127.0.0.1",
					"port":     0,
					"id":       cid + "-ephemeral",
					"provider": UUID,
					"kubernetes": common.MapStr{
						"container": common.MapStr{
							"id":      "foobar-ephemeral",
							"name":    "filebeat-ephemeral",
							"image":   "elastic/filebeat:6.3.0",
							"runtime": "docker",
						},
						"pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
						"node": common.MapStr{
							"name": "node",
						},
						"namespace":   "default",
						"annotations": common.MapStr{},
					},
					"meta": common.MapStr{
						"kubernetes": common.MapStr{
							"namespace": "default",
							"pod": common.MapStr{
								"name": "filebeat",
								"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
							}, "node": common.MapStr{
								"name": "node",
							},
							"container": common.MapStr{
								"name":  "filebeat-ephemeral",
								"image": "elastic/filebeat:6.3.0",
							},
						},
						"container": common.MapStr{
							"image":   common.MapStr{"name": "elastic/filebeat:6.3.0"},
							"id":      "foobar-ephemeral",
							"runtime": "docker",
						},
					},
					"config": []*common.Config{},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Message, func(t *testing.T) {
			mapper, err := template.NewConfigMapper(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			metaGen := metadata.NewPodMetadataGenerator(common.NewConfig(), nil, nil, nil)
			p := &Provider{
				config:    defaultConfig(),
				bus:       bus.New(logp.NewLogger("bus"), "test"),
				templates: mapper,
				logger:    logp.NewLogger("kubernetes"),
			}

			pod := &pod{
				metagen: metaGen,
				config:  defaultConfig(),
				publish: p.publish,
				uuid:    UUID,
				logger:  logp.NewLogger("kubernetes.pod"),
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

func NewMockPodEventerManager(pod *pod) EventManager {
	em := &eventerManager{}
	em.eventer = pod
	return em
}

func getNestedAnnotations(in common.MapStr) common.MapStr {
	out := common.MapStr{}

	for k, v := range in {
		out.Put(k, v)
	}
	return out
}
