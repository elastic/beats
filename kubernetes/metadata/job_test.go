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
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestJob_Generate(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	boolean := true
	tests := []struct {
		input  kubernetes.Resource
		output mapstr.M
		name   string
	}{
		{
			name: "test simple object with owner",
			input: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: defaultNs,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps",
							Kind:       "CronJob",
							Name:       "nginx-job",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a840144",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Job",
					APIVersion: "v1",
				},
			},
			output: mapstr.M{
				"kubernetes": mapstr.M{
					"job": mapstr.M{
						"name": name,
						"uid":  uid,
					},
					"labels": mapstr.M{
						"foo": "bar",
					},
					"cronjob": mapstr.M{
						"name": "nginx-job",
					},
					"namespace": defaultNs,
				},
			},
		},
	}

	cfg := config.NewConfig()
	metagen := NewJobMetadataGenerator(cfg, nil, client)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate(test.input))
		})
	}
}

func TestJob_GenerateFromName(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	boolean := true
	tests := []struct {
		input  kubernetes.Resource
		output mapstr.M
		name   string
	}{
		{
			name: "test simple object with owner",
			input: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: defaultNs,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps",
							Kind:       "CronJob",
							Name:       "nginx-job",
							UID:        "005f3b90-4b9d-12f8-acf0-31020a840144",
							Controller: &boolean,
						},
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Job",
					APIVersion: "v1",
				},
			},
			output: mapstr.M{
				"job": mapstr.M{
					"name": name,
					"uid":  uid,
				},
				"labels": mapstr.M{
					"foo": "bar",
				},
				"cronjob": mapstr.M{
					"name": "nginx-job",
				},
				"namespace": defaultNs,
			},
		},
	}

	for _, test := range tests {
		cfg := config.NewConfig()
		jobs := cache.NewStore(cache.MetaNamespaceKeyFunc)
		err := jobs.Add(test.input)
		require.NoError(t, err)
		metagen := NewJobMetadataGenerator(cfg, jobs, client)

		accessor, err := meta.Accessor(test.input)
		require.NoError(t, err)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.GenerateFromName(fmt.Sprint(accessor.GetNamespace(), "/", accessor.GetName())))
		})
	}
}
