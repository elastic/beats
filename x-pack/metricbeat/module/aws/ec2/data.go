// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schemaMetricSetFields = s.Schema{
		"cpu": s.Object{
			"total": s.Object{
				"pct": c.Float("CPUUtilization"),
			},
			"credit_usage":            c.Float("CPUCreditUsage"),
			"credit_balance":          c.Float("CPUCreditBalance"),
			"surplus_credit_balance":  c.Float("CPUSurplusCreditBalance"),
			"surplus_credits_charged": c.Float("CPUSurplusCreditsCharged"),
		},
		"diskio": s.Object{
			"read": s.Object{
				"bytes": c.Float("DiskReadBytes"),
				"count": c.Float("DiskReadOps"),
			},
			"write": s.Object{
				"bytes": c.Float("DiskWriteBytes"),
				"count": c.Float("DiskWriteOps"),
			},
		},
		"network": s.Object{
			"in": s.Object{
				"bytes":   c.Float("NetworkIn"),
				"packets": c.Float("NetworkPacketsIn"),
			},
			"out": s.Object{
				"bytes":   c.Float("NetworkOut"),
				"packets": c.Float("NetworkPacketsOut"),
			},
		},
		"status": s.Object{
			"check_failed":          c.Int("StatusCheckFailed"),
			"check_failed_instance": c.Int("StatusCheckFailed_Instance"),
			"check_failed_system":   c.Int("StatusCheckFailed_System"),
		},
	}
)
