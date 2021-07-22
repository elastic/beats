// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
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

	data := generatePodData(pod, &Config{})

	mapping := map[string]interface{}{
		"namespace": pod.GetNamespace(),
		"pod": map[string]interface{}{
			"uid":  string(pod.GetUID()),
			"name": pod.GetName(),
			"labels": common.MapStr{
				"foo": "bar",
			},
			"annotations": common.MapStr{
				"app": "production",
			},
			"ip": pod.Status.PodIP,
		},
	}
	processors := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": mapping,
				"target": "kubernetes",
			},
		},
	}

	assert.Equal(t, string(pod.GetUID()), data.uid)
	assert.Equal(t, mapping, data.mapping)
	assert.Equal(t, processors, data.processors)
}

func TestGenerateContainerPodData(t *testing.T) {
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

	providerDataChan := make(chan providerData, 1)

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
	comm := MockDynamicComm{
		context.TODO(),
		providerDataChan,
	}
	generateContainerData(
		&comm,
		pod,
		containers,
		containerStatuses,
		&Config{})

	mapping := map[string]interface{}{
		"namespace": pod.GetNamespace(),
		"pod": map[string]interface{}{
			"uid":  string(pod.GetUID()),
			"name": pod.GetName(),
			"labels": common.MapStr{
				"foo": "bar",
			},
			"ip": pod.Status.PodIP,
		},
		"container": map[string]interface{}{
			"id":      "asdfghdeadbeef",
			"name":    "nginx",
			"image":   "nginx:1.120",
			"runtime": "crio",
		},
	}

	processors := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": mapping,
				"target": "kubernetes",
			},
		},
	}

	cuid := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), "nginx")
	data := <-providerDataChan
	assert.Equal(t, cuid, data.uid)
	assert.Equal(t, mapping, data.mapping)
	assert.Equal(t, processors, data.processors)

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
