// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vpn

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getIPSecTunnelEvents(m *MetricSet) ([]mb.Event, error) {
	query := "<show><vpn><tunnel></tunnel></vpn></show>"
	var response TunnelsResponse

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

	events := getEvents(m, response.Result.Entries)

	return events, nil

}

func getEvents(m *MetricSet, entries []TunnelsEntry) []mb.Event {
	events := make([]mb.Event, 0, len(entries))

	currentTime := time.Now()

	for _, entry := range entries {
		event := mb.Event{MetricSetFields: mapstr.M{
			"id":         entry.ID,
			"name":       entry.Name,
			"gw":         entry.GW,
			"TSi_ip":     entry.TSiIP,
			"TSi_prefix": entry.TSiPrefix,
			"TSi_proto":  entry.TSiProto,
			"TSi_port":   entry.TSiPort,
			"TSr_ip":     entry.TSrIP,
			"TSr_prefix": entry.TSrPrefix,
			"TSr_proto":  entry.TSrProto,
			"TSr_port":   entry.TSrPort,
			"proto":      entry.Proto,
			"mode":       entry.Mode,
			"dh":         entry.DH,
			"enc":        entry.Enc,
			"hash":       entry.Hash,
			"life":       entry.Life,
			"kb":         entry.KB,
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