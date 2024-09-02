// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tunnels

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
	metricsetName = "tunnels"
	vsys          = ""
	query         = "<show><vpn><tunnel></tunnel></vpn></show>"
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
	cfgwarn.Beta("The panos tunnels metricset is beta.")

	config := panos.Config{}
	logger := logp.NewLogger(base.FullyQualifiedName())

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	logger.Debugf("panos_licenses metricset config: %v", config)

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
	log.Infof("panos_tunnels.Fetch initialized client")

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
		}}
		event.Timestamp = currentTime
		event.RootFields = mapstr.M{
			"observer.ip":     m.config.HostIp,
			"host.ip":         m.config.HostIp,
			"observer.vendor": "Palo Alto",
			"observer.type":   "firewall",
		}

		events = append(events, event)
	}

	return events

}
