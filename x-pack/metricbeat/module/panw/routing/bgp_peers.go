// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package routing

import (
	"encoding/xml"
	"net"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const bgpPeersQuery = "<show><routing><protocol><bgp><peer></peer></bgp></protocol></routing></show>"

var bgpLogger *logp.Logger

func getBGPEvents(m *MetricSet) ([]mb.Event, error) {
	// Set logger so all the sub functions have access
	bgpLogger = m.logger
	var response BGPResponse

	output, err := m.client.Op(bgpPeersQuery, panw.Vsys, nil, nil)
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

// Convert yes/no strings to booleans. Convert any errors to false and log a warning
func convertEntryBooleanFields(entry BGPEntry) map[string]bool {

	fields := []struct {
		name  string
		value string
	}{
		{"bgp.password_set", entry.PasswordSet},
		{"bgp.passive", entry.Passive},
		{"bgp.same_confederation", entry.SameConfederation},
		{"bgp.aggregate_confed_as", entry.AggregateConfedAS},
		{"bgp.nexthop_self", entry.NexthopSelf},
		{"bgp.nexthop_thirdparty", entry.NexthopThirdparty},
		{"bgp.nexthop_peer", entry.NexthopPeer},
	}

	result := make(map[string]bool)
	for _, field := range fields {
		boolValue, err := panw.StringToBool(field.value)
		if err != nil {
			bgpLogger.Warnf("Error converting %s: %v", field.name, err)
			boolValue = false
		}
		result[field.name] = boolValue
	}

	return result
}

func formatBGPEvents(m *MetricSet, entries []BGPEntry) []mb.Event {
	events := make([]mb.Event, 0, len(entries))
	timestamp := time.Now()

	for _, entry := range entries {
		booleanFields := convertEntryBooleanFields(entry)
		peer_ip, peer_port, err := net.SplitHostPort(entry.PeerAddress)
		if err != nil {
			bgpLogger.Warnf("Error splitting peer address (%s): %v", entry.PeerAddress, err)
			peer_ip = entry.PeerAddress
		}
		local_ip, local_port, err := net.SplitHostPort(entry.LocalAddress)
		if err != nil {
			bgpLogger.Warnf("Error splitting local address (%s): %v", entry.LocalAddress, err)
			local_ip = entry.LocalAddress
		}

		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"bgp.peer_name":              entry.Peer,
				"bgp.virtual_router":         entry.Vr,
				"bgp.peer_group":             entry.PeerGroup,
				"bgp.peer_router_id":         entry.PeerRouterID,
				"bgp.remote_as_asn":          entry.RemoteAS,
				"bgp.status":                 entry.Status,
				"bgp.status_duration":        entry.StatusDuration,
				"bgp.password_set":           booleanFields["bgp.password_set"],
				"bgp.passive":                booleanFields["bgp.passive"],
				"bgp.multi_hop_ttl":          entry.MultiHopTTL,
				"bgp.peer_ip":                peer_ip,
				"bgp.peer_port":              peer_port,
				"bgp.local_ip":               local_ip,
				"bgp.local_port":             local_port,
				"bgp.reflector_client":       entry.ReflectorClient,
				"bgp.same_confederation":     booleanFields["bgp.same_confederation"],
				"bgp.aggregate_confed_as":    booleanFields["bgp.aggregate_confed_as"],
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
				"bgp.nexthop_self":           booleanFields["bgp.nexthop_self"],
				"bgp.nexthop_thirdparty":     booleanFields["bgp.nexthop_thirdparty"],
				"bgp.nexthop_peer":           booleanFields["bgp.nexthop_peer"],
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
