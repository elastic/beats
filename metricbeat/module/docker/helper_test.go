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

package docker

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestDeDotLabels(t *testing.T) {
	labels := map[string]string{
		"com.docker.swarm.task":      "",
		"com.docker.swarm.task.id":   "1",
		"com.docker.swarm.task.name": "foobar",
	}

	t.Run("dedot enabled", func(t *testing.T) {
		result := DeDotLabels(labels, true)
		assert.Equal(t, common.MapStr{
			"com_docker_swarm_task":      "",
			"com_docker_swarm_task_id":   "1",
			"com_docker_swarm_task_name": "foobar",
		}, result)
	})

	t.Run("dedot disabled", func(t *testing.T) {
		result := DeDotLabels(labels, false)
		assert.Equal(t, common.MapStr{
			"com": common.MapStr{
				"docker": common.MapStr{
					"swarm": common.MapStr{
						"task": common.MapStr{
							"value": "",
							"id":    "1",
							"name":  "foobar",
						},
					},
				},
			},
		}, result)
	})
}
