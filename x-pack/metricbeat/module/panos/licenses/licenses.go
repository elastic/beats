// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenses

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

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Licenses []License `xml:"licenses>entry"`
}

type License struct {
	Feature     string `xml:"feature"`
	Description string `xml:"description"`
	Serial      string `xml:"serial"`
	Issued      string `xml:"issued"`
	Expires     string `xml:"expires"`
	Expired     string `xml:"expired"`
	AuthCode    string `xml:"authcode"`
}

const (
	metricsetName = "licenses"
	vsys          = "shared"
	query         = "<request><license><info></info></license></request>"
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
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	log := m.Logger()
	var response Response

	// Initialize the client
	if err := m.client.Initialize(); err != nil {
		log.Fatalf("Failed to initialize client: %s", err)
	}
	log.Infof("panos_licenses.Fetch initialized client")

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	events := getEvents(m, response.Result.Licenses)

	for _, event := range events {
		reporter.Event(event)
	}

	return nil
}

func getEvents(m *MetricSet, licenses []License) []mb.Event {
	events := make([]mb.Event, 0, len(licenses))

	currentTime := time.Now()

	for _, license := range licenses {
		event := mb.Event{MetricSetFields: mapstr.M{
			"license.feature":     license.Feature,
			"license.description": license.Description,
			"license.serial":      license.Serial,
			"license.issued":      license.Issued,
			"license.expires":     license.Expires,
			"license.expired":     license.Expired,
			"license.auth_code":   license.AuthCode,
			"observer.ip":         m.config.HostIp,
		}}
		event.Timestamp = currentTime
		events = append(events, event)
	}

	return events

}
