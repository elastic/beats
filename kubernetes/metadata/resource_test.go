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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-ucfg"
)

const (
	uid       = "005f3b90-4b9d-12f8-acf0-31020a840133"
	defaultNs = "default"
	name      = "obj"
)

func TestResource_Generate(t *testing.T) {
	boolean := true
	tests := []struct {
		input  kubernetes.Resource
		output mapstr.M
		name   string
	}{
		{
			name: "test simple object",
			input: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					UID:       types.UID(uid),
					Namespace: defaultNs,
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
			},
			output: mapstr.M{
				"kubernetes": mapstr.M{
					"pod": mapstr.M{
						"name": name,
						"uid":  uid,
					},
					"labels": mapstr.M{
						"foo": "bar",
					},
					"namespace": defaultNs,
				},
			},
		},
		{
			name: "test object with owner reference",
			input: &v1.Pod{
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
			},
			output: mapstr.M{
				"kubernetes": mapstr.M{
					"pod": mapstr.M{
						"name": name,
						"uid":  uid,
					},
					"labels": mapstr.M{
						"foo": "bar",
					},
					"namespace": defaultNs,
					"deployment": mapstr.M{
						"name": "owner",
					},
				},
			},
		},
	}

	var cfg Config
	err := ucfg.New().Unpack(&cfg)
	require.NoError(t, err)
	metagen := &Resource{
		config: &cfg,
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.output, metagen.Generate("pod", test.input))
		})
	}
}
