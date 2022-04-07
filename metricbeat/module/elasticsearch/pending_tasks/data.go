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

package pending_tasks

import (
	"encoding/json"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v8/metricbeat/helper/elastic"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"insert_order":     c.Int("insert_order"),
		"priority":         c.Str("priority"),
		"source":           c.Str("source"),
		"time_in_queue.ms": c.Int("time_in_queue_millis"),
	}
)

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool) error {
	tasksStruct := struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}{}

	err := json.Unmarshal(content, &tasksStruct)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Pending Tasks API response")
	}

	if tasksStruct.Tasks == nil {
		return elastic.MakeErrorForMissingField("tasks", elastic.Elasticsearch)
	}

	var errs multierror.Errors
	for _, task := range tasksStruct.Tasks {
		event := mb.Event{}

		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", elasticsearch.ModuleName)

		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("cluster.name", info.ClusterName)
		event.ModuleFields.Put("cluster.id", info.ClusterID)

		event.MetricSetFields, err = schema.Apply(task)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure applying task schema"))
			continue
		}

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
