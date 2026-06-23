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

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNamespace_Generate(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	tests := []struct {
		input  kubernetes.Resource
		output mapstr.M
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
					Annotations: map[string]string{
						"spam": "baz",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			},
			// Use this for 8.0
			/*
				output: mapstr.M{
					"kubernetes": mapstr.M{
						"namespace": mapstr.M{
							"name": name,
							"uid":  uid,
							"labels": mapstr.M{
								"foo": "bar",
							},
						},
					},
				},*/
			output: mapstr.M{"kubernetes": mapstr.M{
				"namespace":     name,
				"namespace_uid": uid,
				"namespace_labels": mapstr.M{
					"foo": "bar",
				},
				"namespace_annotations": mapstr.M{
					"spam": "baz",
				},
			}},
		},
	}

	cfg, err := config.NewConfigFrom(Config{
		IncludeAnnotations: []string{"spam"},
	})
	if err != nil {
		t.Fatalf("Could not merge configs")
	}

	metagen := NewNamespaceMetadataGenerator(cfg, nil, client)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}

func TestNamespace_GenerateFromName(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	tests := []struct {
		input  kubernetes.Resource
		output mapstr.M
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
					Annotations: map[string]string{
						"spam": "baz",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			},
			// Use this for 8.0
			/*
				output: mapstr.M{
					"namespace": mapstr.M{
						"name": name,
						"uid":  uid,
						"labels": mapstr.M{
							"foo": "bar",
						},
					},
				},*/
			output: mapstr.M{
				"namespace":     name,
				"namespace_uid": uid,
				"namespace_labels": mapstr.M{
					"foo": "bar",
				},
				"namespace_annotations": mapstr.M{
					"spam": "baz",
				},
			},
		},
	}

	for _, test := range tests {
		cfg, err := config.NewConfigFrom(Config{
			IncludeAnnotations: []string{"spam"},
		})
		if err != nil {
			t.Fatalf("Could not merge configs")
		}

		namespaces := cache.NewStore(cache.MetaNamespaceKeyFunc)
		err = namespaces.Add(test.input)
		require.NoError(t, err)
		metagen := NewNamespaceMetadataGenerator(cfg, namespaces, client)

		accessor, err := meta.Accessor(test.input)
		require.Nil(t, err)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.GenerateFromName(fmt.Sprint(accessor.GetName())))
		})
	}
}
