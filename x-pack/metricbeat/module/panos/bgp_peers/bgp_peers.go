package bgp_peers

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panos"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/PaloAltoNetworks/pango"
)

const (
	metricsetName = "bgp_peers"
	vsys          = ""
	query         = "<show><routing><protocol><bgp><peer><virtual-router>default</virtual-router></peer></bgp></protocol></routing></show>"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(panos.ModuleName, metricsetName, New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config panos.Config
	logger *logp.Logger
	client *pango.Firewall
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The panos licenses metricset is beta.")

	config := panos.Config{}
	logger := logp.NewLogger(base.FullyQualifiedName())

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	logger.Debugf("panos_bgp_peers metricset config: %v", config)

	client := &pango.Firewall{Client: pango.Client{Hostname: config.HostIp, ApiKey: config.ApiKey}}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		logger:        logger,
		client:        client,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	log := m.Logger()
	var response Response

	// Initialize the client
	if err := m.client.Initialize(); err != nil {
		log.Error("Failed to initialize client: %s", err)
		return err
	}

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		log.Error("Error: %s", err)
		return err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		log.Error("Error: %s", err)
		return err
	}

	events := getEvents(m, response.Result.Entries)

	for _, event := range events {
		report.Event(event)
	}

	return nil
}

func getEvents(m *MetricSet, entries []Entry) []mb.Event {
	events := make([]mb.Event, 0, len(entries))
	currentTime := time.Now()

	for _, entry := range entries {
		event := mb.Event{MetricSetFields: mapstr.M{
			"peer_name":              entry.Peer,
			"virtual_router":         entry.Vr,
			"peer_group":             entry.PeerGroup,
			"peer_router_id":         entry.PeerRouterID,
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
		}}
		event.Timestamp = currentTime
		event.RootFields = mapstr.M{
			"observer.ip": m.config.HostIp,
			"host.ip":     m.config.HostIp,
		}

		events = append(events, event)
	}

	return events
}
