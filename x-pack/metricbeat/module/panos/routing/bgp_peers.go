// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package routing

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getBGPEvents(m *MetricSet) ([]mb.Event, error) {
	query := "<show><routing><protocol><bgp><peer></peer></bgp></protocol></routing></show>"
	var response BGPResponse

	// Initialize the client
	// if err := m.client.Initialize(); err != nil {
	// 	log.Error("Failed to initialize client: %s", err)
	// 	return err
	// }

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error calling API: %s", err)
		return nil, err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error unmarshalling: %s", err)
		return nil, err
	}

	events := formatBGPEvents(m, response.Result.Entries)

	return events, nil
}

func formatBGPEvents(m *MetricSet, entries []BGPEntry) []mb.Event {
	events := make([]mb.Event, 0, len(entries))
	currentTime := time.Now()

	for _, entry := range entries {
		event := mb.Event{MetricSetFields: mapstr.M{
			"bgp.peer_name":              entry.Peer,
			"bgp.virtual_router":         entry.Vr,
			"bgp.peer_group":             entry.PeerGroup,
			"bgp.peer_router_id":         entry.PeerRouterID,
			"bgp.remote_as_asn":          entry.RemoteAS,
			"bgp.status":                 entry.Status,
			"bgp.status_duration":        entry.StatusDuration,
			"bgp.password_set":           entry.PasswordSet,
			"bgp.passive":                entry.Passive,
			"bgp.ulti_hop_ttl":           entry.MultiHopTTL,
			"bgp.peer_address":           entry.PeerAddress,
			"bgp.local_address":          entry.LocalAddress,
			"bgp.reflector_client":       entry.ReflectorClient,
			"bgp.same_confederation":     entry.SameConfederation,
			"bgp.aggregate_confed_as":    entry.AggregateConfedAS,
			"bgp.peering_type":           entry.PeeringType,
			"bgp.connect_retry_interval": entry.ConnectRetryInterval,
			"bgp.open_delay":             entry.OpenDelay,
			"bgp.idle_hold":              entry.IdleHold,
			"bgp.prefix_limit":           entry.PrefixLimit,
			"bgp.holdtime":               entry.Holdtime,
			"bgp.holdtime_config":        entry.HoldtimeConfig,
			"bgp.keepalive":              entry.Keepalive,
			"bgp.keepalive_config":       entry.KeepaliveConfig,
			"bgp.msg_update_in":          entry.MsgUpdateIn,
			"bgp.msg_update_out":         entry.MsgUpdateOut,
			"bgp.msg_total_in":           entry.MsgTotalIn,
			"bgp.msg_total_out":          entry.MsgTotalOut,
			"bgp.last_update_age":        entry.LastUpdateAge,
			"bgp.last_error":             entry.LastError,
			"bgp.status_flap_counts":     entry.StatusFlapCounts,
			"bgp.established_counts":     entry.EstablishedCounts,
			"bgp.orf_entry_received":     entry.ORFEntryReceived,
			"bgp.nexthop_self":           entry.NexthopSelf,
			"bgp.nexthop_thirdparty":     entry.NexthopThirdparty,
			"bgp.nexthop_peer":           entry.NexthopPeer,
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
