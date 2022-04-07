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

func TestNode_Generate(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	name := "obj"
	tests := []struct {
		input  kubernetes.Resource
		output common.MapStr
		name   string
	}{
		{
			name: "test simple object",
			input: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					UID:  types.UID(uid),
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"key1": "value1",
						"key2": "value2",
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
			output: common.MapStr{"kubernetes": common.MapStr{
				"node": common.MapStr{
					"name":     "obj",
					"uid":      uid,
					"hostname": "node1",
				},
				"labels": common.MapStr{
					"foo": "bar",
				},
				"annotations": common.MapStr{
					"key2": "value2",
				},
			}},
		},
	}

	cfg, _ := common.NewConfigFrom(Config{
		IncludeAnnotations: []string{"key2"},
	})
	metagen := NewNodeMetadataGenerator(cfg, nil, client)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}

func TestNode_GenerateFromName(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	name := "obj"
	tests := []struct {
		input  kubernetes.Resource
		output common.MapStr
		name   string
	}{
		{
			name: "test simple object",
			input: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					UID:  types.UID(uid),
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"key": "value",
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
			output: common.MapStr{
				"node": common.MapStr{
					"name":     "obj",
					"uid":      uid,
					"hostname": "node1",
				},
				"labels": common.MapStr{
					"foo": "bar",
				},
				"annotations": common.MapStr{
					"key": "value",
				},
			},
		},
	}

	for _, test := range tests {
		cfg, _ := common.NewConfigFrom(Config{
			IncludeAnnotations: []string{"key"},
		})
		nodes := cache.NewStore(cache.MetaNamespaceKeyFunc)
		nodes.Add(test.input)
		metagen := NewNodeMetadataGenerator(cfg, nodes, client)

		accessor, err := meta.Accessor(test.input)
		require.NoError(t, err)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.GenerateFromName(fmt.Sprint(accessor.GetName())))
		})
	}
}
