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

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"id":    c.Str("job_id"),
		"state": c.Str("state"),
		"data_counts": c.Dict("data_counts", s.Schema{
			"processed_record_count": c.Int("processed_record_count"),
			"invalid_date_count":     c.Int("invalid_date_count"),
		}),
	}
)

type jobsStruct struct {
	Jobs []map[string]interface{} `json:"jobs"`
}

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {

	jobsData := &jobsStruct{}
	err := json.Unmarshal(content, jobsData)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch ML Job Stats API response")
	}

	var errs multierror.Errors
	for _, job := range jobsData.Jobs {

		event := mb.Event{}

		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", elasticsearch.ModuleName)

		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("cluster.name", info.ClusterName)
		event.ModuleFields.Put("cluster.id", info.ClusterID)

		event.MetricSetFields, err = schema.Apply(job)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure applying ml job schema"))
			continue
		}

		r.Event(event)
	}
	return errs.Err()
}
