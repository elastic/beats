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
			"bgp_peer_name":          entry.Peer,
			"virtual_router":         entry.Vr,
			"bgp_peer_group":         entry.PeerGroup,
			"bgp_peer_router_id":     entry.PeerRouterID,
			"remote_as_asn":          entry.RemoteAS,
			"status":                 entry.Status,
			"status_duration":        entry.StatusDuration,
			"password_set":           entry.PasswordSet,
			"passive":                entry.Passive,
			"multi_hop_ttl":          entry.MultiHopTTL,
			"peer_address":           entry.PeerAddress,
			"local_address":          entry.LocalAddress,
			"reflector_client":       entry.ReflectorClient,
			"same_confederation":     entry.SameConfederation,
			"aggregate_confed_as":    entry.AggregateConfedAS,
			"peering_type":           entry.PeeringType,
			"connect_retry_interval": entry.ConnectRetryInterval,
			"open_delay":             entry.OpenDelay,
			"idle_hold":              entry.IdleHold,
			"prefix_limit":           entry.PrefixLimit,
			"holdtime":               entry.Holdtime,
			"holdtime_config":        entry.HoldtimeConfig,
			"keepalive":              entry.Keepalive,
			"keepalive_config":       entry.KeepaliveConfig,
			"msg_update_in":          entry.MsgUpdateIn,
			"msg_update_out":         entry.MsgUpdateOut,
			"msg_total_in":           entry.MsgTotalIn,
			"msg_total_out":          entry.MsgTotalOut,
			"last_update_age":        entry.LastUpdateAge,
			"last_error":             entry.LastError,
			"status_flap_counts":     entry.StatusFlapCounts,
			"established_counts":     entry.EstablishedCounts,
			"orf_entry_received":     entry.ORFEntryReceived,
			"nexthop_self":           entry.NexthopSelf,
			"nexthop_thirdparty":     entry.NexthopThirdparty,
			"nexthop_peer":           entry.NexthopPeer,
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
