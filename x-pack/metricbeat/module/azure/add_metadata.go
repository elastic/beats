// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func addHostMetadata(event *mb.Event, metricList common.MapStr) {
	hostFieldTable := map[string]string{
		"percentage_cpu.avg":      "host.cpu.pct",
		"network_in_total.total":  "host.network.in.bytes",
		"network_in.total":        "host.network.in.packets",
		"network_out_total.total": "host.network.out.bytes",
		"network_out.total":       "host.network.out.packets",
		"disk_read_bytes.total":   "host.disk.read.bytes",
		"disk_write_bytes.total":  "host.disk.write.bytes",
	}

	for metricName, hostName := range hostFieldTable {
		metricValue, err := metricList.GetValue(metricName)
		if err != nil {
			continue
		}

		if value, ok := metricValue.(float64); ok {
			if metricName == "percentage_cpu.avg" {
				value = value / 100
			}
			event.RootFields.Put(hostName, value)
		}
	}
}

func addCloudVMMetadata(event *mb.Event, resource Resource) {
	event.RootFields.Put("cloud.instance.name", resource.Name)
	event.RootFields.Put("host.name", resource.Name)
	if resource.Vm != (VmResource{}) {
		if resource.Vm.Id != "" {
			event.RootFields.Put("cloud.instance.id", resource.Vm.Id)
			event.RootFields.Put("host.id", resource.Vm.Id)
		}
		if resource.Vm.Size != "" {
			event.RootFields.Put("cloud.machine.type", resource.Vm.Size)
		}
	}
}
