// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/elastic/beats/libbeat/common/schema"
	"github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	s = schema.Schema{
		"instance": schema.Object{
			"instance_id": mapstrstr.Str("instance_id"),
		},
		"ec2": schema.Object{
			"cpu_utilization":    mapstrstr.Float("cpu_utilization"),
			"cpu_credit_usage":   mapstrstr.Float("cpu_credit_usage"),
			"cpu_credit_balance": mapstrstr.Float("cpu_credit_balance"),
		},
	}
)
