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

package pipeline

import (
	"encoding/json"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/logstash"
)

var (
	pipelineXPackSchema = s.Schema{
		"ephemeral_id": c.Str("ephemeral_id", s.Optional), // TODO: Remove optional once [1] is resolved
		"hash":         c.Str("hash", s.Optional),         // TODO: Remove optional once [1] is resolved
		"batch_size":   c.Int("batch_size"),
		"workers":      c.Int("workers"),
	}
)

// [1] https://github.com/elastic/logstash/issues/10119

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, content []byte) error {
	event := mb.Event{}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", logstash.ModuleName)

	var data pipelinesResponse
	err := json.Unmarshal(content, &data)
	if err != nil {
		event.Error = errors.Wrap(err, "failure parsing Logstash Pipelines API response")
		r.Event(event)
		return event.Error
	}

	var errs multierror.Errors
	for pipelineID, pipeline := range data.Pipelines {
		fields, err := pipelineXPackSchema.Apply(pipeline)
		if err != nil {
			event.Error = errors.Wrap(err, "failure applying pipeline schema")
			r.Event(event)
			errs = append(errs, event.Error)
		}

		fields.Put("id", pipelineID)

		// TODO: call individual pipeline API? and add representation field once [1] is resolved

		event.RootFields = common.MapStr{
			"timestamp":      common.Time(time.Now()),
			"interval_ms":    m.Module().Config().Period / time.Millisecond,
			"type":           "logstash_state",
			"logstash_state": fields,
		}

		event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Logstash)
		r.Event(event)
	}
	return errs.Err()
}
