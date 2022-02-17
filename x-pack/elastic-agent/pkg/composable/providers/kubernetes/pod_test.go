// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
)

func TestGeneratePodData(t *testing.T) {
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	pod := &kubernetes.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpod",
			UID:       types.UID(uid),
			Namespace: "testns",
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"app": "production",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: kubernetes.PodSpec{
			NodeName: "testnode",
		},
		Status: kubernetes.PodStatus{PodIP: "127.0.0.5"},
	}

	namespaceAnnotations := common.MapStr{
		"nsa": "nsb",
	}
	data := generatePodData(pod, &Config{}, &podMeta{}, namespaceAnnotations)

	mapping := map[string]interface{}{
		"namespace": pod.GetNamespace(),
		"pod": common.MapStr{
			"uid":  string(pod.GetUID()),
			"name": pod.GetName(),
			"ip":   pod.Status.PodIP,
		},
		"namespace_annotations": common.MapStr{
			"nsa": "nsb",
		},
		"labels": common.MapStr{
			"foo": "bar",
		},
		"annotations": common.MapStr{
			"app": "production",
		},
	}

	processors := map[string]interface{}{
		"orchestrator": common.MapStr{
			"cluster": common.MapStr{
				"name": "devcluster",
				"url":  "8.8.8.8:9090"},
		}, "kubernetes": common.MapStr{
			"namespace": "testns",
			"labels": common.MapStr{
				"foo": "bar",
			},
			"annotations": common.MapStr{"app": "production"},
			"pod": common.MapStr{
				"ip":   "127.0.0.5",
				"name": "testpod",
				"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133"}},
	}
	assert.Equal(t, string(pod.GetUID()), data.uid)
	assert.Equal(t, mapping, data.mapping)
	for _, v := range data.processors {
		k := v["add_fields"].(map[string]interface{})
		target := k["target"].(string)
		fields := k["fields"]
		assert.Equal(t, processors[target], fields)
	}
}

func TestGenerateContainerPodData(t *testing.T) {
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	containers := []kubernetes.Container{
		{
			Name:  "nginx",
			Image: "nginx:1.120",
			Ports: []kubernetes.ContainerPort{
				{
					Name:          "http",
					Protocol:      v1.ProtocolTCP,
					ContainerPort: 80,
				},
			},
		},
	}
	containerStatuses := []kubernetes.PodContainerStatus{
		{
			Name:        "nginx",
			Ready:       true,
			ContainerID: "crio://asdfghdeadbeef",
		},
	}
	pod := &kubernetes.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpod",
			UID:       types.UID(uid),
			Namespace: "testns",
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"app": "production",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: kubernetes.PodSpec{
			NodeName:   "testnode",
			Containers: containers,
		},
		Status: kubernetes.PodStatus{
			PodIP:             "127.0.0.5",
			ContainerStatuses: containerStatuses,
		},
	}

	providerDataChan := make(chan providerData, 1)

	comm := MockDynamicComm{
		context.TODO(),
		providerDataChan,
	}
	generateContainerData(
		&comm,
		pod,
		&Config{},
		&podMeta{},
		common.MapStr{
			"nsa": "nsb",
		})

	mapping := map[string]interface{}{
		"namespace": pod.GetNamespace(),
		"pod": common.MapStr{
			"uid":  string(pod.GetUID()),
			"name": pod.GetName(),
			"ip":   pod.Status.PodIP,
		},
		"container": common.MapStr{
			"id":        "asdfghdeadbeef",
			"name":      "nginx",
			"image":     "nginx:1.120",
			"runtime":   "crio",
			"port":      "80",
			"port_name": "http",
		},
		"namespace_annotations": common.MapStr{
			"nsa": "nsb",
		},
		"annotations": common.MapStr{
			"app": "production",
		},
		"labels": common.MapStr{
			"foo": "bar",
		},
	}

	processors := map[string]interface{}{
		"container": common.MapStr{
			"id":      "asdfghdeadbeef",
			"image":   common.MapStr{"name": "nginx:1.120"},
			"runtime": "crio",
		}, "orchestrator": common.MapStr{
			"cluster": common.MapStr{
				"name": "devcluster",
				"url":  "8.8.8.8:9090"},
		}, "kubernetes": common.MapStr{
			"namespace":   "testns",
			"annotations": common.MapStr{"app": "production"},
			"labels":      common.MapStr{"foo": "bar"},
			"pod": common.MapStr{
				"ip":   "127.0.0.5",
				"name": "testpod",
				"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133"}},
	}
	cuid := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), "nginx")
	data := <-providerDataChan
	assert.Equal(t, cuid, data.uid)
	assert.Equal(t, mapping, data.mapping)
	for _, v := range data.processors {
		k := v["add_fields"].(map[string]interface{})
		target := k["target"].(string)
		fields := k["fields"]
		assert.Equal(t, processors[target], fields)
	}

}

