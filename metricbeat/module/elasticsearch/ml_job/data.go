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

package ml_job

import (
	"encoding/json"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/helper/elastic"

	"github.com/menderesk/beats/v7/libbeat/common"
	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"id":    c.Str("job_id"),
		"state": c.Str("state"),
		"data_counts": c.Dict("data_counts", s.Schema{
			"processed_record_count": c.Int("processed_record_count"),
			"invalid_date_count":     c.Int("invalid_date_count"),
		}),
		"model_size": c.Dict("model_size_stats", s.Schema{
			"memory_status": c.Str("memory_status"),
		}),
		"forecasts_stats": c.Dict("forecasts_stats", s.Schema{
			"total": c.Int("total"),
		}),
	}
)

type jobsStruct struct {
	Jobs []map[string]interface{} `json:"jobs"`
}

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool) error {

	jobsData := &jobsStruct{}
	err := json.Unmarshal(content, jobsData)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch ML Job Stats API response")
	}

	var errs multierror.Errors
	for _, job := range jobsData.Jobs {

		if err := elastic.FixTimestampField(job, "data_counts.earliest_record_timestamp"); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := elastic.FixTimestampField(job, "data_counts.latest_record_timestamp"); err != nil {
			errs = append(errs, err)
			continue
		}

		event := mb.Event{}

		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", elasticsearch.ModuleName)

		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("cluster.name", info.ClusterName)
		event.ModuleFields.Put("cluster.id", info.ClusterID)

		if node, exists := job["node"]; exists {
			nodeHash := node.(map[string]interface{})
			event.ModuleFields.Put("node.id", nodeHash["id"])
			event.ModuleFields.Put("node.name", nodeHash["name"])
		}

		event.MetricSetFields, _ = schema.Apply(job)

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
