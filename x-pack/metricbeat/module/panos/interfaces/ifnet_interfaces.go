// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package interfaces

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// these types apply to phyiscal interfaces
var interfaceTypes = map[int]string{
	0:  "Ethernet interface",
	1:  "Aggregate Ethernet (AE) interface",
	2:  "High Availability (HA) interface",
	3:  "VLAN interface",
	5:  "Loopback interface",
	6:  "Tunnel interface",
	10: "SD-WAN interface",
}

func getIFNetInterfaceEvents(m *MetricSet) ([]mb.Event, error) {
	query := "<show><interface>all</interface></show>"
	var response InterfaceResponse

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	events := formatIFInterfaceEvents(m, response.Result)

	return events, nil

}

func formatIFInterfaceEvents(m *MetricSet, input InterfaceResult) []mb.Event {
	events := make([]mb.Event, 0, len(input.HW.Entries)+len(input.Ifnet.Entries))
	currentTime := time.Now()

	// First process the phyiscal interfaces
	for _, entry := range input.HW.Entries {
		iftype := interfaceTypes[entry.Type]

		var members []string
		// If this is an aggregate interface, populate the members
		if entry.Type == 1 {
			members = entry.AEMember.Members
		}

		event := mb.Event{MetricSetFields: mapstr.M{
			"name":      entry.Name,
			"id":        entry.ID,
			"type":      iftype,
			"mac":       entry.MAC,
			"speed":     entry.Speed,
			"duplex":    entry.Duplex,
			"state":     entry.State,
			"mode":      entry.Mode,
			"st":        entry.ST,
			"ae_member": members,
			"logical":   false,
		},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
				"@Timestamp":      currentTime,
			},
		}

		events = append(events, event)
	}

	// Now process the logical interfaces
	for _, entry := range input.Ifnet.Entries {
		event := mb.Event{MetricSetFields: mapstr.M{
			"name":     entry.Name,
			"id":       entry.ID,
			"tag":      entry.Tag,
			"vsys":     entry.Vsys,
			"zone":     entry.Zone,
			"fwd":      entry.Fwd,
			"ip":       entry.IP_CIDR,
			"addr":     entry.Addr,
			"dyn_addr": entry.DynAddr,
			"addr6":    entry.Addr6,
			"logical":  true,
		},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
				"@Timestamp":      currentTime,
			}}

		events = append(events, event)

	}

	return events
}
