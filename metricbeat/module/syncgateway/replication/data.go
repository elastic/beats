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

package replication

import (
	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/syncgateway"
)

var replicationSchema = s.Schema{
	"docs": s.Object{
		"pushed": s.Object{
			"count":  c.Int("sgr_num_docs_pushed"),
			"failed": c.Int("sgr_num_docs_failed_to_push"),
		},
		"checked_sent": c.Int("sgr_docs_checked_sent"),
	},
	"attachment": s.Object{
		"transferred": s.Object{
			"bytes": c.Int("sgr_num_attachment_bytes_transferred"),
			"count": c.Int("sgr_num_attachments_transferred"),
		},
	},
}

func eventMapping(r mb.ReporterV2, content *syncgateway.SgResponse) {
	for replID, replData := range content.Syncgateway.PerReplication {
		replData, _ := replicationSchema.Apply(replData)
		r.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"id":      replID,
				"metrics": replData,
			},
		})
	}
}
