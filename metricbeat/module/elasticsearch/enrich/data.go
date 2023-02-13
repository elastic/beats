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

package enrich

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/joeshaw/multierror"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"node_id": c.Str("node_id"),
		"queue": s.Object{
			"size": c.Int("queue_size"),
		},
		"remote_requests": s.Object{
			"current": c.Int("remote_requests_current"),
			"total":   c.Int("remote_requests_total"),
		},
		"executed_searches": s.Object{
			"total": c.Int("executed_searches_total"),
		},
	}

	task = s.Schema{
		"id":     c.Int("id"),
		"type":   c.Str("type"),
		"action": c.Str("action"),
		"time": s.Object{
			"start": s.Object{
				"ms": c.Int("start_time_in_millis"),
			},
			"running": s.Object{
				"nano": c.Int("running_time_in_nanos"),
			},
		},
		"cancellable":    c.Bool("cancellable"),
		"parent_task_id": c.Str("parent_task_id"),
	}
)

type response struct {
	ExecutingPolicies []map[string]interface{} `json:"executing_policies"`
	CoordinatorStats  []map[string]interface{} `json:"coordinator_stats"`
}

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool) error {
	var data response
	err := json.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("failure parsing Elasticsearch Enrich Stats API response: %w", err)
	}

	var errs multierror.Errors
	for _, stat := range data.CoordinatorStats {

		event := mb.Event{}

		event.ModuleFields = mapstr.M{}
		_, _ = event.ModuleFields.Put("cluster.name", info.ClusterName)
		_, _ = event.ModuleFields.Put("cluster.id", info.ClusterID)

		fields, err := schema.Apply(stat)
		if err != nil {
			errs = append(errs, fmt.Errorf("failure applying enrich coordinator stats schema: %w", err))
			continue
		}

		nodeID, err := fields.GetValue("node_id")
		if err != nil {
			errs = append(errs, fmt.Errorf("failure retrieving node ID from Elasticsearch Enrich Stats API response: %w", err))
		}

		_, _ = event.ModuleFields.Put("node.id", nodeID)
		_ = fields.Delete("node_id")

		event.MetricSetFields = fields

		if isXpack {
			index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
			event.Index = index
		}

		r.Event(event)
	}

	for _, policy := range data.ExecutingPolicies {
		event := mb.Event{}

		event.ModuleFields = mapstr.M{}
		_, _ = event.ModuleFields.Put("cluster.name", info.ClusterName)
		_, _ = event.ModuleFields.Put("cluster.id", info.ClusterID)
		event.MetricSetFields = mapstr.M{}

		policyName, ok := policy["name"]
		if !ok {
			// No name found for policy. Ignore because all policies require a name
			errs = append(errs, errors.New("found an 'executing policy' without a name. Omitting."))
			continue
		}

		taskData, ok := policy["task"]
		if !ok {
			// No task found for policy. Ignore because all policies must contain a task
			errs = append(errs, errors.New("found an 'executing policy' without a task. Omitting."))
			continue
		}

		taskMapstr, ok := taskData.(map[string]interface{})
		if !ok {
			errs = append(errs, errors.New("error trying to convert interface of task data into a map"))
			continue
		}

		fields, err := task.Apply(taskMapstr)
		if err != nil {
			errs = append(errs, fmt.Errorf("failure applying enrich coordinator stats schema: %w", err))
			continue
		}

		_, _ = event.MetricSetFields.Put("executing_policy.name", policyName)
		_, _ = event.MetricSetFields.Put("executing_policy.task", fields)

		// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
		// When using Agent, the index name is overwritten anyways.
		if isXpack {
			index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
			event.Index = index
		}

		r.Event(event)
	}

	return errs.Err()
}
