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

package metadata

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	conf "github.com/elastic/elastic-agent-libs/config"
)

var addResourceMetadata = GetDefaultResourceMetadataConfig()

func TestPod_Generate(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	namespace := "default"
	name := "obj"
	boolean := true
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-rs",
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps",
					Kind:       "Deployment",
					Name:       "nginx-deployment",
					UID:        "005f3b90-4b9d-12f8-acf0-31020a840144",
					Controller: &boolean,
				},
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.12",
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									Protocol:      v1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := client.AppsV1().ReplicaSets(namespace).Create(context.Background(), rs, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create k8s deployment: %v", err)
	}

	tests := []struct {
		input  kubernetes.Resource
		output common.MapStr
		name   string
	}{
		{
			name: "test simple object",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
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
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "obj",
						"uid":  uid,
						"ip":   "127.0.0.5",
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"annotations": common.MapStr{
						"app": "production",
					},
					"namespace": "default",
					"node": common.MapStr{
						"name": "testnode",
					},
				},
			},
		},
		{
			name: "test object with owner reference to Deployment",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"app": "production",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps",
							Kind:       "Deployment",
							Name:       "owner",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a840144",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "obj",
						"uid":  uid,
						"ip":   "127.0.0.5",
					},
					"namespace": "default",
					"deployment": common.MapStr{
						"name": "owner",
					},
					"node": common.MapStr{
						"name": "testnode",
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"annotations": common.MapStr{
						"app": "production",
					},
				},
			},
		},
		{
			name: "test object with owner reference to DaemonSet",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"app": "production",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps",
							Kind:       "DaemonSet",
							Name:       "owner",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a840144",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "obj",
						"uid":  uid,
						"ip":   "127.0.0.5",
					},
					"namespace": "default",
					"daemonset": common.MapStr{
						"name": "owner",
					},
					"node": common.MapStr{
						"name": "testnode",
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"annotations": common.MapStr{
						"app": "production",
					},
				},
			},
		},
		{
			name: "test object with owner reference to Job",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"app": "production",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "batch/v1",
							Kind:       "Job",
							Name:       "owner",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a840144",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "obj",
						"uid":  uid,
						"ip":   "127.0.0.5",
					},
					"namespace": "default",
					"job": common.MapStr{
						"name": "owner",
					},
					"node": common.MapStr{
						"name": "testnode",
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"annotations": common.MapStr{
						"app": "production",
					},
				},
			},
		},
		{
			name: "test object with owner reference to replicaset",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"app": "production",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps",
							Kind:       "ReplicaSet",
							Name:       "nginx-rs",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a8409087",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "obj",
						"uid":  uid,
						"ip":   "127.0.0.5",
					},
					"namespace": "default",
					"deployment": common.MapStr{
						"name": "nginx-deployment",
					},
					"replicaset": common.MapStr{
						"name": "nginx-rs",
					},
					"node": common.MapStr{
						"name": "testnode",
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"annotations": common.MapStr{
						"app": "production",
					},
				},
			},
		},
		{
			name: "test object with owner reference to replicaset honors annotations.dedot: false",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"k8s.app": "production",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps",
							Kind:       "ReplicaSet",
							Name:       "nginx-rs",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a8409087",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "obj",
						"uid":  uid,
						"ip":   "127.0.0.5",
					},
					"namespace": "default",
					"deployment": common.MapStr{
						"name": "nginx-deployment",
					},
					"replicaset": common.MapStr{
						"name": "nginx-rs",
					},
					"node": common.MapStr{
						"name": "testnode",
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"annotations": common.MapStr{
						"k8s": common.MapStr{"app": "production"},
					},
				},
			},
		},
	}

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"include_annotations": []string{"app", "k8s.app"},
		"annotations.dedot":   false,
	})
	assert.NoError(t, err)

	metagen := NewPodMetadataGenerator(config, nil, client, nil, nil, addResourceMetadata)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}

func TestPod_GenerateFromName(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	namespace := "default"
	name := "obj"
	boolean := true
	tests := []struct {
		input  kubernetes.Resource
		output common.MapStr
		name   string
	}{
		{
			name: "test simple object",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"k8s.app": "production",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			output: common.MapStr{
				"pod": common.MapStr{
					"name": "obj",
					"uid":  uid,
					"ip":   "127.0.0.5",
				},
				"namespace": "default",
				"node": common.MapStr{
					"name": "testnode",
				},
				"labels": common.MapStr{
					"foo": "bar",
				},
				"annotations": common.MapStr{
					"k8s_app": "production",
				},
			},
		},
		{
			name: "test object with owner reference",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"app": "production",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps",
							Kind:       "Deployment",
							Name:       "owner",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a840144",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			output: common.MapStr{
				"pod": common.MapStr{
					"name": "obj",
					"uid":  uid,
					"ip":   "127.0.0.5",
				},
				"namespace": "default",
				"deployment": common.MapStr{
					"name": "owner",
				},
				"node": common.MapStr{
					"name": "testnode",
				},
				"labels": common.MapStr{
					"foo": "bar",
				},
				"annotations": common.MapStr{
					"app": "production",
				},
			},
		},
	}

	for _, test := range tests {
		config, err := conf.NewConfigFrom(map[string]interface{}{
			"include_annotations": []string{"app", "k8s.app"},
		})
		assert.NoError(t, err)
		pods := cache.NewStore(cache.MetaNamespaceKeyFunc)
		pods.Add(test.input)
		metagen := NewPodMetadataGenerator(config, pods, client, nil, nil, addResourceMetadata)

		accessor, err := meta.Accessor(test.input)
		require.NoError(t, err)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.GenerateFromName(fmt.Sprint(accessor.GetNamespace(), "/", accessor.GetName())))
		})
	}
}

