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
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, content []byte) error {
	info, err := elasticsearch.GetInfo(m.HTTP, m.HTTP.GetURI())
	if err != nil {
		return err
	}

	var data map[string]interface{}
	err = json.Unmarshal(content, &data)
	if err != nil {
		return err
	}

	jobs, ok := data["jobs"]
	if !ok {
		return elastic.MakeErrorForMissingField("jobs", elastic.Elasticsearch)
	}

	jobsArr, ok := jobs.([]interface{})
	if !ok {
		return fmt.Errorf("jobs is not an array of objects")
	}

	for _, job := range jobsArr {
		job, ok = job.(map[string]interface{})
		if !ok {
			continue
		}

		event := mb.Event{}
		event.RootFields = common.MapStr{
			"cluster_uuid": info.ClusterID,
			"timestamp":    common.Time(time.Now()),
			"interval_ms":  m.Module().Config().Period / time.Millisecond,
			"type":         "job_stats",
			"job_stats":    job,
		}

		event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		r.Event(event)
	}

	return nil
}
