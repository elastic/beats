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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestReplicaset_Generate(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	boolean := true
	tests := []struct {
		input  kubernetes.Resource
		output mapstr.M
		name   string
	}{
		{
			name: "test simple object with owner",
			input: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nginx-rs",
					Namespace: defaultNs,
					UID:       uid,
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
			},
			output: mapstr.M{
				"kubernetes": mapstr.M{
					"replicaset": mapstr.M{
						"name": "nginx-rs",
						"uid":  uid,
					},
					"deployment": mapstr.M{
						"name": "nginx-deployment",
					},
					"namespace": defaultNs,
				},
			},
		},
	}

	cfg := config.NewConfig()
	metagen := NewReplicasetMetadataGenerator(cfg, nil, client)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}

func TestReplicase_GenerateFromName(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	boolean := true
	tests := []struct {
		input  kubernetes.Resource
		output mapstr.M
		name   string
	}{
		{
			name: "test simple object with owner",
			input: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nginx-rs",
					Namespace: defaultNs,
					UID:       uid,
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
			},
			output: mapstr.M{
				"replicaset": mapstr.M{
					"name": "nginx-rs",
					"uid":  uid,
				},
				"deployment": mapstr.M{
					"name": "nginx-deployment",
				},
				"namespace": defaultNs,
			},
		},
	}

	for _, test := range tests {
		cfg := config.NewConfig()
		replicasets := cache.NewStore(cache.MetaNamespaceKeyFunc)
		err := replicasets.Add(test.input)
		require.NoError(t, err)
		metagen := NewReplicasetMetadataGenerator(cfg, replicasets, client)

		accessor, err := meta.Accessor(test.input)
		require.NoError(t, err)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.GenerateFromName(fmt.Sprint(accessor.GetNamespace(), "/", accessor.GetName())))
		})
	}
}
