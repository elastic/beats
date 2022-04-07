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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/kubernetes"
)

func TestService_Generate(t *testing.T) {
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
			input: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app":   "istiod",
						"istio": "pilot",
					},
				},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"service": common.MapStr{
						"name": "obj",
						"uid":  uid,
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"selectors": common.MapStr{
						"app":   "istiod",
						"istio": "pilot",
					},
					"namespace": "default",
				},
			},
		},
		{
			name: "test object with owner reference",
			input: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
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
					Kind:       "Service",
					APIVersion: "v1",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app":   "istiod",
						"istio": "pilot",
					},
				},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"service": common.MapStr{
						"name": "obj",
						"uid":  uid,
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"selectors": common.MapStr{
						"app":   "istiod",
						"istio": "pilot",
					},
					"namespace": "default",
					"deployment": common.MapStr{
						"name": "owner",
					},
				},
			},
		},
	}

	cfg := common.NewConfig()
	metagen := NewServiceMetadataGenerator(cfg, nil, nil, client)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}

func TestService_GenerateFromName(t *testing.T) {
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
			input: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
			},
			output: common.MapStr{
				"service": common.MapStr{
					"name": "obj",
					"uid":  uid,
				},
				"labels": common.MapStr{
					"foo": "bar",
				},
				"namespace": "default",
			},
		},
		{
			name: "test object with owner reference",
			input: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
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
					Kind:       "Service",
					APIVersion: "v1",
				},
			},
			output: common.MapStr{
				"service": common.MapStr{
					"name": "obj",
					"uid":  uid,
				},
				"labels": common.MapStr{
					"foo": "bar",
				},
				"namespace": "default",
				"deployment": common.MapStr{
					"name": "owner",
				},
			},
		},
	}

	for _, test := range tests {
		cfg := common.NewConfig()
		services := cache.NewStore(cache.MetaNamespaceKeyFunc)
		services.Add(test.input)
		metagen := NewServiceMetadataGenerator(cfg, services, nil, client)

		accessor, err := meta.Accessor(test.input)
		require.NoError(t, err)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.GenerateFromName(fmt.Sprint(accessor.GetNamespace(), "/", accessor.GetName())))
		})
	}
}

func TestService_GenerateWithNamespace(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	namespace := "default"
	name := "obj"
	tests := []struct {
		input     kubernetes.Resource
		namespace kubernetes.Resource
		output    common.MapStr
		name      string
	}{
		{
			name: "test simple object",
			input: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: namespace,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
			},
			namespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					UID:  types.UID(uid),
					Labels: map[string]string{
						"nskey": "nsvalue",
					},
					Annotations: map[string]string{
						"ns.annotation": "value",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			},
			output: common.MapStr{
				"kubernetes": common.MapStr{
					"service": common.MapStr{
						"name": "obj",
						"uid":  uid,
					},
					"labels": common.MapStr{
						"foo": "bar",
					},
					"namespace":     "default",
					"namespace_uid": uid,
					"namespace_labels": common.MapStr{
						"nskey": "nsvalue",
					},
					"namespace_annotations": common.MapStr{
						"ns_annotation": "value",
					},
				},
			},
		},
	}

	for _, test := range tests {
		nsConfig, _ := common.NewConfigFrom(map[string]interface{}{
			"include_annotations": []string{"ns.annotation"},
		})
		services := cache.NewStore(cache.MetaNamespaceKeyFunc)
		services.Add(test.input)

		namespaces := cache.NewStore(cache.MetaNamespaceKeyFunc)
		namespaces.Add(test.namespace)
		nsMeta := NewNamespaceMetadataGenerator(nsConfig, namespaces, client)

		metagen := NewServiceMetadataGenerator(nsConfig, services, nsMeta, client)
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}
