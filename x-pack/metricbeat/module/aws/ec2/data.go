// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schemaMetricSetFields = s.Schema{
		"cpu": s.Object{
			"total": s.Object{
				"pct": c.Float("CPUUtilization", s.Optional),
			},
			"credit_usage":            c.Float("CPUCreditUsage", s.Optional),
			"credit_balance":          c.Float("CPUCreditBalance", s.Optional),
			"surplus_credit_balance":  c.Float("CPUSurplusCreditBalance", s.Optional),
			"surplus_credits_charged": c.Float("CPUSurplusCreditsCharged", s.Optional),
		},
		"diskio": s.Object{
			"read": s.Object{
				"bytes": c.Float("DiskReadBytes", s.Optional),
				"count": c.Float("DiskReadOps", s.Optional),
			},
			"write": s.Object{
				"bytes": c.Float("DiskWriteBytes", s.Optional),
				"count": c.Float("DiskWriteOps", s.Optional),
			},
		},
		"network": s.Object{
			"in": s.Object{
				"bytes":   c.Float("NetworkIn", s.Optional),
				"packets": c.Float("NetworkPacketsIn", s.Optional),
			},
			"out": s.Object{
				"bytes":   c.Float("NetworkOut", s.Optional),
				"packets": c.Float("NetworkPacketsOut", s.Optional),
			},
		},
		"status": s.Object{
			"check_failed":          c.Int("StatusCheckFailed", s.Optional),
			"check_failed_instance": c.Int("StatusCheckFailed_Instance", s.Optional),
			"check_failed_system":   c.Int("StatusCheckFailed_System", s.Optional),
		},
	}
)

func eventMapping(input map[string]interface{}, schema s.Schema) (common.MapStr, error) {
	return schema.Apply(input)
}
