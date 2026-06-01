// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package interfaces

import (
	"encoding/xml"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	IPSecTunnelsQuery               = "<show><vpn><tunnel></tunnel></vpn></show>"
	maxConcurrentTunnelStateQueries = 5
)

func tunnelFlowQuery(tunnelID int) string {
	return fmt.Sprintf("<show><running><tunnel><flow><tunnel-id>%d</tunnel-id></flow></tunnel></running></show>", tunnelID)
}

func getTunnelState(m *MetricSet, tunnelID int) (string, error) {
	query := tunnelFlowQuery(tunnelID)
	output, err := m.client.Op(query, panw.Vsys, nil, nil)
	if err != nil {
		return "", fmt.Errorf("error querying tunnel flow for tunnel %d: %w", tunnelID, err)
	}

	var response TunnelFlowResponse
	err = xml.Unmarshal(output, &response)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling tunnel flow response for tunnel %d: %w", tunnelID, err)
	}
	if response.Status != "success" {
		return "", fmt.Errorf("tunnel flow query for tunnel %d returned status %q", tunnelID, response.Status)
	}

	if len(response.Result.IPSec.Entries) > 0 {
		return response.Result.IPSec.Entries[0].State, nil
	}

	return "", nil
}

func getIPSecTunnelEvents(m *MetricSet) ([]mb.Event, error) {
	var response TunnelsResponse

	output, err := m.client.Op(IPSecTunnelsQuery, panw.Vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, fmt.Errorf("error querying IPSec tunnels: %w", err)
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, fmt.Errorf("error unmarshaling IPSec tunnels response: %w", err)
	}
	if response.Status != "success" {
		return nil, fmt.Errorf("IPSec tunnels query returned status %q", response.Status)
	}

	type stateResult struct {
		index int
		id    int
		state string
		err   error
	}

	jobs := make(chan int)
	results := make(chan stateResult, len(response.Result.Entries))

	workers := maxConcurrentTunnelStateQueries
	if workers > len(response.Result.Entries) {
		workers = len(response.Result.Entries)
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				entry := response.Result.Entries[i]
				state, err := getTunnelState(m, entry.ID)
				results <- stateResult{index: i, id: entry.ID, state: state, err: err}
			}
		}()
	}

	for i, entry := range response.Result.Entries {
		if entry.State != "" {
			continue
		}
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	close(results)

	for result := range results {
		if result.err != nil {
			m.logger.Warnf("Failed to get state for tunnel %d: %s", result.id, result.err)
			continue
		}
		if result.state != "" {
			response.Result.Entries[result.index].State = result.state
		}
	}

	events := formatIPSecTunnelEvents(m, response.Result.Entries)

	return events, nil
}

func formatIPSecTunnelEvents(m *MetricSet, entries []TunnelsEntry) []mb.Event {
	if entries == nil {
		return nil
	}

	events := make([]mb.Event, 0, len(entries))
	timestamp := time.Now().UTC()

	for _, entry := range entries {
		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"ipsec_tunnel.id":         entry.ID,
				"ipsec_tunnel.state":      entry.State,
				"ipsec_tunnel.name":       entry.Name,
				"ipsec_tunnel.gw":         entry.GW,
				"ipsec_tunnel.TSi_ip":     entry.TSiIP,
				"ipsec_tunnel.TSi_prefix": entry.TSiPrefix,
				"ipsec_tunnel.TSi_proto":  entry.TSiProto,
				"ipsec_tunnel.TSi_port":   entry.TSiPort,
				"ipsec_tunnel.TSr_ip":     entry.TSrIP,
				"ipsec_tunnel.TSr_prefix": entry.TSrPrefix,
				"ipsec_tunnel.TSr_proto":  entry.TSrProto,
				"ipsec_tunnel.TSr_port":   entry.TSrPort,
				"ipsec_tunnel.proto":      entry.Proto,
				"ipsec_tunnel.mode":       entry.Mode,
				"ipsec_tunnel.dh":         entry.DH,
				"ipsec_tunnel.enc":        entry.Enc,
				"ipsec_tunnel.hash":       entry.Hash,
				"ipsec_tunnel.life.sec":   entry.Life,
				"ipsec_tunnel.kb":         entry.KB,
			},
			RootFields: panw.MakeRootFields(m.config.HostIp, m.hostname),
		}

		events = append(events, event)
	}

	return events
}
