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

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/elastic/beats/libbeat/common"
)

func TestPodMetadata(t *testing.T) {
	UID := "005f3b90-4b9d-12f8-acf0-31020a840133"
	Deployment := "Deployment"
	test := "test"
	ReplicaSet := "ReplicaSet"
	StatefulSet := "StatefulSet"
	True := true
	False := false
	tests := []struct {
		name string
		pod  *Pod
		meta common.MapStr
	}{
		{
			name: "standalone Pod",
			pod: &Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    map[string]string{"a.key": "foo", "a": "bar"},
					UID:       types.UID(UID),
					Namespace: test,
				},
				Spec: v1.PodSpec{
					NodeName: test,
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
		},
		{
			name: "Deployment + Replicaset owned Pod",
			pod: &Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
					UID:    types.UID(UID),
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       Deployment,
							Name:       test,
							Controller: &True,
						},
						{
							Kind:       ReplicaSet,
							Name:       ReplicaSet,
							Controller: &False,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: test,
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
		},
		{
			name: "StatefulSet + Deployment owned Pod",
			pod: &Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
					UID:    types.UID(UID),
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       Deployment,
							Name:       test,
							Controller: &False,
						},
						{
							Kind:       ReplicaSet,
							Name:       ReplicaSet,
							Controller: &True,
						},
						{
							Kind:       StatefulSet,
							Name:       StatefulSet,
							Controller: &True,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: test,
				},
			},
			meta: common.MapStr{
				"pod": common.MapStr{
					"name": "",
					"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
				},
				"node":        common.MapStr{"name": "test"},
				"labels":      common.MapStr{"a": common.MapStr{"value": "bar", "key": "foo"}},
				"replicaset":  common.MapStr{"name": "ReplicaSet"},
				"statefulset": common.MapStr{"name": "StatefulSet"},
			},
		},
		{
			name: "empty owner reference Pod",
			pod: &Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels:          map[string]string{"a.key": "foo", "a": "bar"},
					UID:             types.UID(UID),
					OwnerReferences: []metav1.OwnerReference{{}},
					Namespace:       test,
				},
				Spec: v1.PodSpec{
					NodeName: test,
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
		},
	}

	for _, test := range tests {
		config, err := common.NewConfigFrom(map[string]interface{}{
			"labels.dedot":        false,
			"annotations.dedot":   false,
			"include_annotations": []string{"b", "b.key"},
		})

		metaGen, err := NewMetaGenerator(config)
		if err != nil {
			t.Fatalf("case %q failed: %s", test.name, err.Error())
		}
		assert.Equal(t, metaGen.PodMetadata(test.pod), test.meta, "test failed for case %q", test.name)
	}
}

func TestPodMetadataDeDot(t *testing.T) {
	UID := "005f3b90-4b9d-12f8-acf0-31020a840133"
	Deployment := "Deployment"
	test := "test"
	ReplicaSet := "ReplicaSet"
	StatefulSet := "StatefulSet"
	True := true
	False := false
	tests := []struct {
		pod  *Pod
		meta common.MapStr
	}{
		{
			pod: &Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"a.key": "foo", "a": "bar"},
					UID:         types.UID(UID),
					Namespace:   test,
					Annotations: map[string]string{"b.key": "foo", "b": "bar"},
				},
				Spec: v1.PodSpec{
					NodeName: test,
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
		},
		{
			pod: &Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
					UID:    types.UID(UID),
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       Deployment,
							Name:       test,
							Controller: &True,
						},
						{
							Kind:       ReplicaSet,
							Name:       ReplicaSet,
							Controller: &False,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: test,
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
		},
		{
			pod: &Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
					UID:    types.UID(UID),
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       Deployment,
							Name:       test,
							Controller: &False,
						},
						{
							Kind:       ReplicaSet,
							Name:       ReplicaSet,
							Controller: &True,
						},
						{
							Kind:       StatefulSet,
							Name:       StatefulSet,
							Controller: &True,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: test,
				},
			},
			meta: common.MapStr{
				"pod": common.MapStr{
					"name": "",
					"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
				},
				"node":        common.MapStr{"name": "test"},
				"labels":      common.MapStr{"a": "bar", "a_key": "foo"},
				"replicaset":  common.MapStr{"name": "ReplicaSet"},
				"statefulset": common.MapStr{"name": "StatefulSet"},
			},
		},
	}

	for _, test := range tests {
		config, err := common.NewConfigFrom(map[string]interface{}{
			"include_annotations": []string{"b", "b.key"},
		})
		metaGen, err := NewMetaGenerator(config)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, metaGen.PodMetadata(test.pod), test.meta)
	}
}
