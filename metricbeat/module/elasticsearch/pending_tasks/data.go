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

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"insert_order":     c.Int("insert_order"),
		"priority":         c.Str("priority"),
		"source":           c.Str("source"),
		"time_in_queue.ms": c.Int("time_in_queue_millis"),
	}
)

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {
	tasksStruct := struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}{}

	err := json.Unmarshal(content, &tasksStruct)
	if err != nil {
		err = errors.Wrap(err, "failure parsing Elasticsearch ML Job Stats API response")
		r.Error(err)
		return err
	}

	if tasksStruct.Tasks == nil {
		return elastic.ReportErrorForMissingField("tasks", elastic.Elasticsearch, r)
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
			event.Error = errors.Wrap(err, "failure applying task schema")
			errs = append(errs, event.Error)
		}

		r.Event(event)
	}

	return errs.Err()
}
