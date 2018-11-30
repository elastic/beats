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
		"ec2": schema.Object{
			"cpu.total.pct":                mapstrstr.Float("cpu_utilization"),
			"cpu.credit_usage":             mapstrstr.Float("cpu_credit_usage"),
			"cpu.credit_balance":           mapstrstr.Float("cpu_credit_balance"),
			"cpu.surplus_credit_balance":   mapstrstr.Float("cpu_surplus_credit_balance"),
			"cpu.surplus_credits_charged":  mapstrstr.Float("cpu_surplus_credits_charged"),
			"network.packets_in":           mapstrstr.Float("network_packets_in"),
			"network.packets_out":          mapstrstr.Float("network_packets_out"),
			"network.in.bytes":             mapstrstr.Float("network_in"),
			"network.out.bytes":            mapstrstr.Float("network_out"),
			"disk.read.bytes":              mapstrstr.Float("disk_read_bytes"),
			"disk.write.bytes":             mapstrstr.Float("disk_write_bytes"),
			"disk.read_ops":                mapstrstr.Float("disk_read_ops"),
			"disk.write_ops":               mapstrstr.Float("disk_write_ops"),
			"status.check_failed":          mapstrstr.Float("status_check_failed"),
			"status.check_failed_system":   mapstrstr.Float("status_check_failed_system"),
			"status.check_failed_instance": mapstrstr.Float("status_check_failed_instance"),
		},
		"cloud": schema.Object{
			"provider":          mapstrstr.Str("provider"),
			"instance.id":       mapstrstr.Str("instance.id"),
			"machine.type":      mapstrstr.Str("machine.type"),
			"region":            mapstrstr.Str("region"),
			"availability_zone": mapstrstr.Str("availability_zone"),
			"image.id":          mapstrstr.Str("image.id"),
		},
	}
)