func TestPod_GenerateWithNodeNamespace(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	namespace := "default"
	name := "obj"
	tests := []struct {
		input     kubernetes.Resource
		node      kubernetes.Resource
		namespace kubernetes.Resource
		output    common.MapStr
		name      string
	}{
		{
			name: "test simple object",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
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
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testnode",
					UID:  types.UID(uid),
					Labels: map[string]string{
						"nodekey": "nodevalue",
					},
					Annotations: map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: "v1",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{{Type: v1.NodeHostName, Address: "node1"}},
				},
			},
			namespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					UID:  types.UID(uid),
					Labels: map[string]string{
						"nskey": "nsvalue",
					},
					Annotations: map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			},
			output: common.MapStr{"kubernetes": common.MapStr{
				"pod": common.MapStr{
					"name": "obj",
					"uid":  uid,
					"ip":   "127.0.0.5",
				},
				"namespace":     "default",
				"namespace_uid": uid,
				"namespace_labels": common.MapStr{
					"nskey": "nsvalue",
				},
				"node": common.MapStr{
					"name": "testnode",
					"uid":  uid,
					"labels": common.MapStr{
						"nodekey": "nodevalue",
					},
					"hostname": "node1",
				},
				"labels": common.MapStr{
					"foo": "bar",
				},
				"annotations": common.MapStr{
					"app": "production",
				},
			}},
		},
	}

	for _, test := range tests {
		config, err := conf.NewConfigFrom(map[string]interface{}{
			"include_annotations": []string{"app"},
		})
		assert.NoError(t, err)
		pods := cache.NewStore(cache.MetaNamespaceKeyFunc)
		pods.Add(test.input)

		nodes := cache.NewStore(cache.MetaNamespaceKeyFunc)
		nodes.Add(test.node)
		nodeMeta := NewNodeMetadataGenerator(config, nodes, client)

		namespaces := cache.NewStore(cache.MetaNamespaceKeyFunc)
		namespaces.Add(test.namespace)
		nsMeta := NewNamespaceMetadataGenerator(config, namespaces, client)

		metagen := NewPodMetadataGenerator(config, pods, client, nodeMeta, nsMeta, addResourceMetadata)
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}

func TestPod_GenerateWithNodeNamespaceWithAddResourceConfig(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	namespace := "default"
	name := "obj"
	boolean := true

	tests := []struct {
		input     kubernetes.Resource
		node      kubernetes.Resource
		namespace kubernetes.Resource
		output    common.MapStr
		name      string
	}{
		{
			name: "test simple object",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"app.kubernetes.io/component": "exporter",
					},
					Annotations: map[string]string{
						"app": "production",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps",
							Kind:       "ReplicaSet",
							Name:       "nginx-rs",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a8409087",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},

				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
				Status: v1.PodStatus{PodIP: "127.0.0.5"},
			},
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testnode",
					UID:  types.UID(uid),
					Labels: map[string]string{
						"nodekey":  "nodevalue",
						"nodekey2": "nodevalue2",
					},
					Annotations: map[string]string{
						"node.annotation": "node.value",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: "v1",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{{Type: v1.NodeHostName, Address: "node1"}},
				},
			},
			namespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					UID:  types.UID(uid),
					Labels: map[string]string{
						"app.kubernetes.io/name": "kube-state-metrics",
						"nskey2":                 "nsvalue2",
					},
					Annotations: map[string]string{
						"ns.annotation": "ns.value",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			},
			output: common.MapStr{"kubernetes": common.MapStr{
				"pod": common.MapStr{
					"name": "obj",
					"uid":  uid,
					"ip":   "127.0.0.5",
				},
				"namespace":     "default",
				"namespace_uid": uid,
				"namespace_labels": common.MapStr{
					"app_kubernetes_io/name": "kube-state-metrics",
				},
				"namespace_annotations": common.MapStr{
					"ns_annotation": "ns.value",
				},
				"node": common.MapStr{
					"name": "testnode",
					"uid":  uid,
					"labels": common.MapStr{
						"nodekey2": "nodevalue2",
					},
					"hostname": "node1",
					"annotations": common.MapStr{
						"node_annotation": "node.value",
					},
				},
				"labels": common.MapStr{
					"app_kubernetes_io/component": "exporter",
				},
				"annotations": common.MapStr{
					"app": "production",
				},
				"replicaset": common.MapStr{
					"name": "nginx-rs",
				},
			}},
		},
	}

	for _, test := range tests {
		config, err := conf.NewConfigFrom(map[string]interface{}{
			"include_annotations": []string{"app"},
		})

		assert.NoError(t, err)

		namespaceConfig, _ := conf.NewConfigFrom(map[string]interface{}{
			"include_labels":      []string{"app.kubernetes.io/name"},
			"include_annotations": []string{"ns.annotation"},
		})
		nodeConfig, _ := conf.NewConfigFrom(map[string]interface{}{
			"include_labels":      []string{"nodekey2"},
			"include_annotations": []string{"node.annotation"},
		})
		metaConfig := AddResourceMetadataConfig{
			Namespace:  namespaceConfig,
			Node:       nodeConfig,
			Deployment: false,
		}

		pods := cache.NewStore(cache.MetaNamespaceKeyFunc)
		pods.Add(test.input)

		nodes := cache.NewStore(cache.MetaNamespaceKeyFunc)
		nodes.Add(test.node)
		nodeMeta := NewNodeMetadataGenerator(nodeConfig, nodes, client)

		namespaces := cache.NewStore(cache.MetaNamespaceKeyFunc)
		namespaces.Add(test.namespace)
		nsMeta := NewNamespaceMetadataGenerator(namespaceConfig, namespaces, client)

		metagen := NewPodMetadataGenerator(config, pods, client, nodeMeta, nsMeta, &metaConfig)
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}
