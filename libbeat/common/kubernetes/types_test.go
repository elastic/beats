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
)

func TestPodContainerStatus_GetContainerID(t *testing.T) {
	tests := []struct {
		status *PodContainerStatus
		result string
	}{
		// Check to see if x://y is parsed to return y as the container id
		{
			status: &PodContainerStatus{
				Name:        "foobar",
				ContainerID: "docker://abc",
				Image:       "foobar:latest",
			},
			result: "abc",
		},
		// Check to see if x://y is not the format then "" is returned
		{
			status: &PodContainerStatus{
				Name:        "foobar",
				ContainerID: "abc",
				Image:       "foobar:latest",
			},
			result: "",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.status.GetContainerID(), test.result)
	}
}

func TestPodContainerStatus_GetContainerIDWithRuntime(t *testing.T) {
	tests := []struct {
		status *PodContainerStatus
		result string
	}{
		// Check to see if x://y is parsed to return x as the runtime
		{
			status: &PodContainerStatus{
				Name:        "foobar",
				ContainerID: "docker://abc",
				Image:       "foobar:latest",
			},
			result: "docker",
		},
		// Check to see if x://y is not the format then "" is returned
		{
			status: &PodContainerStatus{
				Name:        "foobar",
				ContainerID: "abc",
				Image:       "foobar:latest",
			},
			result: "",
		},
	}

	for _, test := range tests {
		_, runtime := test.status.GetContainerIDWithRuntime()
		assert.Equal(t, runtime, test.result)
	}
}
