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
)

type response struct {
	ExecutingPolicies []map[string]interface{} `json:"executing_policies"`
	CoordinatorStats  []map[string]interface{} `json:"coordinator_stats"`
}

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {
	var data response
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Enrich Stats API response")
	}

	var errs multierror.Errors
	for _, stat := range data.CoordinatorStats {

		event := mb.Event{}
		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", elasticsearch.ModuleName)

		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("cluster.name", info.ClusterName)
		event.ModuleFields.Put("cluster.id", info.ClusterID)

		fields, err := schema.Apply(stat)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure applying enrich coordinator stats schema"))
			continue
		}

		nodeID, err := fields.GetValue("node_id")
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure retrieving node ID from Elasticsearch Enrich Stats API response"))
		}

		event.ModuleFields.Put("node.id", nodeID)
		fields.Delete("node_id")

		event.MetricSetFields = fields

		r.Event(event)
	}

	return errs.Err()
}
