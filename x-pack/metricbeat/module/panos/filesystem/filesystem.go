// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filesystem

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/PaloAltoNetworks/pango"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panos"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	metricsetName = "filesystem"
	vsys          = ""
	query         = "<show><system><disk-space></disk-space></system></show>"
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
	log.Infof("panos_licenses.Fetch initialized client")

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

	filesystems := getFilesystems(response.Result.Data)
	events := getEvents(m, filesystems)

	for _, event := range events {
		report.Event(event)
	}
	return nil
}

func getFilesystems(input string) []Filesystem {
	lines := strings.Split(input, "\n")
	filesystems := make([]Filesystem, 0)

	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) == 6 {
			filesystem := Filesystem{
				Name:    fields[0],
				Size:    fields[1],
				Used:    fields[2],
				Avail:   fields[3],
				UsePerc: fields[4],
				Mounted: fields[5],
			}
			filesystems = append(filesystems, filesystem)
		}
	}
	return filesystems
}

func getEvents(m *MetricSet, filesystems []Filesystem) []mb.Event {
	events := make([]mb.Event, 0, len(filesystems))

	currentTime := time.Now()

	for _, filesystem := range filesystems {
		event := mb.Event{MetricSetFields: mapstr.M{
			"name":        filesystem.Name,
			"size":        filesystem.Size,
			"used":        filesystem.Used,
			"available":   filesystem.Avail,
			"use_percent": filesystem.UsePerc,
			"mounted":     filesystem.Mounted,
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
