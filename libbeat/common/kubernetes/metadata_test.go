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

	v1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestPodMetadata(t *testing.T) {
	UID := "005f3b90-4b9d-12f8-acf0-31020a840133"
	Deployment := "Deployment"
	test := "test"
	ReplicaSet := "ReplicaSet"
	True := true
	False := false
	tests := []struct {
		pod    *Pod
		meta   common.MapStr
		config *common.Config
	}{
		{
			pod: &Pod{
				Metadata: &metav1.ObjectMeta{
					Labels:    map[string]string{"a.key": "foo", "a": "bar"},
					Uid:       &UID,
					Namespace: &test,
				},
				Spec: &v1.PodSpec{
					NodeName: &test,
				},
			},
			meta: common.MapStr{
				"pod": common.MapStr{
					"name": "",
					"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
				},
				"node":      common.MapStr{"name": "test"},
				"namespace": "test",
				"labels":    common.MapStr{"a": common.MapStr{"value": "bar", "key": "foo"}},
			},
			config: common.NewConfig(),
		},
		{
			pod: &Pod{
				Metadata: &metav1.ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
					Uid:    &UID,
					OwnerReferences: []*metav1.OwnerReference{
						{
							Kind:       &Deployment,
							Name:       &test,
							Controller: &True,
						},
						{
							Kind:       &ReplicaSet,
							Name:       &ReplicaSet,
							Controller: &False,
						},
					},
				},
				Spec: &v1.PodSpec{
					NodeName: &test,
				},
			},
			meta: common.MapStr{
				"pod": common.MapStr{
					"name": "",
					"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
				},
				"node":       common.MapStr{"name": "test"},
				"labels":     common.MapStr{"a": common.MapStr{"value": "bar", "key": "foo"}},
				"deployment": common.MapStr{"name": "test"},
			},
			config: common.NewConfig(),
		},
	}

	for _, test := range tests {
		metaGen, err := NewMetaGenerator(test.config)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, metaGen.PodMetadata(test.pod), test.meta)
	}
}

func TestPodMetadataDeDot(t *testing.T) {
	UID := "005f3b90-4b9d-12f8-acf0-31020a840133"
	Deployment := "Deployment"
	test := "test"
	ReplicaSet := "ReplicaSet"
	True := true
	False := false
	tests := []struct {
		pod    *Pod
		meta   common.MapStr
		config *common.Config
	}{
		{
			pod: &Pod{
				Metadata: &metav1.ObjectMeta{
					Labels:      map[string]string{"a.key": "foo", "a": "bar"},
					Uid:         &UID,
					Namespace:   &test,
					Annotations: map[string]string{"b.key": "foo", "b": "bar"},
				},
				Spec: &v1.PodSpec{
					NodeName: &test,
				},
			},
			meta: common.MapStr{
				"pod": common.MapStr{
					"name": "",
					"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
				},
				"node":        common.MapStr{"name": "test"},
				"namespace":   "test",
				"labels":      common.MapStr{"a": "bar", "a_key": "foo"},
				"annotations": common.MapStr{"b": "bar", "b_key": "foo"},
			},
			config: common.NewConfig(),
		},
		{
			pod: &Pod{
				Metadata: &metav1.ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
					Uid:    &UID,
					OwnerReferences: []*metav1.OwnerReference{
						{
							Kind:       &Deployment,
							Name:       &test,
							Controller: &True,
						},
						{
							Kind:       &ReplicaSet,
							Name:       &ReplicaSet,
							Controller: &False,
						},
					},
				},
				Spec: &v1.PodSpec{
					NodeName: &test,
				},
			},
			meta: common.MapStr{
				"pod": common.MapStr{
					"name": "",
					"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
				},
				"node":       common.MapStr{"name": "test"},
				"labels":     common.MapStr{"a": "bar", "a_key": "foo"},
				"deployment": common.MapStr{"name": "test"},
			},
			config: common.NewConfig(),
		},
	}

	for _, test := range tests {
		config, err := common.NewConfigFrom(map[string]interface{}{
			"labels.dedot":        true,
			"annotations.dedot":   true,
			"include_annotations": []string{"b", "b.key"},
		})
		metaGen, err := NewMetaGenerator(config)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, metaGen.PodMetadata(test.pod), test.meta)
	}
}
