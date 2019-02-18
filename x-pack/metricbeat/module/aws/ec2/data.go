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
				"pct": c.Float("cpu.total.pct", s.Optional),
			},
			"credit_usage":            c.Float("cpu.credit_usage", s.Optional),
			"credit_balance":          c.Float("cpu.credit_balance", s.Optional),
			"surplus_credit_balance":  c.Float("cpu.surplus_credit_balance", s.Optional),
			"surplus_credits_charged": c.Float("cpu.surplus_credits_charged", s.Optional),
		},
		"diskio": s.Object{
			"read": s.Object{
				"bytes": c.Float("diskio.read.bytes", s.Optional),
				"count": c.Float("diskio.read.count", s.Optional),
			},
			"write": s.Object{
				"bytes": c.Float("diskio.write.bytes", s.Optional),
				"count": c.Float("diskio.write.count", s.Optional),
			},
		},
		"network": s.Object{
			"in": s.Object{
				"bytes":   c.Float("network.in.bytes", s.Optional),
				"packets": c.Float("network.in.packets", s.Optional),
			},
			"out": s.Object{
				"bytes":   c.Float("network.out.bytes", s.Optional),
				"packets": c.Float("network.out.packets", s.Optional),
			},
		},
		"status": s.Object{
			"check_failed":          c.Int("status.check_failed", s.Optional),
			"check_failed_instance": c.Int("status.check_failed_instance", s.Optional),
			"check_failed_system":   c.Int("status.check_failed_system", s.Optional),
		},
		"instance": s.Object{
			"image": s.Object{
				"id": c.Str("instance.image.id", s.Optional),
			},
			"state": s.Object{
				"name": c.Str("instance.state.name", s.Optional),
				"code": c.Int("instance.state.code", s.Optional),
			},
			"monitoring": s.Object{
				"state": c.Str("instance.monitoring.state", s.Optional),
			},
			"core": s.Object{
				"count": c.Int("instance.core.count", s.Optional),
			},
			"threads_per_core": c.Int("instance.threads_per_core", s.Optional),
			"public": s.Object{
				"ip":       c.Str("instance.public.ip", s.Optional),
				"dns_name": c.Str("instance.public.dns_name", s.Optional),
			},
			"private": s.Object{
				"ip":       c.Str("instance.private.ip", s.Optional),
				"dns_name": c.Str("instance.private.dns_name", s.Optional),
			},
		},
	}
)

var (
	schemaRootFields = s.Schema{
		"service": s.Object{
			"name": c.Str("service.name", s.Optional),
		},
		"cloud": s.Object{
			"provider":          c.Str("cloud.provider", s.Optional),
			"availability_zone": c.Str("cloud.availability_zone", s.Optional),
			"region":            c.Str("cloud.region", s.Optional),
			"instance": s.Object{
				"id":   c.Str("cloud.instance.id", s.Optional),
				"name": c.Str("cloud.instance.name", s.Optional),
			},
			"machine": s.Object{
				"type": c.Str("cloud.machine.type", s.Optional),
			},
		},
	}
)

func eventMapping(input map[string]interface{}, schema s.Schema) (common.MapStr, error) {
	return schema.Apply(input)
}
