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
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
)

func TestNamespace_Generate(t *testing.T) {
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	name := "obj"
	tests := []struct {
		input  kubernetes.Resource
		output common.MapStr
		name   string
	}{
		{
			name: "test simple object",
			input: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					UID:  types.UID(uid),
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			},
			// Use this for 8.0
			/*output: common.MapStr{
				"namespace": common.MapStr{
					"name": "obj",
					"uid":  uid,
					"labels": common.MapStr{
						"foo": "bar",
					},
				},
			},*/
			output: common.MapStr{
				"namespace":     "obj",
				"namespace_uid": uid,
				"namespace_labels": common.MapStr{
					"foo": "bar",
				},
			},
		},
	}

	cfg := common.NewConfig()
	metagen := NewNamespaceMetadataGenerator(cfg, nil)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}

func TestNamespace_GenerateFromName(t *testing.T) {
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	name := "obj"
	tests := []struct {
		input  kubernetes.Resource
		output common.MapStr
		name   string
	}{
		{
			name: "test simple object",
			input: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					UID:  types.UID(uid),
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			},
			// Use this for 8.0
			/*
				output: common.MapStr{
					"namespace": common.MapStr{
						"name": "obj",
						"uid":  uid,
						"labels": common.MapStr{
							"foo": "bar",
						},
					},
				},*/
			output: common.MapStr{
				"namespace":     "obj",
				"namespace_uid": uid,
				"namespace_labels": common.MapStr{
					"foo": "bar",
				},
			},
		},
	}

	for _, test := range tests {
		cfg := common.NewConfig()
		namespaces := cache.NewStore(cache.MetaNamespaceKeyFunc)
		namespaces.Add(test.input)
		metagen := NewNamespaceMetadataGenerator(cfg, namespaces)

		accessor, err := meta.Accessor(test.input)
		require.Nil(t, err)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.GenerateFromName(fmt.Sprint(accessor.GetName())))
		})
	}
}
