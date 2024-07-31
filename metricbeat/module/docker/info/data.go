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

package info

import (
	"github.com/docker/docker/api/types/system"

	"github.com/elastic/beats/v7/libbeat/common"
)

<<<<<<< HEAD
func eventMapping(info *types.Info) common.MapStr {
	event := common.MapStr{
=======
func eventMapping(info *system.Info) mapstr.M {
	event := mapstr.M{
>>>>>>> 3c65545078 (Upgrad elastic-agent-system-metrics to v0.10.7. (#40397))
		"id": info.ID,
		"containers": common.MapStr{
			"total":   info.Containers,
			"running": info.ContainersRunning,
			"paused":  info.ContainersPaused,
			"stopped": info.ContainersStopped,
		},
		"images": info.Images,
	}

	return event
}
