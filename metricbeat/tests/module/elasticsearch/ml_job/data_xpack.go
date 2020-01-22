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

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch ML Job Stats API response")
	}

	jobs, ok := data["jobs"]
	if !ok {
		return elastic.MakeErrorForMissingField("jobs", elastic.Elasticsearch)
	}

	jobsArr, ok := jobs.([]interface{})
	if !ok {
		return fmt.Errorf("jobs is not an array of maps")
	}

	var errs multierror.Errors
	for _, j := range jobsArr {
		job, ok := j.(map[string]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("job is not a map"))
			continue
		}

		if err := elastic.FixTimestampField(job, "data_counts.earliest_record_timestamp"); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := elastic.FixTimestampField(job, "data_counts.latest_record_timestamp"); err != nil {
			errs = append(errs, err)
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

	return errs.Err()
}
