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

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"id":    c.Str("job_id"),
		"state": c.Str("state"),
		"data_counts": c.Dict("data_counts", s.Schema{
			"processed_record_count": c.Int("processed_record_count"),
			"record": s.Object{
				"earliest": s.Object{
					"ms": c.Int("earliest_record_timestamp"),
				},
				"latest": s.Object{
					"ms": c.Int("latest_record_timestamp"),
				},
				"input": s.Object{
					"count": c.Int("input_record_count"),
				},
			},
			"field": s.Object{
				"processed": s.Object{
					"count": c.Int("processed_field_count"),
				},
			},
			"input": s.Object{
				"bytes": c.Int("input_bytes"),
				"field": s.Object{
					"count": c.Int("input_field_count"),
				},
			},
			"missing_field": s.Object{
				"count": c.Int("missing_field_count"),
			},
			"out_of_order": s.Object{
				"timestamp": s.Object{
					"count": c.Int("out_of_order_timestamp_count"),
				},
			},
			"bucket": s.Object{
				"empty": s.Object{
					"count": c.Int("empty_bucket_count"),
				},
				"sparse": s.Object{
					"count": c.Int("sparse_bucket_count"),
				},
				"count": c.Int("bucket_count"),
			},
			"invalid_date_count": c.Int("invalid_date_count"),
			"last_data_time":     c.Int("last_data_time"),
		}),
		"model_size": c.Dict("model_size_stats", s.Schema{
			"result_type": c.Str("result_type"),
			"model": s.Object{
				"bytes": c.Int("model_bytes"),
			},
			"total": s.Object{
				"field": s.Object{
					"by": s.Object{
						"count": c.Int("total_by_field_count"),
					},
					"over": s.Object{
						"count": c.Int("total_over_field_count"),
					},
					"partition": s.Object{
						"count": c.Int("total_partition_field_count"),
					},
				},
			},
			"bucket_allocation_failures": s.Object{
				"count": c.Int("bucket_allocation_failures_count"),
			},
			"memory_status": c.Int("memory_status"),
			"log_time": s.Object{
				"ms": c.Int("log_time"),
			},
			"timestamp": s.Object{
				"ms": c.Int("timestamp"),
			},
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

		event.MetricSetFields, _ = schema.Apply(job)

		r.Event(event)
	}

	return errs.Err()
}
