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

package healthcheck

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func eventsMapping(containers []types.Container, m *MetricSet) []common.MapStr {
	var events []common.MapStr
	for _, container := range containers {
		event := eventMapping(&container, m)
		if event != nil {
			events = append(events, event)
		}
	}
	return events
}

func eventMapping(cont *types.Container, m *MetricSet) common.MapStr {
	if !hasHealthCheck(cont.Status) {
		return nil
	}

	container, err := m.dockerClient.ContainerInspect(context.TODO(), cont.ID)
	if err != nil {
		logp.Err("Error inpsecting container %v: %v", cont.ID, err)
		return nil
	}
	lastEvent := len(container.State.Health.Log) - 1

	// Checks if a healthcheck already happened
	if lastEvent < 0 {
		return nil
	}

	return common.MapStr{
		mb.ModuleDataKey: common.MapStr{
			"container": docker.NewContainer(cont, m.dedot).ToMapStr(),
		},
		"status":        container.State.Health.Status,
		"failingstreak": container.State.Health.FailingStreak,
		"event": common.MapStr{
			"start_date": common.Time(container.State.Health.Log[lastEvent].Start),
			"end_date":   common.Time(container.State.Health.Log[lastEvent].End),
			"exit_code":  container.State.Health.Log[lastEvent].ExitCode,
			"output":     container.State.Health.Log[lastEvent].Output,
		},
	}
}

// hasHealthCheck detects if healthcheck is available for container
func hasHealthCheck(status string) bool {
	return strings.Contains(status, "(") && strings.Contains(status, ")")
}
