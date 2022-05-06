// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package replication

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/syncgateway"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
			MetricSetFields: mapstr.M{
				"id":      replID,
				"metrics": replData,
			},
		})
	}
}
