// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// addHostMetadata enriches the event with host metadata.
func addHostMetadata(event *mb.Event, metricList mapstr.M) {
	hostFieldTable := map[string]string{
		"percentage_cpu.avg":      "host.cpu.usage",
		"network_in_total.total":  "host.network.ingress.bytes",
		"network_in.total":        "host.network.ingress.packets",
		"network_out_total.total": "host.network.egress.bytes",
		"network_out.total":       "host.network.egress.packets",
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
			_, _ = event.RootFields.Put(hostName, value)
		}
	}
}

// addCloudVMMetadata enriches the event with cloud VM metadata.
func addCloudVMMetadata(event *mb.Event, vm VmResource, subscriptionId string) {
	if vm.Name != "" {
		_, _ = event.RootFields.Put("cloud.instance.name", vm.Name)
		_, _ = event.RootFields.Put("host.name", vm.Name)
	}

	if vm.Id != "" {
		_, _ = event.RootFields.Put("cloud.instance.id", vm.Id)
		_, _ = event.RootFields.Put("host.id", vm.Id)
	}

	if vm.Size != "" {
		_, _ = event.RootFields.Put("cloud.machine.type", vm.Size)
	}

	if subscriptionId != "" {
		_, _ = event.RootFields.Put("cloud.account.id", subscriptionId)
	}
}
