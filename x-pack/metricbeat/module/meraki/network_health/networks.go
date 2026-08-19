// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package network_health

import (
	"strconv"

	sdk "github.com/meraki/dashboard-api-go/v3/sdk"

	"github.com/elastic/beats/v9/metricbeat/mb"
	"github.com/elastic/beats/v9/x-pack/metricbeat/module/meraki"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// ID is the unique identifier for all networks
type ID string

// Network contains attributes, statuses and metrics for Meraki networks
type Network struct {
	id       ID
	name     string
	vpnPeers *[]sdk.ResponseItemApplianceGetOrganizationApplianceVpnStatsMerakiVpnpeers
}

func reportNetworkMetrics(reporter mb.ReporterV2, organizationID string, networks map[ID]*Network) {
	metrics := []mapstr.M{}
	for _, network := range networks {
		if network == nil || network.vpnPeers == nil {
			continue
		}

		metric := mapstr.M{
			"network.id":   network.id,
			"network.name": network.name,
		}

		peers := make([]mapstr.M, len(*network.vpnPeers))
		for i, peer := range *network.vpnPeers {
			peers[i] = vpnPeerToMapStr(&peer)
		}
		metric["network.vpn_peers"] = peers

		metrics = append(metrics, metric)
	}

	meraki.ReportMetricsForOrganization(reporter, organizationID, metrics)
}

func vpnPeerToMapStr(peer *sdk.ResponseItemApplianceGetOrganizationApplianceVpnStatsMerakiVpnpeers) mapstr.M {
	res := mapstr.M{
		"network_id":   peer.NetworkID,
		"network_name": peer.NetworkName,
	}

	if peer.UsageSummary != nil {
		if recv, err := strconv.Atoi(peer.UsageSummary.ReceivedInKilobytes); err == nil {
			res["usage_summary.received.bytes"] = recv * 1024
		}
		if sent, err := strconv.Atoi(peer.UsageSummary.SentInKilobytes); err == nil {
			res["usage_summary.sent.bytes"] = sent * 1024
		}

	}

	if peer.JitterSummaries != nil {
		jitterSummaries := make([]mapstr.M, len(*peer.JitterSummaries))
		for i, jitter := range *peer.JitterSummaries {
			jitterSummaries[i] = mapstr.M{
				"jitter_avg":      jitter.AvgJitter,
				"jitter_max":      jitter.MaxJitter,
				"jitter_min":      jitter.MinJitter,
				"receiver_uplink": jitter.ReceiverUplink,
				"sender_uplink":   jitter.SenderUplink,
			}
		}
		res["jitter_summaries"] = jitterSummaries
	}

	if peer.LatencySummaries != nil {
		latencySummaries := make([]mapstr.M, len(*peer.LatencySummaries))
		for i, latency := range *peer.LatencySummaries {
			latencySummaries[i] = mapstr.M{
				"latency_avg.ms":  latency.AvgLatencyMs,
				"latency_max.ms":  latency.MaxLatencyMs,
				"latency_min.ms":  latency.MinLatencyMs,
				"receiver_uplink": latency.ReceiverUplink,
				"sender_uplink":   latency.SenderUplink,
			}
		}
		res["latency_summaries"] = latencySummaries
	}

	if peer.LossPercentageSummaries != nil {
		lossPercentageSummaries := make([]mapstr.M, len(*peer.LossPercentageSummaries))
		for i, loss := range *peer.LossPercentageSummaries {
			lossPercentageSummaries[i] = mapstr.M{
				"loss_avg.pct":    loss.AvgLossPercentage,
				"loss_max.pct":    loss.MaxLossPercentage,
				"loss_min.pct":    loss.MinLossPercentage,
				"receiver_uplink": loss.ReceiverUplink,
				"sender_uplink":   loss.SenderUplink,
			}
		}
		res["loss_percentage_summaries"] = lossPercentageSummaries
	}

	if peer.MosSummaries != nil {
		mosSummaries := make([]mapstr.M, len(*peer.MosSummaries))
		for i, mos := range *peer.MosSummaries {
			mosSummaries[i] = mapstr.M{
				"mos_avg":         mos.AvgMos,
				"mos_max":         mos.MaxMos,
				"mos_min":         mos.MinMos,
				"receiver_uplink": mos.ReceiverUplink,
				"sender_uplink":   mos.SenderUplink,
			}
		}
		res["mos_summaries"] = mosSummaries
	}

	return res
}
