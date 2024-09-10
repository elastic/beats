// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package interfaces

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
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

const IFNetInterfaceQuery = "<show><interface>all</interface></show>"

func getIFNetInterfaceEvents(m *MetricSet) ([]mb.Event, error) {

	var response InterfaceResponse

	output, err := m.client.Op(IFNetInterfaceQuery, panw.Vsys, nil, nil)
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
	timestamp := time.Now()

	// First process the phyiscal interfaces
	for _, entry := range input.HW.Entries {
		iftype := interfaceTypes[entry.Type]

		var members []string
		// If this is an aggregate interface, populate the members
		if entry.Type == 1 {
			members = entry.AEMember.Members
		}

		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"physical.name":       entry.Name,
				"physical.id":         entry.ID,
				"physical.type":       iftype,
				"physical.mac":        entry.MAC,
				"physical.speed":      entry.Speed,
				"physical.duplex":     entry.Duplex,
				"physical.state":      entry.State,
				"physical.mode":       entry.Mode,
				"physical.full_state": entry.ST,
				"physical.ae_member":  members,
			},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
			},
		}

		events = append(events, event)
	}

	// Now process the logical interfaces
	for _, entry := range input.Ifnet.Entries {
		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"logical.name":     entry.Name,
				"logical.id":       entry.ID,
				"logical.tag":      entry.Tag,
				"logical.vsys":     entry.Vsys,
				"logical.zone":     entry.Zone,
				"logical.fwd":      entry.Fwd,
				"logical.ip":       entry.IP,
				"logical.addr":     entry.Addr,
				"logical.dyn_addr": entry.DynAddr,
				"logical.addr6":    entry.Addr6,
			},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
			}}

		events = append(events, event)

	}

	return events
}