func TestEphemeralContainers(t *testing.T) {
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	containers := []v1.EphemeralContainer{
		{
			EphemeralContainerCommon: v1.EphemeralContainerCommon{
				Image: "nginx:1.120",
				Name:  "nginx",
			},
		},
	}
	containerStatuses := []kubernetes.PodContainerStatus{
		{
			Name:        "nginx",
			Ready:       true,
			ContainerID: "crio://asdfghdeadbeef",
		},
	}
	pod := &kubernetes.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpod",
			UID:       types.UID(uid),
			Namespace: "testns",
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"app": "production",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: kubernetes.PodSpec{
			NodeName:            "testnode",
			EphemeralContainers: containers,
		},
		Status: kubernetes.PodStatus{
			PodIP:                      "127.0.0.5",
			EphemeralContainerStatuses: containerStatuses,
		},
	}

	providerDataChan := make(chan providerData, 1)

	comm := MockDynamicComm{
		context.TODO(),
		providerDataChan,
	}
	generateContainerData(
		&comm,
		pod,
		&Config{},
		&podMeta{},
		common.MapStr{
			"nsa": "nsb",
		})

	mapping := map[string]interface{}{
		"namespace": pod.GetNamespace(),
		"pod": common.MapStr{
			"uid":  string(pod.GetUID()),
			"name": pod.GetName(),
			"ip":   pod.Status.PodIP,
		},
		"labels": common.MapStr{
			"foo": "bar",
		},
		"container": common.MapStr{
			"id":      "asdfghdeadbeef",
			"name":    "nginx",
			"image":   "nginx:1.120",
			"runtime": "crio",
		},
		"namespace_annotations": common.MapStr{
			"nsa": "nsb",
		},
		"annotations": common.MapStr{
			"app": "production",
		},
	}

	processors := map[string]interface{}{
		"container": common.MapStr{
			"id":      "asdfghdeadbeef",
			"image":   common.MapStr{"name": "nginx:1.120"},
			"runtime": "crio",
		}, "orchestrator": common.MapStr{
			"cluster": common.MapStr{
				"name": "devcluster",
				"url":  "8.8.8.8:9090"},
		}, "kubernetes": common.MapStr{
			"namespace":   "testns",
			"labels":      common.MapStr{"foo": "bar"},
			"annotations": common.MapStr{"app": "production"},
			"pod": common.MapStr{
				"ip":   "127.0.0.5",
				"name": "testpod",
				"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133"}},
	}
	cuid := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), "nginx")
	data := <-providerDataChan
	assert.Equal(t, cuid, data.uid)
	assert.Equal(t, mapping, data.mapping)
	for _, v := range data.processors {
		k := v["add_fields"].(map[string]interface{})
		target := k["target"].(string)
		fields := k["fields"]
		assert.Equal(t, processors[target], fields)
	}

}

// MockDynamicComm is used in tests.
type MockDynamicComm struct {
	context.Context
	providerDataChan chan providerData
}

// AddOrUpdate adds or updates a current mapping.
func (t *MockDynamicComm) AddOrUpdate(id string, priority int, mapping map[string]interface{}, processors []map[string]interface{}) error {
	t.providerDataChan <- providerData{
		id,
		mapping,
		processors,
	}
	return nil
}

// Remove
func (t *MockDynamicComm) Remove(id string) {
}

type podMeta struct{}

// Generate generates pod metadata from a resource object
// Metadata map is in the following form:
// {
// 	  "kubernetes": {},
//    "some.ecs.field": "asdf"
// }
// All Kubernetes fields that need to be stored under kubernetes. prefix are populated by
// GenerateK8s method while fields that are part of ECS are generated by GenerateECS method
func (p *podMeta) Generate(obj kubernetes.Resource, opts ...metadata.FieldOptions) common.MapStr {
	ecsFields := p.GenerateECS(obj)
	meta := common.MapStr{
		"kubernetes": p.GenerateK8s(obj, opts...),
	}
	meta.DeepUpdate(ecsFields)
	return meta
}

// GenerateECS generates pod ECS metadata from a resource object
func (p *podMeta) GenerateECS(obj kubernetes.Resource) common.MapStr {
	return common.MapStr{
		"orchestrator": common.MapStr{
			"cluster": common.MapStr{
				"name": "devcluster",
				"url":  "8.8.8.8:9090",
			},
		},
	}
}

// GenerateK8s generates pod metadata from a resource object
func (p *podMeta) GenerateK8s(obj kubernetes.Resource, opts ...metadata.FieldOptions) common.MapStr {
	k8sPod := obj.(*kubernetes.Pod)
	return common.MapStr{
		"namespace": k8sPod.GetNamespace(),
		"pod": common.MapStr{
			"uid":  string(k8sPod.GetUID()),
			"name": k8sPod.GetName(),
			"ip":   k8sPod.Status.PodIP,
		},
		"labels": common.MapStr{
			"foo": "bar",
		},
		"annotations": common.MapStr{
			"app": "production",
		},
	}
}

// GenerateFromName generates pod metadata from a node name
func (p *podMeta) GenerateFromName(name string, opts ...metadata.FieldOptions) common.MapStr {
	return nil
}
