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
			"cpu_utilization":              mapstrstr.Float("cpu_utilization"),
			"cpu_credit_usage":             mapstrstr.Float("cpu_credit_usage"),
			"cpu_credit_balance":           mapstrstr.Float("cpu_credit_balance"),
			"cpu_surplus_credit_balance":   mapstrstr.Float("cpu_surplus_credit_balance"),
			"cpu_surplus_credits_charged":  mapstrstr.Float("cpu_surplus_credits_charged"),
			"network_packets_in":           mapstrstr.Float("network_packets_in"),
			"network_packets_out":          mapstrstr.Float("network_packets_out"),
			"network_in":                   mapstrstr.Float("network_in"),
			"network_out":                  mapstrstr.Float("network_out"),
			"disk_read_bytes":              mapstrstr.Float("disk_read_bytes"),
			"disk_write_bytes":             mapstrstr.Float("disk_write_bytes"),
			"disk_read_ops":                mapstrstr.Float("disk_read_ops"),
			"disk_write_ops":               mapstrstr.Float("disk_write_ops"),
			"status_check_failed":          mapstrstr.Float("status_check_failed"),
			"status_check_failed_system":   mapstrstr.Float("status_check_failed_system"),
			"status_check_failed_instance": mapstrstr.Float("status_check_failed_instance"),
			"provider":                     mapstrstr.Str("provider"),
			"instance.id":                  mapstrstr.Str("instance.id"),
			"machine.type":                 mapstrstr.Str("machine.type"),
			"region":                       mapstrstr.Str("region"),
		},
	}
)
